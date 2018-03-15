package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/lrstanley/girc"
	"github.com/spf13/viper"
)

var client *girc.Client

func ircConnection(config *viper.Viper) {
	clientConfig := girc.Config{
		Server: config.GetString("host"),
		Port:   config.GetInt("port"),
		Nick:   config.GetString("nickname"),
		User:   config.GetString("nickname"),
	}

	if config.GetBool("ssl") {
		clientConfig.SSL = true
	}

	if config.IsSet("cafile") {
		cafile := config.GetString("cafile")
		caCertPool := x509.NewCertPool()
		caCert, err := ioutil.ReadFile(cafile)
		if err != nil {
			log.Fatal(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs:    caCertPool,
			ServerName: config.GetString("host"),
		}

		clientConfig.TLSConfig = tlsConfig
	}

	client = girc.New(clientConfig)

	client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Join("#fleaz")
	})

	// Start thread to process message queue
	go channelReceiver()

	if err := client.Connect(); err != nil {
		log.Fatalf("An error occurred while attempting to connect to %s: %s", client.Server(), err)
	}

}

func channelReceiver() {
	log.Println("ChannelReceiver started")

	for elem := range messageChannel {
		fmt.Println("Took IRC event out of channel.")
		joinChannel(elem.Channel)
		for _, message := range elem.Messages {
			client.Cmd.Message(elem.Channel, message)
		}
	}
}

func joinChannel(newChannel string) {
	for _, channelName := range client.ChannelList() {
		if strings.Compare(newChannel, channelName) == 0 {
			return
		}
	}
	fmt.Printf("Need to join new channel %s\n", newChannel)
	client.Cmd.Join(newChannel)
}
