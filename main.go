package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/spf13/viper"
)

// IRCMessage are send over the messageChannel from the different modules
type IRCMessage struct {
	Messages []string
	Channel  string
}

// Module defines a common interface for all CptHook modules
type Module interface {
	init(c *viper.Viper)
	getChannelList() []string
	getEndpoint() string
	getHandler() http.HandlerFunc
}

var messageChannel = make(chan IRCMessage, 10)

func main() {
	confDirPtr := flag.String("config", "/etc/cpthook.yml", "Path to the configfile")
	flag.Parse()

	// Load configuration from file
	confDir, confName := path.Split(*confDirPtr)
	viper.SetConfigName(strings.Split(confName, ".")[0])
	if len(confDir) > 0 {
		viper.AddConfigPath(confDir)
	} else {
		viper.AddConfigPath(".")
	}
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	var moduleList = viper.Sub("modules")
	var channelList = []string{}

	// Status module
	if moduleList.GetBool("status.enabled") {
		log.Println("Status module is active")
		var statusModule Module = &StatusModule{}
		statusModule.init(viper.Sub("modules.status"))
		channelList = append(channelList, statusModule.getChannelList()...)
		http.HandleFunc(statusModule.getEndpoint(), statusModule.getHandler())
	}

	// Prometheus module
	if moduleList.GetBool("prometheus.enabled") {
		log.Println("Prometheus module is active")
		var prometheusModule Module = &PrometheusModule{}
		prometheusModule.init(viper.Sub("modules.prometheus"))
		channelList = append(channelList, prometheusModule.getChannelList()...)
		http.HandleFunc(prometheusModule.getEndpoint(), prometheusModule.getHandler())
	}

	// Gitlab module
	if moduleList.GetBool("gitlab.enabled") {
		log.Println("Gitlab module is active")
		var gitlabModule Module = &GitlabModule{}
		gitlabModule.init(viper.Sub("modules.gitlab"))
		channelList = append(channelList, gitlabModule.getChannelList()...)
		http.HandleFunc(gitlabModule.getEndpoint(), gitlabModule.getHandler())
	}

	// Simple module
	if moduleList.GetBool("simple.enabled") {
		log.Println("Simple module is active")
		var simpleModule Module = &SimpleModule{}
		simpleModule.init(viper.Sub("modules.simple"))
		channelList = append(channelList, simpleModule.getChannelList()...)
		http.HandleFunc(simpleModule.getEndpoint(), simpleModule.getHandler())
	}

	// Start IRC connection
	go ircConnection(viper.Sub("irc"), channelList)

	// Start HTTP server
	log.Fatal(http.ListenAndServe(viper.GetString("http.listen"), nil))

}
