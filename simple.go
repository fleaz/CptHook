package main

import (
	"bufio"
	"net/http"

	"github.com/spf13/viper"
)

type SimpleModule struct{}

func (m SimpleModule) getChannelList() []string {
	return nil
}

func (m SimpleModule) getEndpoint() string {
	return "/simple"
}

func (m SimpleModule) getHandler(c *viper.Viper) http.HandlerFunc {

	defaultChannel := c.GetString("default_channel")

	return func(wr http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		query := req.URL.Query()

		// Get channel to send to
		channel := query.Get("channel")
		if channel == "" {
			channel = defaultChannel
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
