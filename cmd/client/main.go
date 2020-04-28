package main

import (
	"bufio"
	"flag"
	"fmt"
	"gophx"
	"os"
	"time"
)

func main() {
	user := flag.String("user", "", "user name")
	topic := flag.String("topic", "", "topic name")
	flag.Parse()

	if *user == "" || *topic == "" {
		flag.Usage()
		os.Exit(1)
	}

	config := &gophx.ClientConfig{
		URL:    "ws://localhost:4000/socket/websocket",
		Params: map[string]string{"user": *user},
	}

	client, err := gophx.NewClient(config)
	if err != nil {
		panic(err)
	}
	if err := client.Connect(); err != nil {
		panic(err)
	}

	channel, err := client.Channel(*topic)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Printf("here %v:", *user)
		msg := scanner.Text()
		if msg != "" {
			if err := channel.Push(map[string]interface{}{"body": msg}); err != nil {
				panic(err)
			}
		}
	}

	if scanner.Err() != nil {
		// handle error.
		fmt.Println("error: scanner:", scanner.Err())
	}
	time.Sleep(3 * time.Second)
}
