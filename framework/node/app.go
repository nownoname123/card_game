package node

import (
	"common/logs"
	"encoding/json"
	"framework/remote"
)

// App  就是nats的服务端，处理实际游戏的逻辑的业务
type App struct {
	remoteCli remote.Client
	readChan  chan []byte
	writeChan chan *remote.Msg
	handlers  LogicHandler
}

func Default() *App {
	return &App{
		readChan:  make(chan []byte, 1024),
		writeChan: make(chan *remote.Msg, 1024),
		handlers:  make(LogicHandler),
	}
}

func (a *App) Run(serverId string) error {
	a.remoteCli = remote.NewNatsClient(serverId, a.readChan)

	err := a.remoteCli.Run()
	if err != nil {
		logs.Error("app run error: %v", err)
		return err
	}
	logs.Info("hall in running")
	go a.ReadChanMsg()
	go a.WriteChanMsg()
	return nil

}

func (a *App) WriteChanMsg() {
	//将处理好的消息发送回去
	for {
		select {
		case msg := <-a.writeChan:
			sendMsg, _ := json.Marshal(msg)
			err := a.remoteCli.SendMsg(msg.Dst, sendMsg)
			if err != nil {
				logs.Error("app remote send message error : %v", err)
			}
		}
	}
}

func (a *App) ReadChanMsg() {
	//收到的是其它nats client发送的消息
	for {
		select {
		case msg := <-a.readChan:
			var remoteMsg remote.Msg
			err := json.Unmarshal(msg, &remoteMsg)
			session := remote.NewSession(a.remoteCli, &remoteMsg)
			session.SetData(remoteMsg.SessionData)
			if err != nil {
				logs.Error("json unmarshal error : %v", err)
			}
			//根据路由消息，发送给对应的handler进行处理
			route := remoteMsg.Router
			if handlerFunc := a.handlers[route]; handlerFunc != nil {
				result := handlerFunc(session, remoteMsg.Body.Data)
				message := remoteMsg.Body
				var body []byte
				if result != nil {
					body, _ = json.Marshal(result)
				}
				message.Data = body
				//得到结果了 发送给connector
				responseMsg := &remote.Msg{
					Src:  remoteMsg.Dst,
					Dst:  remoteMsg.Src,
					Body: message,
					Uid:  remoteMsg.Uid,
					Cid:  remoteMsg.Cid,
				}
				logs.Info("response msg is : %v", responseMsg.Body)
				a.writeChan <- responseMsg
			} else {
				logs.Error("not handler func of : %v", route)
			}
		}
	}
}

func (a *App) RegisterHandler(handler LogicHandler) {
	a.handlers = handler
}
func (a *App) Close() {
	if a.remoteCli != nil {
		err := a.remoteCli.Close()
		if err != nil {
			logs.Error("remote client close error : %v", err)
		}
	}
}
