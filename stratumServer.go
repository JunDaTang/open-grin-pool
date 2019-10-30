package main

// http rpc server
import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/logger"
)

type stratumServer struct {
	db   *database
	ln   net.Listener
	conf *config
}

type stratumRequest struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type stratumResponse struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Result  interface{}            `json:"result"`
	Error   map[string]interface{} `json:"error"`
}

type minerSession struct {
	user       string
	rig        string
	agent      string
	difficulty int64
	edgeBits   int
	ctx        context.Context
}

func (ms *minerSession) hasNotLoggedIn() bool {
	return ms.user == ""
}

func (ms *minerSession) handleMethod(res *stratumResponse, db *database) {
	switch res.Method {
	case "status":
		if ms.user == "" {
			logger.Warning("recv status detail before login")
			break
		}
		//result, _ := res.Result.(map[string]interface{})
		//db.setMinerAgentStatus(ms.user, ms.agent, ms.difficulty, result)

		break
	case "submit":
		if res.Error != nil {
			logger.Warning(ms.user, ms.rig, "'s share has err: ", res.Error)
			break
		}
		detail, ok := res.Result.(string)
		logger.Info(ms.user, ms.rig, " has submit a ", detail, " share")
		if ok {
			db.putShare(ms.user, ms.agent, ms.difficulty)
			db.recordShare(ms.user, ms.rig, ms.difficulty)
			if strings.Contains(detail, "block") {
				blockHash := strings.Trim(detail, "block - ")
				db.putBlockHash(blockHash)
				logger.Warning("block ", blockHash, " has been found by ", ms.user, ms.rig)
			}
		}
		break
	}
}

func callStatusPerInterval(ctx context.Context, nc *nodeClient) {
	statusReq := &stratumRequest{
		ID:      "0",
		JsonRpc: "2.0",
		Method:  "status",
		Params:  nil,
	}

	ch := time.Tick(10 * time.Second)
	enc := json.NewEncoder(nc.c)

	for {
		select {
		case <-ch:
			err := enc.Encode(statusReq)
			if err != nil {
				logger.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (ss *stratumServer) handleConn(conn net.Conn) {
	logger.Info("new conn from ", conn.RemoteAddr())
	session := &minerSession{
		difficulty: int64(ss.conf.Node.Diff),
		edgeBits:   int(ss.conf.StratumServer.EdgeBits),
	}
	defer conn.Close()
	var login string
	nc := initNodeStratumClient(ss.conf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go callStatusPerInterval(ctx, nc)

	go nc.registerHandler(ctx, func(sr json.RawMessage) {
		enc := json.NewEncoder(conn)
		err := enc.Encode(sr)
		if err != nil {
			logger.Error(err)
		}

		// internal record
		var res stratumResponse
		_ = json.Unmarshal(sr, &res) // suppress the err

		session.handleMethod(&res, ss.db)
	})
	defer nc.close()

	dec := json.NewDecoder(conn)
	for {
		var jsonRaw json.RawMessage
		var clientReq stratumRequest

		err := dec.Decode(&jsonRaw)
		if err != nil {
			opErr, ok := err.(*net.OpError)
			if ok {
				if opErr.Err.Error() == syscall.ECONNRESET.Error() {
					return
				}
			} else {
				logger.Error(err)
			}
		}

		if len(jsonRaw) == 0 {
			return
		}

		err = json.Unmarshal(jsonRaw, &clientReq)
		if err != nil {
			// logger.Error(err)
			continue
		}

		logger.Info(conn.RemoteAddr(), " sends a ", clientReq.Method, " request:", string(jsonRaw))

		switch clientReq.Method {
		case "login":
			login, _ = clientReq.Params["login"].(string)
			pass, _ := clientReq.Params["pass"].(string)
			agent, _ := clientReq.Params["agent"].(string)

			login = strings.TrimSpace(login)
			pass = strings.TrimSpace(pass)
			agent = strings.TrimSpace(agent)

			user := login
			rig := "0"
			parts := strings.Split(login, ".")
			c := len(parts)
			if c > 1 {
				user = strings.Join(parts[:(c-1)], ".")
				rig = parts[(c - 1)]
			}

			if agent == "" {
				agent = "NoNameMiner" + strconv.FormatInt(rand.Int63(), 10)
			}

			session.user = user
			session.rig = rig
			session.agent = agent
			logger.Info(session.user, "'s ", rig, agent, " has logged in")
			_ = nc.enc.Encode(jsonRaw)

		case "submit":
			target := fmt.Sprintf("edge_bits\":%d", session.edgeBits)
			if !strings.Contains(string(jsonRaw), target) {
				logger.Warning(session.user, session.rig, ": wrong edge_bits share.")
				_, _ = conn.Write([]byte(`{  
					"id":0,
					"jsonrpc":"2.0",
					"method":"submit",
					"error":{  
					   "code":-32700,
					   "message":"wrong edge_bits."
					}
				 }`))
			} else {
				_ = nc.enc.Encode(jsonRaw)
			}
		default:
			if session.hasNotLoggedIn() {
				logger.Warning(login, " has not logged in")
			}

			_ = nc.enc.Encode(jsonRaw)
		}
	}
}

func initStratumServer(db *database, conf *config) {
	ip := net.ParseIP(conf.StratumServer.Address)
	addr := &net.TCPAddr{
		IP:   ip,
		Port: conf.StratumServer.Port,
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Warning("listening on ", conf.StratumServer.Port)

	ss := &stratumServer{
		db:   db,
		ln:   ln,
		conf: conf,
	}

	//go ss.backupPerInterval()

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			logger.Error(err)
		}

		go ss.handleConn(conn)
	}
}
