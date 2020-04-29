package gophx

import (
	"fmt"
	"log"
)

// DefaultOnJoin ...
func DefaultOnJoin(payload interface{}) {
	log.Printf("joined: %+v\n", payload)
}

// DefaultOnJoinError ...
func DefaultOnJoinError(payload interface{}) {
	log.Printf("error join: %+v\n", payload)
}

// DefaultOnMessage ...
func DefaultOnMessage(payload interface{}) {
	log.Println("new message:", payload)
}

// Channel ...
type Channel struct {
	socket      *Socket
	topic       Topic
	OnJoin      func(payload interface{})
	OnJoinError func(payload interface{})
	OnMessage   func(payload interface{})
	nextRef     func() int
	joinRef     int
}

func (c *Channel) join(payload interface{}) error {
	if c.OnJoin == nil {
		c.OnJoin = DefaultOnJoin
	}
	if c.OnJoinError == nil {
		c.OnJoinError = DefaultOnJoinError
	}
	if c.OnMessage == nil {
		c.OnMessage = DefaultOnMessage
	}

	ref := c.nextRef()
	c.joinRef = ref
	msg := Message{Topic: c.topic, Payload: payload, Event: JOIN, Ref: ref}
	if err := c.socket.writeJSON(msg); err != nil {
		return err
	}
	return nil
}

// Push new message
func (c *Channel) Push(payload interface{}) error {
	msg := Message{Topic: c.topic, Payload: payload, Event: MSG, Ref: c.nextRef()}
	fmt.Println(msg)
	return c.socket.WriteJSON(msg)
}
