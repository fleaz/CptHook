package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fleaz/CptHook/input"
	"github.com/spf13/viper"
)

var (
	inputChannel = make(chan input.IRCMessage, 30)
	version      = "dev"
	commit       = "none"
	date         = time.Now().Format(time.RFC3339)
)

func configureLogLevel() {
	if l := viper.GetString("logging.level"); l != "" {
		level, err := log.ParseLevel(l)
		if err != nil {
			log.WithFields(log.Fields{
				"level": l,
			}).Fatal("Uknown loglevel defined in configuration.")
		}
		log.WithFields(log.Fields{
			"level": level,
		}).Info("Setting loglevel defined by configuration")
		log.SetLevel(level)
		return
	}
	log.Info("Loglevel not defined in configuration. Defaulting to ERROR")
	log.SetLevel(log.ErrorLevel)
}

type Configuration struct {
	Modules map[string]InputModule `yaml:"modules"`
}

type InputModule struct {
	Type     string `yaml:"type"`
	Endpoint string `yaml:"endpoint"`
}

func createModuleObject(name string) (input.Module, error) {
	var m input.Module
	var e error
	switch name {
	case "gitlab":
		m = &input.GitlabModule{}
	case "prometheus":
		m = &input.PrometheusModule{}
	case "simple":
		m = &input.SimpleModule{}
	case "icinga2":
		m = &input.Icinga2Module{}
	default:
		e = fmt.Errorf("ignoring configuration for unknown module: %q", name)
	}

	return m, e
}

func validateConfig(c Configuration) {
	var foundErrors []string

	for blockName, blockConfig := range c.Modules {
		if blockConfig.Type == "" {
			foundErrors = append(foundErrors, fmt.Sprintf("Block %q is missing its type", blockName))
		}
		if blockConfig.Endpoint == "" {
			foundErrors = append(foundErrors, fmt.Sprintf("Block %q is missing its endpoint", blockName))
		}
	}

	if len(foundErrors) > 0 {
		log.Error("Found the following errors in the configuration:")
		for _, e := range foundErrors {
			log.Error(e)
		}
		os.Exit(1)
	} else {
		log.Info("Configuration parsed without errors")
	}

}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"remote": r.RemoteAddr,
			"method": r.Method,
			"host":   r.Host,
			"uri":    r.URL,
		}).Debug("Received HTTP request")

		next.ServeHTTP(w, r)
	})
}

func ircCheckMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !client.IsConnected() {
			log.WithFields(log.Fields{
				"remote": r.RemoteAddr,
				"uri":    r.URL,
			}).Warn("IRC server is disconnected. Dropping incoming HTTP request")

			// In some weird situations the IsConnected function detects that we are no longer connected,
			// but the reconnect logic in irc.go doesn't detects the connection problem and won't reconnect
			// Therefore if we detect that problem here, we Close() the connection manually and force a reconenct
			client.Close()

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("IRC server disconnected"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		log.Fatal(err)
	}
	configureLogLevel()

	var channelList = []string{}
	var config = Configuration{}

	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal(err)
	}

	validateConfig(config)

	for blockName, blockConfig := range config.Modules {
		module, err := createModuleObject(blockConfig.Type)
		if err != nil {
			log.Warn(err)
			continue
		}
		log.Infof("Loaded block %q from config (Type %q, Endpoint %q)", blockName, blockConfig.Type, blockConfig.Endpoint)
		configPath := fmt.Sprintf("modules.%s", blockName)
		module.Init(viper.Sub(configPath), &inputChannel)
		channelList = append(channelList, module.GetChannelList()...)
		http.HandleFunc(blockConfig.Endpoint, loggingMiddleware(ircCheckMiddleware(module.GetHandler())))
	}

	// Start IRC connection
	go ircConnection(viper.Sub("irc"), channelList)

	// Start HTTP server
	srv := &http.Server{
		Addr:         viper.GetString("http.listen"),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	srv.SetKeepAlivesEnabled(false)

	log.WithFields(log.Fields{
		"listen": viper.GetString("http.listen"),
	}).Info("Started HTTP Server")

	log.Fatal(srv.ListenAndServe(), nil)

}
