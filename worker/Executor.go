package worker

import (
	"github.com/MrDragon1122/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

// 任务执行器
type Executor struct {
}

// 定义单例
var (
	G_executor *Executor
)

// 初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}

// 执行任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	// 实现真正的随机数
	rand.Seed(time.Now().UnixNano())
	go func() {
		//任务执行结果
		result := &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		// 随机睡眠 解决由于服务器时钟不一致导致的，分布式不均匀的问题
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		// 首先获取分布式锁
		jobLock := G_jobMgr.CreateJobLock(info.Job.Name)

		// 记录任务开始时间
		result.StartTime = time.Now()

		// 抢占分布式锁
		err := jobLock.TryLock()
		defer jobLock.Unlock()

		if err != nil { // 上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			// 上锁成功后，重置任务开始时间
			result.StartTime = time.Now()

			// 执行shell命令
			cmd := exec.CommandContext(info.CommandCtx, "/bin/bash", "-c", info.Job.Command)

			// 执行并捕获输出
			output, err := cmd.CombinedOutput()

			// 记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}

		// 任务执行完成后，把执行的结果返回给Scheduler，Scheduler从ExecutingTable中删除执行记录
		G_scheduler.PushJobResult(result)
	}()
}
