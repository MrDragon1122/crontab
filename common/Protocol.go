package common

import (
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"golang.org/x/net/context"
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
	Job        *Job
	PlanTime   time.Time          // 理论执行时间
	RealTime   time.Time          // 实际执行时间
	CommandCtx context.Context    // 用于command的context
	CancelFunc context.CancelFunc // 用于取消command命令
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

// 任务执行日志结果
type JobLog struct {
	JobName      string `json:"jobName" bson:"jobName"`           // 任务名称
	Command      string `json:"command" bson:"command"`           // shell命令
	Output       string `json:"output" bson:"output"`             // 执行输出
	Err          string `json:"err" bson:"err"`                   // err输出
	PlanTime     int64  `json:"planTime" bson:"planTime"`         // 计划调度时间
	ScheduleTime int64  `json:"scheduleTime" bson:"scheduleTime"` // 开始调度时间
	StartTime    int64  `json:"startTime" bson:"startTime"`       // 命令执行开始时间
	EndTime      int64  `json:"endTime" bson:"endTime"`           // 命令执行结束时间
}

// 日志批次
type LogBatch struct {
	Logs []interface{}
}

// 日志过滤条件
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

// 任务日志排序规则
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` // {startTime:-1}
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

// 从etcd的key中提取任务名称
func ExtractKillerName(jobKey string) (jobName string) {
	return strings.TrimPrefix(jobKey, JOB_KILLER_DIR)
}

// 从etcd的key中提取worker ip
func ExtractWorkerIp(Key string) (workerIp string) {
	return strings.TrimPrefix(Key, JOB_WORKER_DIR)
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

	jobExecuteInfo.CommandCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.Background())

	return
}
