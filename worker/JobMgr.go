package worker

import (
	"github.com/MrDragon1122/crontab/common"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
	"time"
	"traefik/log"
)

// 定义结构体
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

// 定义单例
var G_jobMgr *JobMgr

// 初始化
func InitJobMgr() (err error) {
	// 初始化etcd配置
	config := clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     // etcd集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // 超时
	}

	// 建立etcd的连接
	client, err := clientv3.New(config)
	if err != nil {
		return
	}

	// 生成kv和lease
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)
	watcher := clientv3.NewWatcher(client)

	// 初始化单例
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	return
}

// 监听任务的变化
func (jobMgr *JobMgr) WatchJobs() (err error) {
	// 1、get一下/cron/jobs/目录下的所有任务， 并且获知当前集群的version
	getResponse, err := jobMgr.kv.Get(context.Background(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return
	}

	// 遍历输出所有的任务
	for _, val := range getResponse.Kvs {
		var job *common.Job
		if job, err = common.Unpack(val.Value); err == nil {
			jobEvent := common.BuildJobEvent(common.JOB_EVENT_SAVE, job)

			// 把任务同步给调度协程scheduler
			G_scheduler.PushJobEvent(jobEvent)
		}
	}

	// 2、从该version监听变化事件
	go func() {
		// 从get时刻的后续版本开始监听变化
		watchStartRevison := getResponse.Header.Revision + 1 // 监听当前版本的下一个

		// 启动监听,/cron/jobs/目录的后续变化
		WatchChan := G_jobMgr.watcher.Watch(context.Background(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevison), clientv3.WithPrefix())

		// 处理监听事件 判定事件类型
		var jobEvent *common.JobEvent
		for watchResp := range WatchChan {
			for _, watchEvent := range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					job, err := common.Unpack(watchEvent.Kv.Value)
					if err != nil {
						log.Errorf("watch func unpackjob err: %v", job)
						continue
					}
					// 构建一个更新Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE: // 任务删除
					// get job name
					jobName := common.ExtractJobName(string(watchEvent.Kv.Key))

					// 构建一个删除Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, &common.Job{Name: jobName})

				}

				// 推送给scheduler
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return
}
