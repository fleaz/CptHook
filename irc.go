package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lrstanley/girc"
	"github.com/spf13/viper"
)

var (
	client     *girc.Client
	clientLock = &sync.RWMutex{}
)

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

	clientLock.Lock()
	client = girc.New(clientConfig)

	client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		clientLock.Unlock()
		for _, name := range removeDuplicates(channelList) {
			joinChannel(name)
		}
	})

	client.Handlers.Add(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {
		if e.IsFromUser() {
			log.Debugf("Received a query: %v", e)
			message := "Hi. I'm a CptHook bot."
			if version == "dev" {
				message += fmt.Sprintf(" I was compiled by hand at %v", date)
			} else {
				message += fmt.Sprintf(" I am running v%v (Commit: %v, Builddate: %v)", version, commit, date)
			}
			c.Cmd.ReplyTo(e, message)
		}
	})

	// Start thread to process message queue
	go channelReceiver()

	for {
		if err := client.Connect(); err != nil {
			clientLock.Lock()
			log.Warnf("Connection to %s terminated: %s", client.Server(), err)
			log.Warn("Reconnecting in 30 seconds...")
			time.Sleep(30 * time.Second)
		}
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

func channelReceiver() {
	log.Info("ChannelReceiver started")

	for elem := range inputChannel {
		log.Debug("Took IRC event out of channel.")
		joinChannel(elem.Channel)
		for _, message := range elem.Messages {
			clientLock.RLock()
			client.Cmd.Message(elem.Channel, message)
			clientLock.RUnlock()
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

	clientLock.RLock()
	client.Cmd.Join(newChannel)
	clientLock.RUnlock()
}
