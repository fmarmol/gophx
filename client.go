package gophx

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// DefaultOnConnect is a default callback when client is connected
func DefaultOnConnect() {
	log.Println("connected")
}

// DefaultOnConnectError ...
func DefaultOnConnectError(err error) {
	log.Printf("error connection: %v\n", err)
}

// Client ...
type Client struct {
	socket   *websocket.Conn
	channels map[Topic]*Channel
	config   *ClientConfig
}

// ClientConfig ...
type ClientConfig struct {
	URL            string
	OnConnect      func()
	OnConnectError func(error)
	Params         map[string]string
}

// NewClient ...
func NewClient(config *ClientConfig) (*Client, error) {
	if config.URL == "" {
		return nil, errors.New("url not valid")
	}
	if config.OnConnect == nil {
		config.OnConnect = DefaultOnConnect
	}
	if config.OnConnectError == nil {
		config.OnConnectError = DefaultOnConnectError
	}
	return &Client{config: config, channels: map[Topic]*Channel{}}, nil
}

// Connect ...
func (c *Client) Connect() error {
	listParams := []string{}
	for key, value := range c.config.Params {
		listParams = append(listParams, key+"="+value)
	}
	params := strings.Join(listParams, "&")
	url := fmt.Sprintf("%v?%v", c.config.URL, params)
	// TODO: add custom header
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})

	pingHandler := func(appData string) error {
		log.Println("ping:", appData)
		return nil
	}
	pongHandler := func(appData string) error {
		log.Println("pong:", appData)
		return nil
	}

	handler := func(code int, text string) error {
		log.Println("connection closing:", code, text)
		return nil
	}

	conn.SetCloseHandler(handler)
	conn.SetPingHandler(pingHandler)
	conn.SetPingHandler(pongHandler)
	if err != nil {
		c.config.OnConnectError(err)
		return err
	}
	c.config.OnConnect()
	go func() {
		var message Message
		for {
			if err := c.socket.ReadJSON(&message); err != nil {
				panic(err)
			}
			log.Printf("received: %+v\n", message)
			channel, ok := c.channels[message.Topic]
			if !ok {
				panic("oups")
			}
			if message.Event == REPLY {
				if message.Ref == channel.joinRef {
					v, ok := message.Payload.(map[string]interface{})
					if ok {
						switch v["status"] {
						case "error":
							channel.OnJoinError(message.Payload)
							delete(c.channels, message.Topic)
						case "ok":
							channel.OnJoin(message.Payload)
						default:
							panic("unknown")
						}
					} else {
						log.Println("could not manage response:", message)
					}
				}
			}
			if message.Event == MSG {
				channel.OnMessage(message.Payload)
			}
		}
	}()
	c.socket = conn
	return nil
}

// Counter ...
func Counter() func() int {
	i := 1
	return func() int {
		val := i
		i++
		return val
	}
}

// Channel ...
func (c *Client) Channel(topic string) (*Channel, error) {
	channel := &Channel{
		topic:   Topic(topic),
		nextRef: Counter(),
		socket:  c.socket,
	}
	c.channels[Topic(topic)] = channel
	if err := channel.join(nil); err != nil {
		return nil, err
	}
	return channel, nil
}
