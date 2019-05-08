package main

import (
	"flag"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fleaz/CptHook/input"
	"github.com/spf13/viper"
)

var (
	inputChannel = make(chan input.IRCMessage, 10)
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
	Enalbed string `yaml:"enabled"`
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
		e = fmt.Errorf("found configuration for unknown module: %q", name)
	}

	return m, e
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
	var configurtaion = Configuration{}

	err = viper.Unmarshal(&configurtaion)
	if err != nil {
		log.Fatal(err)
	}

	for moduleName := range configurtaion.Modules {
		module, err := createModuleObject(moduleName)
		if err != nil {
			log.Warn(err)
			continue
		}
		log.Infof("Loaded module %q", moduleName)
		configPath := fmt.Sprintf("modules.%s", moduleName)
		module.Init(viper.Sub(configPath), &inputChannel)
		channelList = append(channelList, module.GetChannelList()...)
		http.HandleFunc(module.GetEndpoint(), module.GetHandler())

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
