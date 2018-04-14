package main

import (
	"net/http/httptest"
	"net/http"
	"testing"
	"github.com/spf13/viper"
	"fmt"
	"os"
)


func TestPrometheusHandler(t *testing.T){
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	file, e := os.Open("./tests/prometheus.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", "/", file)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(prometheusHandler(viper.Sub("modules.prometheus")))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
				status, http.StatusOK)
	}
}