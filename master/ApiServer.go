package master

import (
	"encoding/json"
	"github.com/MrDragon1122/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
	"traefik/log"
)

// 任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

// 定义单例对象
var (
	G_apiServer *ApiServer
)

// 存储任务接口 (控制台调用)
// Post job = {"name":"job1", "command":"echo hello", "cronExpr":"* * * * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  common.Job
		bytes   []byte
	)

	// 1、解析post表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 2、取表单中的job字段
	postJob = req.PostForm.Get("job")

	// 3、反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	// 4、保存到etcd
	oldJob, err = G_jobMgr.SaveJob(&job)
	if err != nil {
		goto ERR
	}

	log.Infof("save job %v success", job)

	// 5、返回正常应答({"error":0, "msg":"", "data":{...}})
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}

	return

	// 6、返回异常应答
ERR:
	log.Errorf("handle job save err: %v", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

	return
}

// 删除任务接口
// job = {"name":job1}
func handleJobDel(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		jobName string
		oldJobs []common.Job
		bytes   []byte
	)

	// 获取表单数据 删除的job name
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	jobName = req.PostForm.Get("name")

	// 删除job
	if oldJobs, err = G_jobMgr.DelJob(jobName); err != nil {
		goto ERR
	}

	log.Infof("del job %v success", oldJobs)

	// 删除成功返回
	if bytes, err = common.BuildResponse(0, "success", oldJobs); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	log.Errorf("handle del job err: %v", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
	return
}

// 从etcd获取所有的任务
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	// 调用etcd接口查询所有的任务
	var (
		jobs  []common.Job
		err   error
		bytes []byte
	)

	// 获取任务
	if jobs, err = G_jobMgr.GetAllJob(); err != nil {
		goto ERR
	}

	log.Infof("get jobs list: %v", jobs)

	// 返回任务
	if bytes, err = common.BuildResponse(0, "success", jobs); err == nil {
		resp.Write(bytes)
	}

	return

ERR:
	log.Errorf("handle get jobs list err: %v", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
	return
}

// 初始化服务
func InitApiServer() (err error) {
	// 配置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDel)
	mux.HandleFunc("/job/list", handleJobList)

	// 启动TCP监听
	var listener net.Listener
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	httpServer := &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	// 启动http服务
	go httpServer.Serve(listener)

	return
}
