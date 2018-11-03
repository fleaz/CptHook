package input

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
	viper.AddConfigPath("../")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	file, e := os.Open("./test_data/prometheus.json")
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
	c := make(chan IRCMessage, 1)
	prometheusModule.Init(viper.Sub("modules.prometheus"), &c)
	handler := http.HandlerFunc(prometheusModule.GetHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
