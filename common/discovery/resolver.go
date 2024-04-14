package discovery

import (
	"common/config"
	"common/logs"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"time"
)

type Resolver struct {
	conf        config.EtcdConf
	etcdCli     *clientv3.Client //etcd 连接
	DialTimeout int              //超时时间
	closeCh     chan struct{}
	key         string
	cc          resolver.ClientConn
	srvAddrList []resolver.Address
	watchCh     clientv3.WatchChan
}

func NewResolver(conf config.EtcdConf) *Resolver {

	return &Resolver{
		conf:        conf,
		DialTimeout: conf.DialTimeout,
	}
}

// Build 当grpc.Dial的时候，就会同步调用此方法
func (r Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	//获取到调用的key (user/v1) 连接etcd 获取其value
	r.cc = cc
	//1.连接etcd
	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		logs.Fatal("grpc client connect etcd error: %v", err)
		return nil, err
	}

	//2根据key获取Value
	r.key = target.URL.Path
	err = r.sync()
	if err != nil {
		logs.Fatal("name: %s get value error in grpc :%v", r.key, err)
		return nil, err
	}
	//拿到value后关闭etcd连接
	r.closeCh = make(chan struct{})
	//若节点有变动了，想要实时更新信息
	//做一个监听的程序
	go r.watch()
	return nil, nil
}
func (r Resolver) Scheme() string {
	return "etcd"
}
func (r Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.DialTimeout)*time.Second)
	defer cancel()
	//前缀型查找 查找到user/v1/xxxxx:xxxxx
	res, err := r.etcdCli.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		logs.Error("sync error in get value %v", err)
		return err
	}
	for _, v := range res.Kvs {
		server, err := ParseValue(v.Value)
		if err != nil {
			logs.Error("grpc client parse etcd value failed name =%s err:", r.key, err)
			continue
		}
		//将获取到的信息告诉grpc
		r.srvAddrList = append(r.srvAddrList, resolver.Address{
			Addr:       server.Addr,
			Attributes: attributes.New("weight", server.Weight),
		})

	}
	if len(r.srvAddrList) == 0 {
		logs.Error("no service found")
		return nil
	}
	err = r.cc.UpdateState(resolver.State{
		Addresses: r.srvAddrList,
	})
	return err
}
func (r Resolver) watch() {
	//定时1分钟同步一次数据
	ticker := time.NewTicker(time.Minute)
	//监听节点的时间，从而触发不太的操作
	r.watchCh = r.etcdCli.Watch(context.Background(), r.key, clientv3.WithPrefix())
	//监听close事件，关闭etcd

	for {
		select {
		case <-r.closeCh:
			r.Close()

		case res, ok := <-r.watchCh:
			if ok {
				//根据不同的事件触发不同的操作
				r.update(res.Events)
			}
		case <-ticker.C:
			if err := r.sync(); err != nil {
				logs.Error("watch :: name: %s get value error in grpc :%c", r.key, err)

			}
		}
	}
}
func (r Resolver) update(events []*clientv3.Event) {
	for _, v := range events {
		switch v.Type {
		case clientv3.EventTypePut:
			//key value
			server, err := ParseValue(v.Kv.Value)
			if err != nil {
				logs.Error("grpc client update (event put) parse etcd value failed name =%s err:", r.key, err)
				continue
			}
			addr := resolver.Address{
				Addr:       server.Addr,
				Attributes: attributes.New("weight", server.Weight),
			}
			if !Exist(r.srvAddrList, addr) {
				r.srvAddrList = append(r.srvAddrList, addr)
				err = r.cc.UpdateState(resolver.State{
					Addresses: r.srvAddrList,
				})
				if err != nil {
					logs.Error("grpc client update (event put) fail in delete :%v", err)
				}
			}
		case clientv3.EventTypeDelete:
			//删除r.srvAddrList其中匹配的
			server, err := ParseKey(string(v.Kv.Key))
			if err != nil {
				logs.Error("grpc client update (event delete) parse etcd value failed name =%s err:", r.key, err)
				continue
			}
			addr := resolver.Address{Addr: server.Addr}
			//r.srvAddrList remove 操作 就是一个切片删除操作
			if list, ok := Remove(r.srvAddrList, addr); ok {
				r.srvAddrList = list
			}
			err = r.cc.UpdateState(resolver.State{
				Addresses: r.srvAddrList,
			})
			if err != nil {
				logs.Error("grpc client update (event delete) fail in delete :%v", err)
			}
		}
	}
}

func Remove(list []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range list {
		if list[i].Addr == addr.Addr {
			list[i] = list[len(list)-1]
			return list[:len(list)-1], true
		}
	}
	return nil, false
}
func Exist(list []resolver.Address, addr resolver.Address) bool {
	pd := false
	for _, v := range list {
		if v == addr {
			pd = true
			break
		}
	}
	return pd
}
func (r Resolver) Close() {
	if r.etcdCli != nil {
		err := r.etcdCli.Close()
		if err != nil {
			logs.Error("resolver close etcd err:%v", err)
		}
	}
}
