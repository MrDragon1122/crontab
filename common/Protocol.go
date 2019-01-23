package common

import (
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

// 定时任务
type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cronExpr"` // cron表达式
}

// 任务调度计划
type JobSchedulerPlan struct {
	Job      *Job
	Expr     *cronexpr.Expression // cron表达式
	NextTime time.Time            // 下次调度时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job      *Job
	PlanTime time.Time // 理论执行时间
	RealTime time.Time // 实际执行时间
}

// http接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // 事件类型save delete
	Job       *Job
}

// 任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo // 执行状态
	Output      []byte          // 脚本输出
	Err         error           // 脚本错误信息
	StartTime   time.Time       // 启动时间
	EndTime     time.Time       // 结束时间
}

// 应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	// 定义response
	response := Response{
		Errno: errno,
		Msg:   msg,
		Data:  data,
	}

	// json序列化
	resp, err = json.Marshal(response)
	return
}

// 反序列化job
func Unpack(value []byte) (ret *Job, err error) {
	var job Job
	if err = json.Unmarshal(value, &job); err != nil {
		return
	}

	ret = &job
	return
}

// 从etcd的key中提取任务名称
// /cron/jobs/job10抹掉/cron/jobs/
func ExtractJobName(jobKey string) (jobName string) {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// 任务变化事件有两种，1 更新任务 2 删除任务
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// 构造任务执行计划
func BuildJobSchedulerPlan(job *Job) (jobSchedulerPlan *JobSchedulerPlan, err error) {
	// 解析job的cron表达式
	expr, err := cronexpr.Parse(job.CronExpr)
	if err != nil {
		return
	}

	jobSchedulerPlan = &JobSchedulerPlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	return
}

// 构造任务执行状态
func BuildJobExecuteInfo(jobSchedulerPlan *JobSchedulerPlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulerPlan.Job,
		PlanTime: jobSchedulerPlan.NextTime,
		RealTime: time.Now(), // 真实调度时间
	}
	return
}
