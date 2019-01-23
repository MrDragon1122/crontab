package main

import (
	"flag"
	"github.com/MrDragon1122/crontab/worker"
	"os"
	"traefik/log"
)

var (
	confFile string // config路径
)

// 命令行参数初始化
func initArgs() {
	flag.StringVar(&confFile, "config", "./worker.json", "指定worker.json")
	flag.Parse()
}

func main() {
	// 初始化命令行参数
	initArgs()

	// 加载配置
	if err := worker.InitConfig(confFile); err != nil {
		log.Errorf("init config err: %v", err)
		os.Exit(1)
	}
	log.Info("init config success")

	// 初始化job mgr
	if err := worker.InitJobMgr(); err != nil {
		log.Errorf("init job mgr err: %v", err)
		os.Exit(2)
	}
	log.Info("init job mgr success")

	// 启动执行器
	if err := worker.InitExecutor(); err != nil {
		log.Errorf("init executor err: %v", err)
		os.Exit(3)
	}
	log.Info("init executor success")

	// 启动调度进程
	worker.InitScheduler()
	log.Infof("init scheduler success")

	// 启动监听进程
	log.Info("start worker job watch")
	worker.G_jobMgr.WatchJobs()

	// 阻塞
	select {}
}
