package master

import (
	"encoding/json"
	"github.com/MrDragon1122/crontab/common"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"time"
)

type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

// 单例
var (
	G_jobMgr *JobMgr
)

func InitJobMgr() (err error) {
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

	// 赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

// 存储job
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob common.Job, err error) {
	// 把任务保存到/cron/jobs/任务名 -> json
	// 定义etcd的key ： value
	jobKey := common.JOB_SAVE_DIR + job.Name

	jobValue, err := json.Marshal(job)
	if err != nil {
		return
	}

	// 保存到etcd
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	putResponse, err := jobMgr.kv.Put(ctx, jobKey, string(jobValue), clientv3.WithPrevKV())
	if err != nil {
		return
	}

	// 返回旧job,首先判定是否有返回值（更新时返回）
	if putResponse.PrevKv != nil {
		err = json.Unmarshal(putResponse.PrevKv.Value, &oldJob)
	}

	return
}

// 删除job
func (jobMgr *JobMgr) DelJob(name string) (oldJobs []common.Job, err error) {
	// 构建key
	jobKey := common.JOB_SAVE_DIR + name

	// 调用kv执行删除操作
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	delResp, err := jobMgr.kv.Delete(ctx, jobKey, clientv3.WithPrevKV())
	if err != nil {
		return
	}

	// 返回旧的job 判定slice是否为空，使用len
	if len(delResp.PrevKvs) != 0 {
		for _, val := range delResp.PrevKvs {
			var job common.Job
			if e := json.Unmarshal(val.Value, &job); e != nil {
				continue
			}
			oldJobs = append(oldJobs, job)
		}
	}

	return
}

// 获取所有的任务
func (jobMgr *JobMgr) GetAllJob() (jobs []common.Job, err error) {
	// Jobkey前缀
	jobKey := common.JOB_SAVE_DIR

	// 获取所有的任务
	var getResp *clientv3.GetResponse
	if getResp, err = jobMgr.kv.Get(context.Background(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}

	// 必须初始化数组空间，针对于返回值是否有效的判定，只需要判定长度是否为0即可
	jobs = make([]common.Job, 0)

	// 获取所有任务
	for _, val := range getResp.Kvs {
		var job common.Job
		if err = json.Unmarshal(val.Value, &job); err != nil {
			return
		}
		jobs = append(jobs, job)
	}

	return
}

// 杀死任务
func (jobMgr *JobMgr) KillJob(name string) (err error) {
	// 构建etcd的key
	jobKey := common.JOB_KILLER_DIR + name

	// 让worker监控到一次put操作，利用租约设定killer的自动过期时间
	leaseResp, err := jobMgr.lease.Grant(context.Background(), 1)
	if err != nil {
		return
	}

	// 获取租约id
	leaseID := leaseResp.ID

	// kv put操作
	if _, err = jobMgr.kv.Put(context.Background(), jobKey, "", clientv3.WithLease(leaseID)); err != nil {
		return
	}

	return
}
