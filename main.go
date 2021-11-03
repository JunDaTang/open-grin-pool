package main

import (
	"github.com/google/logger"
	"open-grin-pool/api"
	"open-grin-pool/config"
	"open-grin-pool/db"
	"open-grin-pool/util"
	"os"
	"sync"
)

func main() {
	// 解析配置文件
	config.ParseConfig("config.json")

	// 初始化日志文件
	lf, err := os.OpenFile(config.Cfg.Log.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Errorf("Failed to open log file: %v", err)
	}
	defer lf.Close()
	defer logger.Init("pool", config.Cfg.Log.Verbose, config.Cfg.Log.SystemLog, lf).Close()

	// 初始化数据库
	db.InitDB()

	var gw sync.WaitGroup
	gw.Add(1)
	// apiserver
	util.Gogogo(func() {
		defer gw.Done()
		api.InitAPIServer(config.Cfg.APIServer.Address, config.Cfg.APIServer.Port)
	})

	util.Gogogo(func() {
		defer gw.Done()
		initStratumServer()
	})

	util.Gogogo(func() {
		defer gw.Done()
		initPayer()
	})
	gw.Wait()
	logger.Error("pool exit.")
}
