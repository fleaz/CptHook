package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

type IRCMessage struct {
	Messages []string
	Channel  string
}

var messageChannel = make(chan IRCMessage, 10)

func main() {
	confDirPtr := flag.String("configdir", ".", "The folder of your cpthook.yml (Required)")
	flag.Parse()

	// Load configuration frm file
	viper.SetConfigName("cpthook")
	viper.AddConfigPath(*confDirPtr)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	var moduleList = viper.Sub("modules")

	// Status module
	if moduleList.GetBool("status.enabled") {
		log.Println("Status module is active")
		http.HandleFunc("/status", statusHandler)
	} else {
		log.Println("Status module disabled of not configured")
	}

	// Prometheus module
	if moduleList.GetBool("prometheus.enabled") {
		log.Println("Prometheus module is active")
		http.HandleFunc("/prometheus", prometheusHandler(viper.Sub("modules.prometheus")))
	}

	// Gitlab module
	if moduleList.GetBool("gitlab.enabled") {
		log.Println("Gitlab module is active")
		http.HandleFunc("/gitlab", gitlabHandler(viper.Sub("modules.gitlab")))
	}

	// Start IRC connection
	go ircConnection(viper.Sub("irc"))

	// Start HTTP server
	log.Fatal(http.ListenAndServe(viper.GetString("http.listen"), nil))

}
