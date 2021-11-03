package main

import (
	"context"
	"encoding/json"
	"github.com/google/logger"
	"io"
	"net"
	"open-grin-pool/config"
)

type nodeClient struct {
	c   net.Conn
	enc *json.Encoder
	dec *json.Decoder
}

func initNodeStratumClient() *nodeClient {
	ip := net.ParseIP(config.Cfg.Node.Address)
	raddr := &net.TCPAddr{
		IP:   ip,
		Port: config.Cfg.Node.StratumPort,
	}
	conn, err := net.DialTCP("tcp4", nil, raddr)
	if err != nil {
		logger.Error(err)
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	return &nodeClient{
		c:   conn,
		enc: enc,
		dec: dec,
	}
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(sr json.RawMessage)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var sr json.RawMessage

			err := nc.dec.Decode(&sr)
			if err != nil {
				logger.Error(err)
				if err == io.EOF {
					if nc.reconnect() != nil {
						return
					}
				}
				continue
			}

			resp, err := sr.MarshalJSON()
			if err != nil {
				logger.Error(err)
				continue
			}

			logger.Info("Node returns a response: ", string(resp))
			go callback(sr)
		}
	}
}

func (nc *nodeClient) reconnect() error {
	ip := net.ParseIP(config.Cfg.Node.Address)
	raddr := &net.TCPAddr{
		IP:   ip,
		Port: config.Cfg.Node.StratumPort,
	}
	conn, err := net.DialTCP("tcp4", nil, raddr)
	if err != nil {
		logger.Error(err)
		return err
	}

	nc.c = conn
	nc.enc = json.NewEncoder(conn)
	nc.dec = json.NewDecoder(conn)

	return nil
}

func (nc *nodeClient) close() {
	if err := nc.c.Close(); err != nil {
		logger.Error(err)
	}
}
