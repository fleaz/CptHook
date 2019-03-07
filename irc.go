package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lrstanley/girc"
	"github.com/spf13/viper"
)

var client *girc.Client

func ircConnection(config *viper.Viper, channelList []string) {
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
		for _, name := range channelList {
			joinChannel(name)
		}
	})

	// Start thread to process message queue
	go channelReceiver()

	for {
		if err := client.Connect(); err != nil {
			log.Warnf("Connection to %s terminated: %s", client.Server(), err)
			log.Warn("Reconnecting in 30 seconds...")
			time.Sleep(30 * time.Second)
		}
	}

}

func channelReceiver() {
	log.Info("ChannelReceiver started")

	for elem := range inputChannel {
		log.Debug("Took IRC event out of channel.")
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

	log.WithFields(log.Fields{
		"channel": newChannel,
	}).Debug("Need to join new channel")

	client.Cmd.Join(newChannel)
}
