package main

import (
	"fmt"
	"net/http"
)

func prometheusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got prometheus http event")
	var event IRCMessage
	event.Messages = append(event.Messages, "prometheus rocks")
	messageChannel <- event
}
