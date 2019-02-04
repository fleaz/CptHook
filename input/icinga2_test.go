package input

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

func TestIcinga2Handler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath("../")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	file, e := os.Open("./test_data/icinga2.json")
	if e != nil {
		log.Fatal(e)
	}

	req, err := http.NewRequest("POST", "/", file)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var icinga2Module Module = &Icinga2Module{}
	c := make(chan IRCMessage, 10)
	icinga2Module.Init(viper.Sub("modules.icinga2"), &c)
	handler := http.HandlerFunc(icinga2Module.GetHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
