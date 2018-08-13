package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

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

	if config.IsSet("auth") {
		auth := config.Sub("auth")

		switch auth.GetString("method") {
		case "SASL-Plain":
			clientConfig.SASL = &girc.SASLPlain{
				User: auth.GetString("username"),
				Pass: auth.GetString("password"),
			}

		case "SASL-External":
			clientConfig.SASL = &girc.SASLExternal{
				Identity: auth.GetString("identity"),
			}

		default:
			panic("Unsupported authentication method")
		}
	}

	if config.IsSet("ssl") {
		// Enable / Disable SSL
		config.SetDefault("ssl.enabled", true)
		clientConfig.SSL = config.GetBool("ssl.enabled")

		clientConfig.TLSConfig = &tls.Config{
			ServerName: config.GetString("host"),
		}

		// Configure server certificate
		if cafile := config.GetString("ssl.cafile"); cafile != "" {
			caCert, err := ioutil.ReadFile(cafile)
			if err != nil {
				log.Fatal(err)
			}

			clientConfig.TLSConfig.RootCAs = x509.NewCertPool()
			clientConfig.TLSConfig.RootCAs.AppendCertsFromPEM(caCert)
		}

		// Configure client certificate
		if config.IsSet("ssl.client_cert") {
			certfile := config.GetString("ssl.client_cert.certfile")
			keyfile := config.GetString("ssl.client_cert.keyfile")

			cert, err := tls.LoadX509KeyPair(certfile, keyfile)
			if err != nil {
				log.Fatalf("Invalid client certificate: %s", err)
			}

			clientConfig.TLSConfig.Certificates = []tls.Certificate{cert}
		}
	}

	client = girc.New(clientConfig)

	client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Whois(clientConfig.Nick)
	})

	// Start thread to process message queue
	go channelReceiver()

	for {
		if err := client.Connect(); err != nil {
			log.Printf("Connection to %s terminated: %s", client.Server(), err)
			log.Printf("Reconnecting to %s in 30 seconds...", client.Server())
			time.Sleep(30 * time.Second)
		}
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
