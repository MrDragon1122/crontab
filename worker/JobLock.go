package worker

import (
	"github.com/MrDragon1122/crontab/common"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

// 分布式锁(TXN事务)
type JobLock struct {
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string             // 任务名
	cancelFunc context.CancelFunc // 取消续租
	leaseId    clientv3.LeaseID   // 租约ID
	isLocked   bool               // 是否上锁成功
}

// 初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	return &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
}

// 尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	// 1 创建租约（5s，防止节点down，无法释放锁）
	leaseGrantResp, err := jobLock.lease.Grant(context.Background(), 5)
	if err != nil {
		return
	}

	// context用于取消自动续租
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	// 租约ID
	leaseId := leaseGrantResp.ID

	// 2 自动续租
	leaseKeepAliveRespChan, err := jobLock.lease.KeepAlive(cancelCtx, leaseId)
	if err != nil {
		// 取消自动续租
		cancelFunc()

		// 释放租约
		jobLock.lease.Revoke(context.Background(), leaseId)
		return
	}

	// 处理续租应答的协程
	go func() {
		for {
			select {
			// 自动续租应答
			case keepResp := <-leaseKeepAliveRespChan:
				if keepResp == nil {
					return
				}
			}
		}
	}()

	// 3 创建事务txr
	txn := jobLock.kv.Txn(context.Background())

	// 锁路径
	lockKey := common.JOB_LOCK_DIR + jobLock.jobName

	// 4 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))

	// 提交事务
	txnResp, err := txn.Commit()
	if err != nil { // 彻底回滚
		// 取消自动续租
		cancelFunc()

		// 释放租约
		jobLock.lease.Revoke(context.Background(), leaseId)
		return
	}

	// 5 成功返回 失败释放租约
	if !txnResp.Succeeded {
		err = common.ERR_LOCK_ALREADY_REQUIRED

		// 取消自动续租
		cancelFunc()

		// 释放租约
		jobLock.lease.Revoke(context.Background(), leaseId)
		return
	}

	// 6 抢锁成功
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true

	return
}

// 释放锁
func (jobLock *JobLock) Unlock() {
	if jobLock.isLocked {
		// 取消程序自动续租的协程
		jobLock.cancelFunc()

		// 释放租约
		jobLock.lease.Revoke(context.Background(), jobLock.leaseId)
	}
	return
}
