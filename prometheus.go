package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

func prometheusHandler(w http.ResponseWriter, r *http.Request, c *viper.Viper) {
	fmt.Println("Got http event for /prometheus")
	var event IRCMessage
	event.Messages = append(event.Messages, "prometheus rocks")
	event.Channel = c.GetString("channel")
	messageChannel <- event
}
