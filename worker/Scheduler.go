package worker

import (
	"github.com/MrDragon1122/crontab/common"
	"time"
	"traefik/log"
)

type Scheduler struct {
	jobEventChan     chan *common.JobEvent               // etcd中任务事件队列
	jobPlanTable     map[string]*common.JobSchedulerPlan // 任务调度计划表 key:value = jobName:jobschedulerPlan
	jobExecuingTable map[string]*common.JobExecuteInfo   // 任务执行表
	jobResultChan    chan *common.JobExecuteResult
}

// 定义单例
var (
	G_scheduler *Scheduler
)

// 初始化调度器
func InitScheduler() {
	G_scheduler = &Scheduler{
		jobEventChan:     make(chan *common.JobEvent, 1000),
		jobPlanTable:     make(map[string]*common.JobSchedulerPlan),
		jobExecuingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:    make(chan *common.JobExecuteResult, 1000),
	}

	// 启动调度协程
	go G_scheduler.schedulerLoop()

	return
}

// 推送jobEvent
func (scheduler *Scheduler) PushJobEvent(event *common.JobEvent) {
	scheduler.jobEventChan <- event
}

// 处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		jobSchedulerPlan, err := common.BuildJobSchedulerPlan(jobEvent.Job)
		if err != nil {
			return // 直接忽略任务
		}

		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulerPlan
	case common.JOB_EVENT_DELETE:
		if _, ok := scheduler.jobPlanTable[jobEvent.Job.Name]; ok {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	}
}

// 重新计算任务调度状态,实现任务的准确调度
func (scheduler *Scheduler) TrySchedule() (schedulerAfter time.Duration) {
	now := time.Now()
	var nearTime *time.Time

	// 如果任务表为空
	if len(scheduler.jobPlanTable) == 0 {
		schedulerAfter = 1 * time.Second
		return
	}

	for _, jobPlan := range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//TODO:尝试执行任务
			scheduler.TryStartJob(*jobPlan)

			// 更新下次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}

		// 统计最近一个要过期的任务时间
		if nearTime == nil || nearTime.After(jobPlan.NextTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	schedulerAfter = (*nearTime).Sub(now)

	return
}

// 调度协程
func (scheduler *Scheduler) schedulerLoop() {
	// 检测所有的任务
	// 初始化一次
	schedulerAfter := scheduler.TrySchedule()

	// 调度的定时器
	schedulerTimer := time.NewTimer(schedulerAfter)

	for {
		select {
		case jobEvent := <-scheduler.jobEventChan:
			// 对内存中维护的任务列表做CRDU
			scheduler.handleJobEvent(jobEvent)
		case <-schedulerTimer.C: // 最近的任务到期了
		case jobResult := <-scheduler.jobResultChan:
			scheduler.handleJobResult(jobResult)
		}

		schedulerAfter = scheduler.TrySchedule()
		schedulerTimer.Reset(schedulerAfter)
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan common.JobSchedulerPlan) {
	// 调度和执行是2件事
	// 执行的任务可能运行很久，1分钟调度60次，但是只能一次，防止并发
	// 如果任务正在执行，则跳过本次调度
	if _, ok := scheduler.jobExecuingTable[jobPlan.Job.Name]; ok {
		log.Infof("%v no execute success，skip this execution", jobPlan.Job.Name)
		return
	}

	// 构建执行状态信息
	jobExecuteInfo := common.BuildJobExecuteInfo(&jobPlan)

	// 保存执行状态信息
	scheduler.jobExecuingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	log.Infof("do job：%v", jobExecuteInfo.Job.Name)
	G_executor.ExecuteJob(jobExecuteInfo)
}

// 回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}

// 处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	// 删除执行状态
	delete(scheduler.jobExecuingTable, result.ExecuteInfo.Job.Name)

	// 存储执行结果
	log.Infof("job execute success, output：%v, err: %v", string(result.Output), result.Err)
}
