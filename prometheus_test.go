package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

func TestPrometheusHandler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	file, e := os.Open("./tests/prometheus.json")
	if e != nil {
		log.Fatal(e)
	}

	req, err := http.NewRequest("POST", "/", file)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var prometheusModule Module = &PrometheusModule{}
	prometheusModule.init(viper.Sub("modules.prometheus"))
	handler := http.HandlerFunc(prometheusModule.getHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
