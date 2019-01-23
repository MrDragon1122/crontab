package common

// 目录相关常量
const (
	// 任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	// 任务强杀目录
	JOB_KILLER_DIR = "/cron/killer/"

	// 任务锁目录
	JOB_LOCK_DIR = "/cron/lock/"
)

// 任务事件常量
const (
	JOB_EVENT_SAVE   int = iota // 保存任务事件
	JOB_EVENT_DELETE            // 删除任务事件
	JOB_EVENT_KILLER
)
