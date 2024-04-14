package main

import (
	"common/config"
	"common/metrics"
	"connector/app"
	"context"
	"fmt"
	games "framework/game"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"
)

/*
cobra.Command 是一个结构体，代表一个命令，其各个属性含义如下：

Use 是命令的名称。

Short 代表当前命令的简短描述。

Long 表示当前命令的完整描述。

Run 属性是一个函数，当执行命令时会调用此函数。
*/
var rootCmd = &cobra.Command{
	Use:   "connector",
	Short: "connector 管理连接，session以及路由请求",
	Long:  `connector 管理连接，session以及路由请求`,
	Run: func(cmd *cobra.Command, args []string) {
	},
	PostRun: func(cmd *cobra.Command, args []string) {
	},
}
var (
	configFile    string
	gameConfigDir string
	serverId      string
)

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "application.yml", "app config yml file")
	rootCmd.Flags().StringVar(&gameConfigDir, "gameDir", "../config", "game config dir")
	rootCmd.Flags().StringVar(&serverId, "serverId", "", "app server id， required")
	_ = rootCmd.MarkFlagRequired("serverId")
}

func main() {
	//1加载配置

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	//flag.Parse()
	config.InitConfig(configFile)
	games.InitConfig(gameConfigDir)
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
	err := app.Run(context.Background(), serverId)
	if err != nil {
		log.Println("app run error:", err)
		os.Exit(-1)
	}
}
