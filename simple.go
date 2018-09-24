package main

import (
	"bufio"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type SimpleModule struct {
	defaultChannel string
}

func (m *SimpleModule) init(c *viper.Viper) {
	m.defaultChannel = c.GetString("default_channel")
}

func (m SimpleModule) getChannelList() []string {
	return []string{m.defaultChannel}
}

func (m SimpleModule) getEndpoint() string {
	return "/simple"
}

func (m SimpleModule) getHandler() http.HandlerFunc {

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
		messageChannel <- IRCMessage{
			Messages: lines,
			Channel:  channel,
		}
	}
}
