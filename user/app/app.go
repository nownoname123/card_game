package app

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"core/repo"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user/internal/service"
	"user/pb"
)

// Run 启动程序，启动grpc服务
func Run(ctx context.Context) error {
	//日志库
	logs.InitLog(config.Conf.AppName)
	//etcd注册中心，将grpc的服务注册到etcd中，客户端访问的时候通过etcd获取grpc的地址
	register := discovery.NewRegister()

	//启用grpc服务端
	server := grpc.NewServer()

	//注册grpc service 需要数据库mongo redis

	//初始化 数据库管理
	manager := repo.New()

	go func() {
		lis, err := net.Listen("tcp", config.Conf.Grpc.Addr)
		if err != nil {
			logs.Fatal("user grpc sever listen err1 :", err)
		}

		err = register.Register(config.Conf.Etcd)
		if err != nil {
			logs.Fatal("user grpc register etcd  err:", err)
		}
		//注册grpc服务
		pb.RegisterUserServiceServer(server, service.NewAccountService(manager))
		//阻塞操作

		err = server.Serve(lis)
		if err != nil {
			logs.Fatal("user grpc sever run err:%v", err)
		}
	}()

	//遇到中断信号如中断，退出，终止，挂断
	c := make(chan os.Signal, 1)
	stop := func() {
		server.Stop()
		register.Close()
		manager.Close()
		time.Sleep(3 * time.Second) //给3秒时间 停止必要的服务
		logs.Info("stop app finish")
	}
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case <-ctx.Done():
			stop()
			return nil
		case s := <-c:
			logs.Info("get a signal %s\n", s.String())
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				stop()
				logs.Info("user grpc server exit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("hang up ! user app quit")
				return nil
			default:
				return nil
			}
		}
	}

}
