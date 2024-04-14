package app

import (
	"common/config"
	"common/logs"
	"context"
	"fmt"
	"gate/router"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序，启动grpc服务
func Run(ctx context.Context) error {
	//日志库
	logs.InitLog(config.Conf.AppName)

	go func() {
		//gin启动 注册路由
		r := router.RegisterRoute()
		if err := r.Run(fmt.Sprintf(":%d", config.Conf.HttpPort)); err != nil {
			logs.Fatal("gate gin run err:%v", err)
		}
	}()

	//遇到中断信号如中断，退出，终止，挂断
	c := make(chan os.Signal, 1)
	stop := func() {

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
