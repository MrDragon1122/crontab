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

// 杀死任务 利用etcd的watch功能实现，通知机制
func handlerJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		jobName string
		bytes   []byte
	)
	// 获取表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	jobName = req.PostForm.Get("name")

	// 杀死任务
	if err = G_jobMgr.KillJob(jobName); err != nil {
		goto ERR
	}

	log.Infof("kill job %v success", jobName)

	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(bytes)
	}

	return

ERR:
	log.Error("handle kill job err: %v", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

	return
}

// 查询日志
func handlerJobLog(resp http.ResponseWriter, req *http.Request) {
	var (
		bytes []byte
	)

	// 解析表单
	if err := req.ParseForm(); err != nil {
		return
	}

	// 获取请求参数/job/log?name=job1&skip=0&limit=10  skip 从第几条开始 limit 限制条数
	name := req.Form.Get("name")
	skipParam := req.Form.Get("skip")
	limitParam := req.Form.Get("limit") // 注意表单的哈数

	// 转化为数字
	skip, err := strconv.ParseInt(skipParam, 10, 64)
	if err != nil {
		skip = 0
	}
	limit, err := strconv.ParseInt(limitParam, 10, 64)
	if err != nil {
		limit = 20
	}

	// 查询日志list
	logArr, err := G_logMgr.ListLog(name, &skip, &limit)
	if err != nil {
		goto ERR
	}

	// 构建成功信息
	log.Infof("job log %v success", name)
	if bytes, err = common.BuildResponse(0, "success", logArr); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	log.Error("handle job log err: %v", err)
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

	return
}

// 输出worker list
func handleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		bytes []byte
	)

	workerArr, err := G_workerMgr.ListWorkers()
	if err != nil {
		goto ERR
	}

	// 构建成功信息
	log.Info("get worker list success")
	if bytes, err = common.BuildResponse(0, "success", workerArr); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	log.Error("handle get worker list err: %v", err)
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
	mux.HandleFunc("/job/kill", handlerJobKill)
	mux.HandleFunc("/job/log", handlerJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	// 知识点：路由匹配时支持最大路由匹配原则

	// http支持静态路由文件
	// 静态文件目录  开发前端页面，调用后端的接口，实现前后端分离的操作
	staticDir := http.Dir(G_config.WebRoot)
	staticHandler := http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler)) // stripprefix函数的作用是对路由规则进行二次处理，去掉多余的路由规则 ./webroot/index.html

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
