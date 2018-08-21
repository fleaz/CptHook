package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
)

type TravisModule struct {
	defaultChannel string
}

type payload struct {
	ID       int    `json:"id"`
	State    string `json:"state"`
	Duration int    `json:"duration"`
}

func (m *TravisModule) init(c *viper.Viper) {
	m.defaultChannel = c.GetString("default_channel")
}

func (m TravisModule) getEndpoint() string {
	return "/travis"
}

func (m TravisModule) getChannelList() []string {
	return []string{m.defaultChannel}
}

func (m TravisModule) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got http event for /travis")
		r.ParseForm()
		var encPayload = r.Form.Get("payload")
		t, err := url.QueryUnescape(encPayload)
		if err != nil {
			log.Println("Not properly URL-encoded")
			log.Fatal(err)
		}
		var p payload
		err = json.Unmarshal([]byte(t), &p)
		if err != nil {
			log.Println("Not valid json")
			log.Fatal(err)
		}
		fmt.Printf("Build took %d seconds\n", p.Duration)
	}
}
