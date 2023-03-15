package mqtt

import (
	"Broadcast/player"
	"Broadcast/rpc"
	"Broadcast/utils"
	"errors"
	goMqtt "github.com/eclipse/paho.mqtt.golang"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BroadcastClient struct {
	controller *player.Controller
	client     goMqtt.Client
	locker     *sync.Mutex
}

func NewBroadcastClient(controller *player.Controller) *BroadcastClient {
	return &BroadcastClient{
		controller: controller,
	}
}

// StartListener 开始连接并监听MQTT消息
//
//	host MQTT服务地址和端口
//	userName 访问MQTT服务的用户名
//	password 访问MQTT服务的密码
//	clientId 连接MQTT服务的客户端ID
func (c *BroadcastClient) StartListener(host, userName, password, clientId string) {
	clientOptions := goMqtt.NewClientOptions().
		AddBroker(host).
		SetUsername(userName).
		SetPassword(password).
		SetClientID(clientId).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetKeepAlive(120 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetWriteTimeout(10 * time.Second).
		SetOnConnectHandler(func(client goMqtt.Client) {
			// 连接建立后的回调函数
			utils.Logger.Debug("Mqtt is connected!", "clientId", clientId)
			// 订阅主题
			for {
				token := client.Subscribe(TopicRpcRequest, QoS2, c.receiveRpcRequest)
				if err := token.Error(); err != nil {
					time.Sleep(10 * time.Second)
					continue
				}
				break
			}
		}).
		SetConnectionLostHandler(func(client goMqtt.Client, err error) {
			// 连接被关闭后的回调函数
			utils.Logger.Debug("Mqtt is disconnected!", "clientId", clientId, "reason", err.Error())
		})

	c.locker = &sync.Mutex{}
	c.client = goMqtt.NewClient(clientOptions)
	//go c.heartbeat(60)

	// 建立连接
	for {
		err := c.ensureConnected()
		if err != nil {
			utils.Logger.Errorf("connect mqtt err: %s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		break
	}
}

func (c *BroadcastClient) StopListener() {
	if c.client == nil {
		return
	}

	c.client.Unsubscribe(TopicRpcRequest)
	c.client.Disconnect(3000)
	c.client = nil
}

// 确保连接
func (c *BroadcastClient) ensureConnected() error {
	if !c.client.IsConnected() {
		c.locker.Lock()
		defer c.locker.Unlock()
		if !c.client.IsConnected() {
			if token := c.client.Connect(); token.Wait() && token.Error() != nil {
				return token.Error()
			}
		}
	}
	return nil
}

// Publish 用默认MQTT客户端对象发布消息
//
//	topic 主题
//	qos 传送质量
//	retained 是否保留信息
//	data 要发送的数据
func (c *BroadcastClient) Publish(topic string, qos byte, retained bool, data []byte) error {
	if c.client == nil {
		return errors.New("call StartListener first")
	}
	if err := c.ensureConnected(); err != nil {
		return err
	}

	token := c.client.Publish(topic, qos, retained, data)
	if err := token.Error(); err != nil {
		return err
	}

	// return false is the timeout occurred
	if !token.WaitTimeout(time.Second * 10) {
		return errors.New("mqtt publish wait timeout")
	}

	return nil
}

//// Publish 用指定MQTT客户端对象发布消息
////
////	c MQTT客户端对象
////	topic 主题
////	qos 传送质量
////	retained 是否保留信息
////	data 要发送的数据
//func Publish(client goMqtt.Client, topic string, qos byte, retained bool, data []byte) error {
//	if client == nil {
//		return errors.New("call StartListener first")
//	}
//}

// receiveRpcRequest 接收RPC请求主题消息的处理回调函数
//
//	goClient 接收到消息的客户端对象
//	topic 接收到消息的主题
//	msg 接收到的消息
func (c *BroadcastClient) receiveRpcRequest(client goMqtt.Client, message goMqtt.Message) {
	topic := message.Topic()
	msg := strings.Replace(string(message.Payload()), "'", "\"", -1)

	utils.Logger.Debugf("recv topic: %+v", topic)
	utils.Logger.Debugf("recv msg: %+v", msg)

	var requestId int
	var err error
	for i := len(topic) - 1; i > -1; i-- {
		if topic[i] == '/' {
			if requestId, err = strconv.Atoi(topic[i+1:]); err != nil {
				return
			}
			break
		}
	}

	var process *rpc.Process
	if process, err = rpc.NewProcess(msg); err == nil {
		if msg, err = process.Run(c.controller); err == nil {
			msg = BuildSuccessResponse(msg)
		} else {
			msg = BuildFailResponse(err.Error())
		}
	} else {
		msg = BuildFailResponse(err.Error())
	}
	if err = c.Publish(TopicRpcResponse+strconv.Itoa(requestId), QoS0, false, []byte(msg)); err != nil {
		utils.Logger.Errorf("send response err, %s", err.Error())
	}
}

//func (c *BroadcastClient) heartbeat(interval int) {
//	if c.client == nil {
//		return
//	}
//
//	tik := time.NewTicker(time.Duration(interval) * time.Second)
//	for {
//		select {
//		case <-tik.C:
//			if err := c.Publish(TopicTelemetry, QoS1, false, []byte("heartbeat")); err != nil {
//				utils.Logger.Errorf("发送心跳包出错，%s", err.Error())
//			}
//		}
//	}
//	tik.Stop()
//}
