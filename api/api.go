package api

import (
	"encoding/json"
	"fmt"
	"github.com/google/logger"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"open-grin-pool/config"
	"open-grin-pool/db"
	"strconv"
)

func RevenueHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	table := db.DBServer.GetLastDayRevenue()
	raw, _ = json.Marshal(table)

	_, err := w.Write(raw)
	if err != nil {
		logger.Error(err)
		return
	}
}

func SharesHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	table := db.DBServer.GetShares()
	raw, _ = json.Marshal(table)

	_, err := w.Write(raw)
	if err != nil {
		logger.Error(err)
		return
	}
}

func PoolHandler(w http.ResponseWriter, r *http.Request) {
	var blockBatch []string
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	blockBatch = db.DBServer.GetAllBlockHashes()

	req, _ := http.NewRequest("GET", "http://"+config.Cfg.Node.Address+":"+strconv.Itoa(config.Cfg.Node.APIPort)+"/v1/status", nil)
	req.SetBasicAuth(config.Cfg.Node.AuthUser, config.Cfg.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.Error(err)
		return
	}

	dec := json.NewDecoder(res.Body)
	var nodeStatus interface{}
	_ = dec.Decode(&nodeStatus)

	table := map[string]interface{}{
		"node_status":  nodeStatus,
		"mined_blocks": blockBatch,
	}
	raw, err := json.Marshal(table)
	if err != nil {
		logger.Error(err)
		return
	}

	_, err = w.Write(raw)
	if err != nil {
		logger.Error(err)
		return
	}
}

type registerPaymentMethodForm struct {
	Pass          string `json:"pass"`
	PaymentMethod string `json:"pm"`
}

func MinerHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte

	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	login := vars["miner_login"]

	switch r.Method {
	case "POST":
		rawBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error(err)
			return
		}
		var form registerPaymentMethodForm
		err = json.Unmarshal(rawBody, &form)
		if err != nil {
			logger.Error(err)
			return
		}

		if db.DBServer.VerifyMiner(login, form.Pass) == db.CorrectPassword {
			db.DBServer.UpdatePayment(login, form.PaymentMethod)
			raw = []byte("{'status':'ok'}")
		} else {
			raw = []byte("{'status':'failed'}")
		}

		break
	default: // GET
		var err error
		m := db.DBServer.GetMinerStatus(login)
		raw, err = json.Marshal(m)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	if _, err := w.Write(raw); err != nil {
		logger.Error(err)
	}
}

func InitAPIServer(address string, port int) {
	r := mux.NewRouter()
	r.HandleFunc("/pool", PoolHandler)
	r.HandleFunc("/miner/{miner_login}", MinerHandler)
	r.HandleFunc("/revenue", RevenueHandler)
	r.HandleFunc("/shares", SharesHandler)
	http.Handle("/", r)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil))
}
