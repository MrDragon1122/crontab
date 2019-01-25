package worker

import (
	"context"
	"github.com/MrDragon1122/crontab/common"
	"github.com/mongodb/mongo-go-driver/mongo"
	"time"
)

// MongoDB存储日志
type LogSink struct {
	client        *mongo.Client
	logCollection *mongo.Collection
	logChan       chan *common.JobLog
}

// 定义单例
var (
	G_logsink *LogSink
)

func InitLogSink() (err error) {
	// 建立连接
	client, err := mongo.Connect(context.Background(), G_config.MongodbUri)
	if err != nil {
		return
	}

	// 选择db和collection
	G_logsink = &LogSink{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
		logChan:       make(chan *common.JobLog, 5000),
	}

	go G_logsink.writeLoop()

	return
}

// 日志存储协程
func (logSink *LogSink) writeLoop() {
	var logBatch = &common.LogBatch{}

	// 设定5秒自动提交log，不论是否达到条数的阈值
	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case log := <-logSink.logChan:
			// 插入到日志批次
			logBatch.Logs = append(logBatch.Logs, log)

			// 如果批次到达一定数量，则进行存储操作
			if len(logBatch.Logs) >= 50 {
				// 发送日志
				logSink.SaveMongoDB(logBatch)

				logBatch.Logs = logBatch.Logs[:0]

				// 重置定时器,避免重复存储
				timer.Reset(1 * time.Second)
			}
		case <-timer.C:
			if len(logBatch.Logs) != 0 {
				logSink.SaveMongoDB(logBatch)
				logBatch.Logs = logBatch.Logs[:0]
			}

			timer.Reset(1 * time.Second)
		}
	}
}

// 批量写入日志
func (logSink *LogSink) SaveMongoDB(logBatch *common.LogBatch) {
	logSink.logCollection.InsertMany(context.Background(), logBatch.Logs)
}

// 发送日志
func (logSink *LogSink) Append(joblog *common.JobLog) {
	select {
	case logSink.logChan <- joblog:
	default:
		// 日志满了，就丢弃
	}
}
