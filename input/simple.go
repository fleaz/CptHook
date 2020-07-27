package input

import (
	"bufio"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type SimpleModule struct {
	defaultChannel string
	channel        chan IRCMessage
}

func (m *SimpleModule) Init(c *viper.Viper, channel *chan IRCMessage) {
	m.defaultChannel = c.GetString("default_channel")
	m.channel = *channel
}

func (m SimpleModule) GetChannelList() []string {
	return []string{m.defaultChannel}
}

func (m SimpleModule) GetHandler() http.HandlerFunc {

	return func(wr http.ResponseWriter, req *http.Request) {
		log.Debug("Got a request for the SimpleModule")
		defer req.Body.Close()

		query := req.URL.Query()

		// Get channel to send to
		channel := query.Get("channel")
		if channel == "" {
			channel = m.defaultChannel
		}

		// Split body into lines
		var lines []string
		scanner := bufio.NewScanner(req.Body)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		// Send message
		msg := IRCMessage{
			Messages: lines,
			Channel:  channel,
		}
		msg.generateID()
		log.WithFields(log.Fields{
			"MsgID":  msg.ID,
			"Module": "Simple",
		}).Info("Dispatching message to IRC handler")
		m.channel <- msg
	}
}
