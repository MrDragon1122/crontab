package master

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	ApiPort         int `json:"apiPort"`
	ApiReadTimeout  int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`

	EtcdEndpoints   []string `json:"etcdEndpoints"`
	EtcdDialTimeout int      `json:"etcdDialTimeout"`
}

// 定义单例
var G_config *Config

func InitConfig(filename string) (err error) {
	// 把配置文件读进来
	var bytes []byte
	if bytes, err = ioutil.ReadFile(filename); err != nil {
		return
	}

	// json反序列化
	var conf Config
	if err = json.Unmarshal(bytes, &conf); err != nil {
		return
	}

	// 赋值单例
	G_config = &conf

	return
}
