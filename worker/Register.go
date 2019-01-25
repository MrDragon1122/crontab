package worker

import (
	"fmt"
	"github.com/MrDragon1122/crontab/common"
	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"net"
	"time"
)

// 注册节点到etcd /cron/workers/ip
type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIp string //本机Ip
}

var (
	G_register *Register
)

// 获取本机ip
func getLocalIp() (ipv4 string, err error) {
	// 获取所有网卡
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	// 取第一个非lo的网卡ip
	for _, addr := range addrs {
		// ipv4, ipv6
		if ipNet, isIpNet := addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // ipv4
				return
			}
		}
	}

	err = fmt.Errorf("no machine ip")
	return
}

func InitRegister() (err error) {
	// 初始化etcd配置
	config := clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     // etcd集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // 超时
	}

	// 建立etcd的连接
	client, err := clientv3.New(config)
	if err != nil {
		return
	}

	// 生成kv和lease
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)

	G_register = &Register{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	// 获取本地ip
	if G_register.localIp, err = getLocalIp(); err != nil {
		return
	}

	// 启动服务注册
	go G_register.keepOnline()

	return
}

// 注册到etcd /cron/workers/ip
func (register *Register) keepOnline() (err error) {
	// 注册路径
	rekey := common.JOB_WORKER_DIR + register.localIp

	for {
		// 注册租约
		grantResp, err := register.lease.Grant(context.Background(), 10)
		if err != nil {
			// 自动重试
			time.Sleep(1 * time.Second)
			continue
		}

		// 获取租约ID
		leaseID := grantResp.ID

		// 自动续租
		keepAliveChan, err := register.lease.KeepAlive(context.Background(), leaseID)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		_, err = register.kv.Put(ctx, rekey, "", clientv3.WithLease(leaseID))
		if err != nil {
			time.Sleep(1 * time.Second)
			cancel()
			continue
		}

		// 处理续租应答
		for {
			select {
			// 自动续租应答
			case keepResp := <-keepAliveChan:
				if keepResp == nil {
					break
				}
			}
		}

		time.Sleep(1 * time.Second)
		cancel()
	}
	return
}
