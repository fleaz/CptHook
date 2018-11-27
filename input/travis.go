package input

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
)

type TravisModule struct {
	defaultChannel string
	channel        chan IRCMessage
}

type repository struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner_name"`
	URL   string `json:"URL"`
}

type payload struct {
	ID            int `json:"id"`
	Slug          string
	State         string     `json:"state"`
	StatusMessage string     `json:"status_message"`
	Duration      int        `json:"duration"`
	Branch        string     `json:"branch"`
	CommitMessage string     `json:"message"`
	Repository    repository `json:"repository"`
	Author        string     `json:"author_name"`
	Commiter      string     `json:"committer_name"`
}

func (m *TravisModule) Init(c *viper.Viper, channel *chan IRCMessage) {
	m.defaultChannel = c.GetString("default_channel")
	m.channel = *channel
}

func (m TravisModule) GetEndpoint() string {
	return "/travis"
}

func (m TravisModule) GetChannelList() []string {
	return []string{m.defaultChannel}
}

func (m TravisModule) GetHandler() http.HandlerFunc {
	const startedString = "\x0315{{ .Author }}\x03 started a build for \x0312{{ .Slug }}\x03 with the commit '{{ .CommitMessage }}' on branch {{ .Branch }}"
	const finishedString = "\x0315{{ .Author }}\x03's build for \x0312{{ .Slug }}\x03 finished after {{ .Duration }}sec and {{ .StatusMessage }}"

	BuildStatus := map[string]string{
		"Pending":       "is \x0306pending\x03",
		"Passed":        "has \x0303passed\x03",
		"Fixed":         "is \x0303fixed\x03",
		"Broken":        "is \x0304broken\x03",
		"Failed":        "has \x0304failed\x03",
		"Still Failing": "is \x0304still failing\x03",
		"Canceled":      "was \x0304canceled\x03",
		"Errored":       "has \x0313errored\x03",
	}

	startedTemplate, err := template.New("travis event").Parse(startedString)
	finishedTemplate, err := template.New("travis event").Parse(finishedString)
	if err != nil {
		log.Fatalf("Failed to parse eventString template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got http event for /travis")
		r.ParseForm()
		var encPayload = r.Form.Get("payload")
		t, err := url.QueryUnescape(encPayload)
		if err != nil {
			log.Println("Not properly URL-encoded")
			log.Fatal(err)
		}
		var p payload
		err = json.Unmarshal([]byte(t), &p)
		if err != nil {
			log.Println("Not valid json")
			log.Fatal(err)
		}

		p.Slug = r.Header["Travis-Repo-Slug"][0]
		p.StatusMessage = BuildStatus[p.StatusMessage]
		var buf bytes.Buffer

		if p.State == "started" {
			startedTemplate.Execute(&buf, &p)
		} else {
			finishedTemplate.Execute(&buf, &p)
		}

		m.channel <- IRCMessage{
			Messages: []string{buf.String()},
			Channel:  m.defaultChannel,
		}
	}
}
