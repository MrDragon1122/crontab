package main

import (
	"flag"

	"github.com/MrDragon1122/crontab/master"
	"os"
	"traefik/log"
)

var (
	confFile string // 配置文件路径
)

// 解析命令行参数
func initArgs() {
	// master -config ./master.json
	// master -h
	flag.StringVar(&confFile, "config", "./master.json", "指定master.json") // 参数名称，默认值，说明
	flag.Parse()
}

func main() {
	// 初始化命令行参数
	initArgs()

	// 加载配置
	if err := master.InitConfig(confFile); err != nil {
		log.Errorf("init config error: %v", err)
		os.Exit(1)
	}
	log.Info("init config success")

	// 初始化日志管理器
	if err := master.InitLogMgr(); err != nil {
		log.Errorf("init log mgr error: %v", err)
		os.Exit(2)
	}
	log.Info("init log mgr success")

	// 初始化集群管理器
	if err := master.InitWorkerMgr(); err != nil {
		log.Errorf("init worker mgr error: %v", err)
		os.Exit(3)
	}
	log.Info("init worker mgr success")

	// 初始化任务管理器
	if err := master.InitJobMgr(); err != nil {
		log.Errorf("init job mgr error: %v", err)
		os.Exit(4)
	}
	log.Info("init job mgr success")

	// 启动Api Http请求
	if err := master.InitApiServer(); err != nil {
		log.Errorf("init api server error: %v", err)
		os.Exit(5)
	}
	log.Info("init api server success")

	// 阻塞
	select {}
}
