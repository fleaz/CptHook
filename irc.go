package main

import (
	"fmt"
	//"github.com/lrstanley/girc"
)

func ircChannelWorker() {
	for elem := range messageChannel {
		fmt.Println("Got message")
		fmt.Println(elem.Messages)
	}
}
