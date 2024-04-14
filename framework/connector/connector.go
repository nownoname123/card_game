package connector

import (
	"common/logs"
	"fmt"
	games "framework/game"
	"framework/net"
	"framework/remote"
)

type Connector struct {
	isRunning bool
	wsManager *net.Manager
	handles   net.LogicHandler
	remoteCli remote.Client
}

func Default() *Connector {
	return &Connector{
		handles: make(net.LogicHandler),
	}
}
func (c *Connector) Run(serverId string) {
	if !c.isRunning {
		//启动websocket
		c.wsManager = net.NewManager()
		c.wsManager.ConnectorHandlers = c.handles

		//启动nats nats不存储消息
		c.remoteCli = remote.NewNatsClient(serverId, c.wsManager.RemoteReadChan)
		_ = c.remoteCli.Run()
		c.wsManager.RemoteCli = c.remoteCli
		c.Serve(serverId)
	}
}
func (c *Connector) Close() {
	if c.isRunning {
		//关闭websocket
		c.wsManager.Close()
	}
}
func (c *Connector) Serve(serverID string) {
	logs.Info("id:%v is running...", serverID)
	//需要一个ws地址 在游戏中会加载很多配置信息
	//游戏中的配置读取采用json的方式，需要读取json的配置文件
	c.wsManager.ServerId = serverID
	conf := games.Conf.GetConnector(serverID)
	if conf == nil {
		logs.Fatal("no connector config found")
		return
	}
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.ClientPort)
	logs.Info("get addr :%v", addr)
	c.wsManager.Run(addr)
}

func (c *Connector) RegisterHandler(handles net.LogicHandler) {
	c.handles = handles
}
