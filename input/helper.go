package input

import (
	"math/rand"
	"net/http"

	"github.com/spf13/viper"
)

var letterRunes = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Module defines a common interface for all CptHook modules
type Module interface {
	Init(c *viper.Viper, channel *chan IRCMessage)
	GetChannelList() []string
	GetHandler() http.HandlerFunc
}

// IRCMessage are send over the inputChannel from the different modules
type IRCMessage struct {
	ID       string
	Messages []string
	Channel  string
}

func (m *IRCMessage) generateID() {
	b := make([]rune, 6)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	m.ID = string(b)
}
