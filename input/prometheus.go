package input

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type PrometheusModule struct {
	defaultChannel string
	channel        chan IRCMessage
	hostnameFilter *regexp.Regexp
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

func shortenInstanceName(name string, pattern *regexp.Regexp) string {
	if net.ParseIP(name) != nil {
		// Don't try to shorten an IP address
		return name
	}
	match := pattern.FindStringSubmatch(name)
	if len(match) > 1 {
		return match[1]
	}
	return name
}

func (m PrometheusModule) GetChannelList() []string {
	return []string{m.defaultChannel}
}

func (m *PrometheusModule) Init(c *viper.Viper, channel *chan IRCMessage) {
	m.defaultChannel = c.GetString("channel")
	pattern, err := regexp.Compile(c.GetString("hostname_filter"))
	if err != nil {
		log.Fatalf("Error while parsing hostname_filter: %s", err)
	}
	m.channel = *channel
	m.hostnameFilter = pattern
}

func (m PrometheusModule) GetHandler() http.HandlerFunc {

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
		log.Debug("Got a request for the PrometheusModule")
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
				name := getNameFromLabels(&alert, m.hostnameFilter)
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
				event.Channel = m.defaultChannel
				m.channel <- event
			}
		}
	}

}

// getNameFromLabels tries to determine a meaningful name for an alert
// If the alert has no 'instance' label, we use the 'alertname' which should always
// be present in an alert
func getNameFromLabels(alert *alert, pattern *regexp.Regexp) string {
	if instance, ok := alert.Labels["instance"]; ok {
		return shortenInstanceName(instance.(string), pattern)
	} else if alertName, ok := alert.Labels["alertname"]; ok {
		return alertName.(string)
	} else {
		return "unknown"
	}
}
