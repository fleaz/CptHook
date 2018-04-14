package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Alert struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    string                 `json:"startsAt"`
	EndsAt      string                 `json:"endsAt"`
}

type Notification struct {
	Version           string                 `json:"version"`
	GroupKey          string                 `json:"groupKey"`
	Status            string                 `json:"status"`
	Receiver          string                 `json:"receiver"`
	GroupLables       map[string]interface{} `json:"groupLabels"`
	CommonLabels      map[string]interface{} `json:"commonLabels"`
	CommonAnnotations map[string]interface{} `json:"commonAnnotations"`
	ExternalURL       string                 `json:"externalURL"`
	Alerts            []Alert                `json:"alerts"`
}

type NotificationContext struct {
	Alert         *Alert
	Notification  *Notification
	InstanceCount int
	Status        string
	ColorStart    string
	ColorEnd      string
}

type Instance struct {
	Name  string
	Value string
}

func SortAlerts(alerts []Alert) (firing, resolved []Alert) {
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

func prometheusHandler(c *viper.Viper) http.HandlerFunc {
	fmt.Println("Got http event for /prometheus")

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


	return func(w http.ResponseWriter, r *http.Request){
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)

		var notification Notification

		if err := decoder.Decode(&notification); err != nil {
			log.Println(err)
			return
		}

		body, err := json.Marshal(&notification)

		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("JSON: %v", string(body))

		var sortedAlerts = make(map[string][]Alert)
		sortedAlerts["firing"], sortedAlerts["resolved"] = SortAlerts(notification.Alerts)

		var instance Instance
		var instanceList []Instance
		var buf bytes.Buffer

		for alertStatus, alertList := range sortedAlerts {
			// Clear buffer
			buf.Reset()
			// Clear InstanceList
			instanceList = instanceList[:0]

			for _, alert := range alertList {
				name := alert.Labels["instance"].(string)
				// TODO: Add hostname shortening
				value, ok := alert.Annotations["value"].(string)
				if ok {
					instance = Instance{Name: name, Value: value}
				} else {
					instance = Instance{Name: name}
				}
				instanceList = append(instanceList, instance)
			}

			context := NotificationContext{
				Alert:         &notification.Alerts[0],
				Notification:  &notification,
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
