package master

import (
	"github.com/MrDragon1122/crontab/common"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"golang.org/x/net/context"
)

// mongodb存储相关
type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

// 定义单例
var (
	G_logMgr *LogMgr
)

// 初始化mongodb
func InitLogMgr() (err error) {
	client, err := mongo.Connect(context.Background(), G_config.MongodbUri)
	if err != nil {
		return
	}

	G_logMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}

	return
}

// 获取日志列表
func (logMgr *LogMgr) ListLog(name string, skip, limit *int64) (logArr []*common.JobLog, err error) {
	// 初始化logArr
	logArr = make([]*common.JobLog, 0)

	// 定义日志过滤条件
	filter := &common.JobLogFilter{JobName: name}

	// 按照任务开始时间倒排
	logSort := &common.SortLogByStartTime{SortOrder: -1}

	// 查询mongodb日志库
	cursor, err := logMgr.logCollection.Find(context.Background(), filter,
		&options.FindOptions{Sort: logSort,
			Skip:  skip,
			Limit: limit}) // 排序、翻页

	if err != nil {
		return
	}
	defer cursor.Close(context.Background())

	// 遍历游标
	for cursor.Next(context.Background()) {
		jobLog := &common.JobLog{}
		if err := cursor.Decode(jobLog); err != nil {
			continue
		}

		logArr = append(logArr, jobLog)
	}

	return
}
