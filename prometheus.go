package main

import (
	"fmt"
	"net/http"
)

func prometheusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got http event for /prometheus")
	var event IRCMessage
	event.Messages = append(event.Messages, "prometheus rocks")
	event.Channel = "#fleaz"
	messageChannel <- event
}
