package input

import (
	"net/http"

	"github.com/spf13/viper"
)

// Module defines a common interface for all CptHook modules
type Module interface {
	Init(c *viper.Viper, channel *chan IRCMessage)
	GetChannelList() []string
	GetHandler() http.HandlerFunc
}

// IRCMessage are send over the inputChannel from the different modules
type IRCMessage struct {
	Messages []string
	Channel  string
}
