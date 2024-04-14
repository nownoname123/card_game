package discovery

//Register grpc 服务注册到etcd
//原理：创建一个租约，grpc注册到etcd，绑定租约
//租约会过期，过了租约，etcd就会删除这个grpc的服务信息
//实现心跳：会完成续租，如果etcd没有，就新注册
import (
	"common/config"
	"common/logs"
	"context"
	"encoding/json"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Register struct {
	etcdCli     *clientv3.Client                        //etcd连接
	leaseId     clientv3.LeaseID                        //租约id
	DialTimeout int                                     //超时时间 秒
	ttl         int64                                   //租约时间 秒
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse // 心跳channel
	info        Server                                  //注册的服务信息
	closeCh     chan struct{}
}

func NewRegister() *Register {
	return &Register{
		DialTimeout: 3,
	}
}

// Close 传递关闭信号给关闭channel
func (r *Register) Close() {
	r.closeCh <- struct{}{}
}
func (r *Register) Register(conf config.EtcdConf) error {
	//注册信息
	info := Server{
		Name:    conf.Register.Name,
		Addr:    conf.Register.Addr,
		Weight:  conf.Register.Weight,
		Version: conf.Register.Version,
		Ttl:     conf.Register.Ttl,
	}
	//建立etcd的连接
	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   conf.Addrs,
		DialTimeout: time.Duration(conf.DialTimeout) * time.Second,
	})
	if err != nil {
		return err
	}
	r.info = info
	err = r.register()
	if err != nil {
		return err
	}
	r.closeCh = make(chan struct{})
	//放入协程中，根据心跳的结果做出对应的操作
	go r.watch()

	return nil

}

// createLease 创建租约
func (r *Register) createLease(ctx context.Context, ttl int64) error {
	grant, err := r.etcdCli.Grant(ctx, ttl)
	if err != nil {
		logs.Error("createLease failed,err:%v", err)
		return err
	}
	r.leaseId = grant.ID
	return nil
}

// bindLease 绑定租约
func (r *Register) bindLease(ctx context.Context, key string, value string) error {
	//pull操作
	_, err := r.etcdCli.Put(ctx, key, value, clientv3.WithLease(r.leaseId))
	if err != nil {
		logs.Error("etcd bindLease error:%v", err)
		return err
	}
	logs.Info("register service success key = %s", key)
	return nil
}

// KeepAlive 心跳检测
func (r *Register) KeepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	//心跳检测，要求是一个长连接，如果做了超时，长连接久断了,因此不设置超时
	//一直不停发消息，保持续租
	KeepAliveResponses, err := r.etcdCli.KeepAlive(context.Background(), r.leaseId)
	if err != nil {
		logs.Error("etcd bindLease error:%v", err)
		return KeepAliveResponses, err
	}
	return KeepAliveResponses, nil
}

func (r *Register) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()
	var err error
	//创建租约
	if err = r.createLease(ctx, r.info.Ttl); err != nil {
		return err
	}
	//心跳检测
	if r.keepAliveCh, err = r.KeepAlive(); err != nil {
		return err
	}

	data, err := json.Marshal(r.info)
	if err != nil {
		logs.Error("etcd register json marshal error:%v", err)
		return err
	}
	return r.bindLease(ctx, r.info.BuildRegisterKey(), string(data))
}
func (r *Register) unregister() error {
	_, err := r.etcdCli.Delete(context.Background(), r.info.BuildRegisterKey())

	return err
}

// watch 续约或者新注册  如果是close就做注销注册操作
func (r *Register) watch() {
	//租约到期了，检测是否需要自动注册
	ticker := time.NewTicker(time.Duration(r.info.Ttl) * time.Second)
	for {
		select {
		case <-r.closeCh:
			if err := r.unregister(); err != nil {
				logs.Error("close error in watch%v", err)
			}
			//租约撤销
			_, err := r.etcdCli.Revoke(context.Background(), r.leaseId)
			if err != nil {
				logs.Error("revoke lease error:%v", err)
			}
			if r.etcdCli != nil {
				err := r.etcdCli.Close()
				if err != nil {
					logs.Error("etcd close error in watch err : %v", err)
				}
			}
			logs.Info("unregister etcd... ")
		//心跳检测到了，续约（因为还要用）
		case res := <-r.keepAliveCh:
			logs.Info("alive : %v", res)
			// res == nil etcd重启了，相当于连接断开，需要重新注册
			if res != nil {
				if err := r.register(); err != nil {
					logs.Error("keep alive register error in watch%v", err)
				}
			}
		case <-ticker.C:
			//心跳检测没有返回，注册这个服务
			if r.keepAliveCh == nil {
				if err := r.register(); err != nil {
					logs.Error("register error in watch%v", err)
				}
			}
		}
	}
}
