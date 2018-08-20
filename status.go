package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

type StatusModule struct{}

func (m StatusModule) init(c *viper.Viper) {}

func (m StatusModule) getEndpoint() string {
	return "/status"
}

func (m StatusModule) getChannelList() []string {
	return []string{}
}

func (m StatusModule) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got http event for /status")
		t, _ := template.ParseFiles("templates/status.html")
		t.Execute(w, nil)
	}
}
