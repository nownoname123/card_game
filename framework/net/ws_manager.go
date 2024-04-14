package net

import (
	"common/logs"
	"common/utils"
	"encoding/json"
	"errors"
	"fmt"
	games "framework/game"
	"framework/protocol"
	"framework/remote"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	webSocketUpgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type CheckOriginHandel func(r *http.Request) bool
type EventHandler func(packet *protocol.Packet, c Connection) error

type HandlerFunc func(Session *Session, body []byte) (any, error)
type LogicHandler map[string]HandlerFunc

type Manager struct {
	CheckOriginHandel CheckOriginHandel
	// RWMutex 是一个 reader/writer 互斥锁，RWMutex 它不限制资源的并发读
	sync.RWMutex
	websocketUpgrade  *websocket.Upgrader
	ServerId          string
	clients           map[string]Connection
	ClientReadChan    chan *MsgPack
	handlers          map[protocol.PackageType]EventHandler
	ConnectorHandlers LogicHandler
	RemoteReadChan    chan []byte
	RemoteCli         remote.Client
	RemotePushChan    chan *remote.Msg
}

func NewManager() *Manager {
	return &Manager{
		ClientReadChan: make(chan *MsgPack, 1024),
		clients:        make(map[string]Connection),
		handlers:       make(map[protocol.PackageType]EventHandler),
		RemoteReadChan: make(chan []byte, 1024),
		RemotePushChan: make(chan *remote.Msg, 1024),
	}
}

func (m *Manager) Run(addr string) {
	go m.ClientReadChannelHandler()
	go m.RemoteReadChannelHandler()
	go m.RemotePushChannelHandler()
	http.HandleFunc("/", m.serverWs)
	//设置不同的消息处理器
	m.setupEventHandlers()
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logs.Fatal("connector listen serve err: %v", err)
	}
}
func (m *Manager) serverWs(w http.ResponseWriter, r *http.Request) {
	//websocket 基于http协议
	if m.websocketUpgrade == nil {
		m.websocketUpgrade = &webSocketUpgrade
	}
	Conn, err := m.websocketUpgrade.Upgrade(w, r, nil)
	if err != nil {
		logs.Error("upgrade websocket error : %v", err)
		return
	}
	client := NewWsConnection(Conn, m)
	m.addClient(client)
	client.Run()
}
func (m *Manager) addClient(client *WsConnection) {
	//添加客户端前先上锁
	m.Lock()
	defer m.Unlock()
	m.clients[client.Cid] = client

}

// ClientReadChannelHandler 读到的消息统一交给管理这个connection的manager处理
func (m *Manager) ClientReadChannelHandler() {
	for {
		select {
		case body, ok := <-m.ClientReadChan:
			if ok {
				m.decodeClientPack(body)
			} else {
				logs.Error("body :%v read error maybe channel close", body.Body)
			}
		}
	}
}

func (m *Manager) decodeClientPack(body *MsgPack) {
	//处理消息
	logs.Info("receiver message :%v", string(body.Body))
	//解析协议
	packet, err := protocol.Decode(body.Body)
	if err != nil {
		logs.Error("decode message error : %v", err)
		return
	}
	err = m.routeEven(packet, body.Cid)
	if err != nil {
		logs.Error("routeEven error(not route found) in decodeClientPack : %v", err)
		return
	}

}
func (m *Manager) routeEven(packet *protocol.Packet, cid string) error {
	//根据packetType来做不同的处理
	conn, ok := m.clients[cid]
	//如果这个连接本身没有被删除，还可以继续使用,如果不存在，就需要先创建连接在调用消息处理器
	if ok {
		handler, ok := m.handlers[packet.Type]
		if ok {
			return handler(packet, conn)
		} else {
			return errors.New("no packet type found")
		}
	} else {
		return errors.New("no client found")
	}

}
func (m *Manager) Close() {
	for cid, v := range m.clients {
		v.Close()
		delete(m.clients, cid)
	}
}

func (m *Manager) setupEventHandlers() {
	m.handlers[protocol.Handshake] = m.HandshakeHandler
	m.handlers[protocol.HandshakeAck] = m.HandshakeAckHandler
	m.handlers[protocol.Heartbeat] = m.HeartbeatHandler

	m.handlers[protocol.Data] = m.MessageHandler
	m.handlers[protocol.Kick] = m.KickHandler
}

func (m *Manager) HandshakeHandler(packet *protocol.Packet, c Connection) error {
	//处理握手消息，所有版本都支持，所以默认直接同意
	res := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Heartbeat: 3,
		},
	}
	var data, _ = json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("encode packet err:%v", err)
		return err
	}
	return c.SendMessage(buf)
}
func (m *Manager) HandshakeAckHandler(packet *protocol.Packet, c Connection) error {
	//接收到ack后从日志当中输出
	logs.Info("receiver handshake ack message...")
	return nil
}

func (m *Manager) HeartbeatHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver heartbeat message:%v", packet.Type)
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("encode packet err:%v", err)
		return err
	}
	return c.SendMessage(buf)
}

func (m *Manager) MessageHandler(packet *protocol.Packet, c Connection) error {
	//先解析message
	message := packet.MessageBody()

	logs.Info("receiver message body, type=%v, router=%v, data:%v",
		message.Type, message.Route, string(message.Data))

	//connector.entryHandler.entry
	routeStr := message.Route
	routers := strings.Split(routeStr, ".")
	if len(routers) != 3 {
		return errors.New("router unsupported")
	}

	serverType := routers[0]
	handlerMethod := fmt.Sprintf("%s.%s", routers[1], routers[2])
	connectorConfig := games.Conf.GetConnectorByServerType(serverType)

	if connectorConfig != nil {
		//本地connector服务器处理 返回session
		handler, ok := m.ConnectorHandlers[handlerMethod]
		if ok {
			data, err := handler(c.GetSession(), message.Data)
			if err != nil {
				return err
			}
			marshal, _ := json.Marshal(data)
			message.Type = protocol.Response
			message.Data = marshal
			encode, err := protocol.MessageEncode(message)
			if err != nil {
				return err
			}
			res, err := protocol.Encode(packet.Type, encode)
			if err != nil {
				return err
			}
			return c.SendMessage(res)
		}
	} else {
		//nats 远端调用处理 hall.userHandler.updateUserAddress

		//dst 返回一个服务器地址,通过serverType选择
		dst, err := m.selectDst(serverType)
		logs.Info("ws_manager start nats :%v", dst)
		if err != nil {
			logs.Error("remote send msg selectDst err:%v", err)
			return err
		}
		msg := &remote.Msg{
			Cid:         c.GetSession().Cid,
			Uid:         c.GetSession().Uid,
			Src:         m.ServerId,
			Dst:         dst,
			Router:      handlerMethod,
			Body:        message,
			SessionData: c.GetSession().data,
		}
		data, _ := json.Marshal(msg)
		err = m.RemoteCli.SendMsg(dst, data)
		if err != nil {
			logs.Error("remote send msg err：%v", err)
			return err
		}
	}
	return nil
}

func (m *Manager) KickHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver kick  message...")
	return nil
}

func (m *Manager) RemoteReadChannelHandler() {
	for {
		select {
		case body, ok := <-m.RemoteReadChan:
			logs.Info("sub nats msg: %v", string(body))
			if ok {
				var msg remote.Msg
				err := json.Unmarshal(body, &msg)

				if err != nil {
					logs.Error("nats remote message format error :%v", err)
					continue
				}
				if msg.Type == remote.SessionType {
					//需要特殊处理，session是存储在connection中的session，并不推送客户端
					logs.Info("msg.type: %v", msg.Type)
					m.setSessionData(msg)
					continue
				}
				if msg.Body != nil {
					if msg.Body.Type == protocol.Request || msg.Body.Type == protocol.Response {
						//给客户端回信息，都是response
						msg.Body.Type = protocol.Response
						m.Response(&msg)
					}
					if msg.Body.Type == protocol.Push {
						m.RemotePushChan <- &msg
					}
				}
			}
		}
	}
}

func (m *Manager) selectDst(serverType string) (string, error) {
	serversConfigs, ok := games.Conf.ServersConf.TypeServer[serverType]
	if !ok {
		return "nil", errors.New("no server")
	}
	//如果有，拿到多个，就随机一个(一个简单的服务器选择策略）
	rand.New(rand.NewSource(time.Now().UnixNano()))
	index := rand.Intn(len(serversConfigs))
	return serversConfigs[index].ID, nil
}

func (m *Manager) Response(msg *remote.Msg) {
	logs.Info("msg.cid == :%v", msg.Cid)
	connection, ok := m.clients[msg.Cid]
	if !ok {
		logs.Info("%s client down，uid=%s", msg.Cid, msg.Uid)
		return
	}
	buf, err := protocol.MessageEncode(msg.Body)
	if err != nil {
		logs.Error("Response MessageEncode err:%v", err)
		return
	}
	res, err := protocol.Encode(protocol.Data, buf)
	if err != nil {
		logs.Error("Response Encode err:%v", err)
		return
	}
	if msg.Body.Type == protocol.Push {
		//循环所有的clients，传递消息给对应的push user中
		for _, v := range m.clients {
			if utils.Contains(msg.PushUser, v.GetSession().Uid) {
				err := v.SendMessage(res)
				if err != nil {
					logs.Error("send message error in Response:%v", res)
				}
			}
		}
	} else {
		err := connection.SendMessage(res)
		if err != nil {
			logs.Error("send message error in Response 2:%v", res)
		}
	}
}

func (m *Manager) RemotePushChannelHandler() {
	for {
		select {
		case body, ok := <-m.RemotePushChan:
			if ok {
				if body.Body.Type == protocol.Push {
					m.Response(body)
				}
			}
		}
	}
}

func (m *Manager) setSessionData(msg remote.Msg) {
	m.RLock()
	defer m.RUnlock()

	connection, ok := m.clients[msg.Cid]
	if ok {

		connection.GetSession().SetData(msg.Uid, msg.SessionData)
	} else {
		logs.Error("no clients in set session data ")

	}
}
