package net

import (
	"common/logs"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"sync/atomic"
	"time"
)

var (
	pongWait              = 10 * time.Second
	writeWait             = 10 * time.Second
	pingInterval          = (pongWait * 9) / 10
	maxMessageSize int64  = 1024
	cidBase        uint64 = 10086
)

type WsConnection struct {
	Cid        string
	Conn       *websocket.Conn
	manager    *Manager
	ReadChan   chan *MsgPack
	WriteChan  chan []byte
	Session    *Session
	PingTicker *time.Ticker
}

func (c *WsConnection) GetSession() *Session {
	return c.Session
}

func (c *WsConnection) SendMessage(buf []byte) error {
	c.WriteChan <- buf
	return nil
}

func NewWsConnection(Conn *websocket.Conn, manager *Manager) *WsConnection {
	cid := fmt.Sprintf("%s-%s-%d", uuid.New().String(), manager.ServerId, atomic.AddUint64(&cidBase, 1))
	return &WsConnection{
		Conn:      Conn,
		manager:   manager,
		Cid:       cid,
		WriteChan: make(chan []byte, 1024),
		ReadChan:  manager.ClientReadChan,
		Session:   NewSession(cid),
	}
}
func (c *WsConnection) Run() {
	go c.readMessage()
	go c.writeMessage()
	//做一些心跳检测：websocket中的ping pong 机制
	c.Conn.SetPongHandler(c.PongHandler)
}
func (c *WsConnection) readMessage() {
	defer func() {
		//读完就移走这个客户端
		c.manager.removeClient(c)
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logs.Error("SetReadDeadline err:%v", err)
		return
	}
	//读要不停的读，（只接收二进制消息，不接受文本消息）
	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		//客户端发来的消息是二进制消息，存放到channel中
		if messageType == websocket.BinaryMessage {
			if c.ReadChan != nil {
				c.ReadChan <- &MsgPack{
					Cid:  c.Cid,
					Body: message,
				}
			}
		} else {
			logs.Error("unsupported message type : %d", messageType)
		}
	}
}
func (c *WsConnection) writeMessage() {
	//定时发送ping
	c.PingTicker = time.NewTicker(pingInterval)
	//将write channel中的数据输出
	for {
		select {
		case message, ok := <-c.WriteChan:
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					logs.Error("connection closed or error :%v", err)
				}
				return
			}
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				logs.Error("client[%s] write message error: %v", c.Cid, err)
			}
		case <-c.PingTicker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logs.Error("client[%s] ping message error: %v", c.Cid, err)
				c.Close()
			} else {
				logs.Info("Ping...")
			}

		}
	}
}

func (m *Manager) removeClient(wc *WsConnection) {
	for cid, c := range m.clients {
		if cid == wc.Cid {
			c.Close()
			delete(m.clients, cid)
		}
	}
}
func (c *WsConnection) Close() {
	if c.PingTicker != nil {
		c.PingTicker.Stop()
	}
	if c.Conn != nil {
		err := c.Conn.Close()
		if err != nil {
			logs.Error("conn : %v close error: %v", c.Cid, err)
			return
		}
	}

}

func (c *WsConnection) PongHandler(data string) error {

	logs.Info("cid: %d,receive a pong", c.Cid)
	err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return err
	}
	return nil
}
