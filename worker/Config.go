package worker

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	EtcdEndpoints   []string
	EtcdDialTimeout int
}

// 定义单例
var G_config *Config

func InitConfig(filename string) (err error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	// json反序列化
	var conf Config
	if err = json.Unmarshal(bytes, &conf); err != nil {
		return
	}

	// 初始化单例
	G_config = &conf

	return
}
