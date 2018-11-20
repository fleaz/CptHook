package input

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

func TestGitlabHandler(t *testing.T) {
	viper.SetConfigName("cpthook")
	viper.AddConfigPath("../")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panic(err)
	}

	file, e := os.Open("./test_data/gitlab.json")
	if e != nil {
		log.Fatal(e)
	}

	req, err := http.NewRequest("POST", "/", file)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	var gitlabModule Module = &GitlabModule{}
	c := make(chan IRCMessage, 10)
	gitlabModule.Init(viper.Sub("modules.gitlab"), &c)
	handler := http.HandlerFunc(gitlabModule.GetHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v wanted %v",
			status, http.StatusOK)
	}
}
