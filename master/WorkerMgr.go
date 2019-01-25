package master

import (
	"github.com/MrDragon1122/crontab/common"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"time"
)

// cron/workers
type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

// 定义单例
var (
	G_workerMgr *WorkerMgr
)

// 初始化
func InitWorkerMgr() (err error) {
	// 初始化配置
	config := clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     // 字符串数组
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // 超时
	}

	// 建立连接
	client, err := clientv3.New(config)
	if err != nil {
		return
	}

	// 得到KV和Lease
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)

	G_workerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

func (workerMgr *WorkerMgr) ListWorkers() (workerArr []string, err error) {
	// 初始化操作
	workerArr = make([]string, 0)

	// 获取wokers
	getResponse, err := workerMgr.kv.Get(context.Background(), common.JOB_WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		return
	}

	for _, val := range getResponse.Kvs {
		// key: /cron/woker/192.1.1.1
		workerIp := common.ExtractWorkerIp(string(val.Key))
		workerArr = append(workerArr, workerIp)
	}

	return
}
