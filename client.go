package gophx

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

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

// Socket wrapper
type Socket struct {
	*websocket.Conn
	sync.Mutex
}

func (s *Socket) writeJSON(v interface{}) error {
	s.Lock()
	defer s.Unlock()
	return s.WriteJSON(v)
}

// HeartBeat ...
func (s *Socket) HeartBeat(payload interface{}, each time.Duration) {
	go func() {
		for {
			if err := s.writeJSON(payload); err != nil {
				log.Println("error heartbeat:", err)
			}
			time.Sleep(each)
		}
	}()
}

// Client ...
type Client struct {
	socket   *Socket
	channels map[Topic]*Channel
	config   *ClientConfig
	sync.Mutex
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
	if err != nil {
		return err
	}
	c.socket = &Socket{Conn: conn}

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

	// HEARTBEAT PART
	// non blocking call
	c.socket.HeartBeat(Message{Topic: Topic("phoenix"), Event: HEARTBEAT, Payload: "OK"}, 10*time.Second)
	go func() {
		var message Message
		for {
			if err := c.socket.ReadJSON(&message); err != nil {
				panic(err)
			}
			// phoenix topic is for heartbeat
			if message.Topic == "phoenix" {
				continue
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
	if _, ok := c.channels[Topic(topic)]; ok {
		return nil, fmt.Errorf("channel already exists for topic: %v", topic)
	}
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
