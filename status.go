package main

import (
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type StatusModule struct{}

func (m *StatusModule) init(c *viper.Viper) {}

func (m StatusModule) getEndpoint() string {
	return "/status"
}

func (m StatusModule) getChannelList() []string {
	return []string{}
}

func (m StatusModule) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Got a request for the StatusModule")
		t, _ := template.ParseFiles("templates/status.html")
		t.Execute(w, nil)
	}
}
