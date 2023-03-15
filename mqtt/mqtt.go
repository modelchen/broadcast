package mqtt

import (
	"errors"
	"fmt"
	goMqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	nativeClient  goMqtt.Client
	clientOptions *goMqtt.ClientOptions
	// 消息收到之后处理函数
	observer func(c *Client, topic, msg string)
}

// Subscribe 订阅消息
func (client *Client) Subscribe(observer func(c *Client, topic, msg string), qos byte, topics ...string) error {
	if len(topics) == 0 {
		return errors.New("the topic is empty")
	}

	if observer == nil {
		return errors.New("the observer func is nil")
	}

	if client.observer != nil {
		return errors.New("an existing observer subscribed on this client, you must unsubscribe it before you subscribe a new observer")
	}
	client.observer = observer

	filters := make(map[string]byte)
	for _, topic := range topics {
		filters[topic] = qos
	}
	client.nativeClient.SubscribeMultiple(filters, client.messageHandler)

	return nil
}

func (client *Client) messageHandler(c goMqtt.Client, msg goMqtt.Message) {
	if client.observer == nil {
		fmt.Println("not subscribe message observer")
		return
	}
	//message, err := decodeMessage(msg.Payload())
	message := string(msg.Payload())
	//if err != nil {
	//	fmt.Println("failed to decode message")
	//	return
	//}
	client.observer(client, msg.Topic(), message)
}

//func decodeMessage(payload []byte) (*Message, error) {
//	message := new(Message)
//	decoder := json.NewDecoder(strings.NewReader(string(payload)))
//	decoder.UseNumber()
//	if err := decoder.Decode(&message); err != nil {
//		return nil, err
//	}
//	return message, nil
//}

func (client *Client) Unsubscribe(topics ...string) {
	client.observer = nil
	client.nativeClient.Unsubscribe(topics...)
}

func BuildSuccessResponse(data string) string {
	if data == "" {
		data = "{}"
	}
	return fmt.Sprintf(ResponseRpcFmt, ResultTrue, "", data)
}

func BuildFailResponse(err string) string {
	return fmt.Sprintf(ResponseRpcFmt, ResultFalse, err, "{}")
}
