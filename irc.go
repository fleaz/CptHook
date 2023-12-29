package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/lrstanley/girc"
	"github.com/spf13/viper"
)

var client *girc.Client

func ircConnection(config *viper.Viper, channelList []string) {
	clientConfig := girc.Config{
		Server:    config.GetString("host"),
		Port:      config.GetInt("port"),
		Nick:      config.GetString("nickname"),
		User:      config.GetString("nickname"),
		PingDelay: 30 * time.Second,
	}

	if config.IsSet("auth") {
		log.Info("Configuring SASL-Auth for IRC connection")
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
		log.Info("Configuring SSL for IRC connection")
		config.SetDefault("ssl.enabled", true)
		clientConfig.SSL = config.GetBool("ssl.enabled")

		clientConfig.TLSConfig = &tls.Config{
			ServerName: config.GetString("host"),
		}

		// Configure server certificate
		if cafile := config.GetString("ssl.cafile"); cafile != "" {
			log.WithFields(logrus.Fields{
				"cafile": cafile,
			}).Info("Using custom CA certificate for the IRC connection")
			caCert, err := ioutil.ReadFile(cafile)
			if err != nil {
				log.Fatal(err)
			}

			clientConfig.TLSConfig.RootCAs = x509.NewCertPool()
			clientConfig.TLSConfig.RootCAs.AppendCertsFromPEM(caCert)
		}

		// Configure client certificate
		if config.IsSet("ssl.client_cert") {
			log.Info("Configuring SSL client certificate for IRC connection")
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
		log.Info("Sucessfully connected to the IRC server. Starting to join channel.")
		for _, name := range removeDuplicates(channelList) {
			joinChannel(name)
		}
	})

	client.Handlers.Add(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {
		if e.IsFromUser() {
			log.WithFields(log.Fields{
				"Event": e.String(),
			}).Debug("Received a PRIMSG")
			message := "Hi. I'm a CptHook bot. Visit https://github.com/fleaz/CptHook to learn more."
			if version == "dev" {
				message += fmt.Sprintf(" I was compiled by hand at %v", date)
			} else {
				message += fmt.Sprintf(" I am running v%v (Commit: %v, Builddate: %v)", version, commit, date)
			}
			c.Cmd.ReplyTo(e, message)
		}
	})

	// Start thread to process message queue
	go channelReceiver(config.GetBool("use_notice"))

	log.Info("Connecting to IRC server")
	for {
		// client.Connect() blocks while we are connected.
		// If the the connection is dropped/broken (recognized if we don't get a PONG 30 seconds
		// after we sent a PING) an error is returned.
		err := client.Connect()
		// If we manually Close() the connection, the Connect() function will exit without an error
		if err != nil {
			log.Warnf("Connection terminated. Reason: %s\n", err)
		}
		log.Warn("Reconnecting in 10 seconds...")
		time.Sleep(10 * time.Second)
	}

}

func contains(e string, slice []string) bool {
	for _, x := range slice {
		if x == e {
			return true
		}
	}
	return false
}

func removeDuplicates(input []string) []string {
	var output []string
	for _, element := range input {
		if !contains(element, output) {
			output = append(output, element)
		}
	}
	return output
}

func channelReceiver(useNotice bool) {
	log.Info("ChannelReceiver started")

	for elem := range inputChannel {
		log.WithFields(log.Fields{
			"MsgID":   elem.ID,
			"text":    elem.Messages,
			"channel": elem.Channel,
		}).Debug("IRC handler received a message")
		joinChannel(elem.Channel)
		for _, message := range elem.Messages {
			if useNotice {
				client.Cmd.Notice(elem.Channel, message)
			} else {
				client.Cmd.Message(elem.Channel, message)
			}
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
	}).Info("Need to join a new channel")
	client.Cmd.Join(newChannel)

}
