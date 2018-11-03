package input

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

func TestSimpleHandler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath("../")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	body := strings.NewReader("Hello, World!")

	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var simpleModule Module = &SimpleModule{}
	simpleModule.Init(viper.Sub("modules.simple"), nil)
	handler := http.HandlerFunc(simpleModule.GetHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
