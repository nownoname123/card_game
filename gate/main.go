package main

import (
	"common/config"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"gate/app"
	"log"
	"os"
	"time"
)

var (
	configFile = flag.String("config", "application.yml", "config file")
)

func main() {
	//1加载配置
	flag.Parse()
	config.InitConfig(*configFile)
	fmt.Println(config.Conf)
	//2启动监控 用协程启动，不阻塞
	go func() {
		err := metrics.Sever(fmt.Sprintf("0.0.0.0:%v", config.Conf.MetricPort))
		if err != nil {
			time.Sleep(10 * time.Minute)
			panic(err)
		}

	}()
	//3启动程序（启动grpc服务）
	err := app.Run(context.Background())
	if err != nil {
		log.Println("app run error:", err)
		os.Exit(-1)
	}
}
