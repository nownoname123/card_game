package remote

import (
	"common/logs"
	"errors"
	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	serverId string
	conn     *nats.Conn
	readChan chan []byte
}

func NewNatsClient(serverId string, readChan chan []byte) *NatsClient {
	return &NatsClient{
		serverId: serverId,
		readChan: readChan,
	}
}
func (c *NatsClient) Run() error {
	url := "192.168.32.128:4222"
	//games.Conf.ServersConf.Nats.Url
	logs.Info("nats connect :%v", url)
	var err error
	c.conn, err = nats.Connect(url)
	logs.Info("c.conn = %v", c.conn.ConnectedUrl())
	if err != nil {
		logs.Error("nats connect error :%v", err)
		return err
	}

	//创建订阅
	go c.Sub()
	return nil
}
func (c *NatsClient) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}
func (c *NatsClient) SendMsg(dst string, data []byte) error {
	logs.Info("send msg is running.... server id is : %v", c.serverId)
	if c.conn != nil {
		logs.Info("send message to : %v", dst)
		err := c.conn.Publish(dst, data)
		if err != nil {
			logs.Error("Send message error :%v", err)
			return err
		}
	} else {
		logs.Error("conn is nil ")
		return errors.New("conn is nil")
	}

	return nil
}

func (c *NatsClient) Sub() {
	logs.Info("sub is running.... server id is : %v", c.serverId)
	_, err := c.conn.Subscribe(c.serverId, func(msg *nats.Msg) {
		//收到的其它nats client发送的消息
		logs.Info("发布：%v", msg.Data)
		c.readChan <- msg.Data
	})
	if err != nil {
		logs.Error("nats sub err : %v", err)
		return
	}
}
