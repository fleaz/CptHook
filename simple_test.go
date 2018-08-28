package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"github.com/spf13/viper"
)

func TestSimpleHandler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	body := strings.NewReader("Hello, World!")

	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var simpleModule Module = &SimpleModule{}
	simpleModule.init(viper.Sub("modules.simple"))
	handler := http.HandlerFunc(simpleModule.getHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
