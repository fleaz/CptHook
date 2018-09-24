package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestTravisHandler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	payload, e := ioutil.ReadFile("./tests/travis.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	// Craft a Travis Request with payload in form-data
	form := url.Values{}
	enc := url.QueryEscape(string(payload))
	form.Add("payload", string(enc))

	req, err := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Travis-Repo-Slug", "f-breidenstein/CptHook")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var travisModule Module = &TravisModule{}
	travisModule.init(viper.Sub("modules.travis"))
	handler := http.HandlerFunc(travisModule.getHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
