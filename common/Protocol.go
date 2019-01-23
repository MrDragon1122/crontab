package common

import "encoding/json"

// 定时任务
type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cronExpr"` // cron表达式
}

// http接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
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
