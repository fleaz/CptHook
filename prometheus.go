package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type PrometheusModule struct {
}

type alert struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    string                 `json:"startsAt"`
	EndsAt      string                 `json:"endsAt"`
}

type notification struct {
	Version           string                 `json:"version"`
	GroupKey          string                 `json:"groupKey"`
	Status            string                 `json:"status"`
	Receiver          string                 `json:"receiver"`
	GroupLables       map[string]interface{} `json:"groupLabels"`
	CommonLabels      map[string]interface{} `json:"commonLabels"`
	CommonAnnotations map[string]interface{} `json:"commonAnnotations"`
	ExternalURL       string                 `json:"externalURL"`
	Alerts            []alert                `json:"alerts"`
}

type notificationContext struct {
	Alert         *alert
	Notification  *notification
	InstanceCount int
	Status        string
	ColorStart    string
	ColorEnd      string
}

type instance struct {
	Name  string
	Value string
}

func sortAlerts(alerts []alert) (firing, resolved []alert) {
	for _, alert := range alerts {
		tStart, _ := time.Parse(time.RFC3339, alert.StartsAt)
		tEnd, _ := time.Parse(time.RFC3339, alert.EndsAt)
		if tEnd.After(tStart) {
			resolved = append(resolved, alert)
		} else {
			firing = append(firing, alert)
		}
	}
	return
}

func getColorcode(status string) string {
	switch status {
	case "firing":
		return "\x0305"
	case "resolved":
		return "\x0303"
	default:
		return "\x0300"
	}
}

func shortenInstanceName(name string, pattern string) string {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(name)
	if len(match) > 1 {
		return match[1]
	}
	return name
}

func (m PrometheusModule) getEndpoint() string {
	return "/prometheus"
}

func (m PrometheusModule) getChannelList() []string {
	return []string{"foo", "bar"}
}

func (m PrometheusModule) getHandler(c *viper.Viper) http.HandlerFunc {

	const firingTemplateString = "[{{ .ColorStart }}{{ .Status }}{{ .ColorEnd }}:{{ .InstanceCount }}] {{ .Alert.Labels.alertname}} - {{ .Alert.Annotations.description}}"
	const resolvedTemplateString = "[{{ .ColorStart }}{{ .Status }}{{ .ColorEnd }}:{{ .InstanceCount }}] {{ .Alert.Labels.alertname}}"
	const hostListTemplateString = "→ {{range $i, $instance := . }}{{if $i}}, {{end}}{{$instance.Name}}{{if $instance.Value}} ({{$instance.Value}}){{end}}{{end}}"

	firingTemplate, err := template.New("notification").Parse(firingTemplateString)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	resolvedTemplate, err := template.New("notification").Parse(resolvedTemplateString)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	hostListTemplate, err := template.New("notification").Parse(hostListTemplateString)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got http event for /prometheus")
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)

		var n notification

		if err := decoder.Decode(&n); err != nil {
			log.Println(err)
			return
		}

		_, err := json.Marshal(&n)

		if err != nil {
			log.Println(err)
			return
		}

		var sortedAlerts = make(map[string][]alert)
		sortedAlerts["firing"], sortedAlerts["resolved"] = sortAlerts(n.Alerts)

		var inst instance
		var instanceList []instance
		var buf bytes.Buffer

		for alertStatus, alertList := range sortedAlerts {
			// Clear buffer
			buf.Reset()
			// Clear InstanceList
			instanceList = instanceList[:0]

			for _, alert := range alertList {
				name := alert.Labels["instance"].(string)
				name = shortenInstanceName(name, c.GetString("hostname_filter"))
				value, ok := alert.Annotations["value"].(string)
				if ok {
					inst = instance{Name: name, Value: value}
				} else {
					inst = instance{Name: name}
				}
				instanceList = append(instanceList, inst)
			}

			context := notificationContext{
				Alert:         &n.Alerts[0],
				Notification:  &n,
				Status:        strings.ToUpper(alertStatus),
				InstanceCount: len(instanceList),
				ColorStart:    getColorcode(alertStatus),
				ColorEnd:      "\x03",
			}

			if context.InstanceCount > 0 {
				// Sort instances
				//sort.Strings(instanceList)
				if strings.Compare(alertStatus, "firing") == 0 {
					_ = firingTemplate.Execute(&buf, &context)
				} else {
					_ = resolvedTemplate.Execute(&buf, &context)
				}
				var event IRCMessage
				event.Messages = append(event.Messages, buf.String())
				buf.Reset()
				_ = hostListTemplate.Execute(&buf, &instanceList)
				event.Messages = append(event.Messages, buf.String())
				event.Channel = c.GetString("channel")
				messageChannel <- event
			}
		}
	}

}
