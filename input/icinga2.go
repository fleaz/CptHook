package input

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

func JsonToTime(ts json.Number) time.Time {
	t, err := ts.Float64()
	if err == nil {
		return time.Unix(int64(t), 0)
	}
	return time.Unix(0, 0)
}

func AgoString(t time.Time) string {
	return "since " + strings.Trim(humanize.RelTime(time.Now(), t, "", ""), " ")
}

func ColorHostState(s string) string {
	HostState := map[string]string{
		"UP":   "\x0303Up\x03",
		"DOWN": "\x0304Down\x03",
	}
	return HostState[s]
}

func ColorServiceState(s string) string {
	ServiceState := map[string]string{
		"UNKNOWN":  "\x0313Unknown\x03",
		"CRITICAL": "\x0304Critical\x03",
		"WARNING":  "\x0308Warning\x03",
		"OK":       "\x0303Ok\x03",
	}
	return ServiceState[s]
}

type Host struct {
	CheckAttempt           float32     `json:"check_attempt"`
	Name                   string      `json:"name"`
	DisplayName            string      `json:"display_name"`
	HostGroups             []string    `json:"hostgroups"`
	State                  string      `json:"state"`
	StateType              string      `json:"state_type"`
	LastState              string      `json:"last_state"`
	LastStateType          string      `json:"last_state_type"`
	LastHardState          json.Number `json:"last_hard_state"`
	Output                 string      `json:"output"`
	WebURL                 string      `json:"url"`
	LastStateChangeStr     json.Number `json:"last_state_change"`
	LastHardStateChangeStr json.Number `json:"last_hard_state_change"`
}

func (h Host) ColoredState() string {
	return ColorHostState(h.State)
}

func (h Host) ColoredLastState() string {
	return ColorServiceState(h.LastState)
}

func (h Host) LastStateChange() time.Time {
	return JsonToTime(h.LastStateChangeStr)
}

func (h Host) LastHardStateChange() time.Time {
	return JsonToTime(h.LastHardStateChangeStr)
}

func (h Host) AgoString() string {
	return AgoString(h.LastStateChange())
}

type Service struct {
	CheckAttempt           float32     `json:"check_attempt"`
	Name                   string      `json:"name"`
	DisplayName            string      `json:"display_name"`
	State                  string      `json:"state"`
	StateType              string      `json:"state_type"`
	LastState              string      `json:"last_state"`
	LastStateType          string      `json:"last_state_type"`
	LastHardState          json.Number `json:"last_hard_state"`
	Output                 string      `json:"output"`
	WebURL                 string      `json:"url"`
	LastStateChangeStr     json.Number `json:"last_state_change"`
	LastHardStateChangeStr json.Number `json:"last_hard_state_change"`
}

func (s Service) ColoredState() string {
	return ColorServiceState(s.State)
}

func (s Service) ColoredLastState() string {
	return ColorServiceState(s.LastState)
}

func (s Service) LastStateChange() time.Time {
	return JsonToTime(s.LastStateChangeStr)
}

func (s Service) LastHardStateChange() time.Time {
	return JsonToTime(s.LastHardStateChangeStr)
}

func (s Service) AgoString() string {
	return AgoString(s.LastStateChange())
}

type Notification struct {
	Author    string      `json:"author"`
	Comment   string      `json:"comment"`
	Target    string      `json:"target"`
	Type      string      `json:"type"`
	Timestamp json.Number `json:"timet"`
	DateTime  string      `json:"long_date_time"`
	Host      Host        `json:"host"`
	Service   Service     `json:"service"`
	Channels  []string    `json:"channels"`
}

type Icinga2Module struct {
	channelMapping hgmapping
	channel        chan IRCMessage
}

type hgmapping struct {
	DefaultChannel    string              `mapstructure:"default"`
	HostGroupMappings map[string][]string `mapstructure:"hostgroups"`
	ExplicitMappings  map[string][]string `mapstructure:"explicit"`
}

func (m *Icinga2Module) Init(c *viper.Viper, channel *chan IRCMessage) {
	err := c.UnmarshalKey("default", &m.channelMapping.DefaultChannel)
	if err != nil {
		log.Panic(err)
	}
	err = c.UnmarshalKey("hostgroups", &m.channelMapping.HostGroupMappings)
	if err != nil {
		log.Panic(err)
	}
	err = c.UnmarshalKey("explicit", &m.channelMapping.ExplicitMappings)
	if err != nil {
		log.Panic(err)
	}
	m.channel = *channel
}

func (m Icinga2Module) sendMessage(message string, notification Notification) {
	var channelNames []string
	var hostname = notification.Host.Name
	if contains(m.channelMapping.ExplicitMappings, hostname) { // Check if explicit mapping exists
		for _, channelName := range m.channelMapping.ExplicitMappings[hostname] {
			channelNames = append(channelNames, channelName)
		}
	} else {
		var found = false
		for _, hostgroup := range notification.Host.HostGroups { // Check if hostgroup mapping exists
			if contains(m.channelMapping.HostGroupMappings, hostgroup) {
				for _, channelName := range m.channelMapping.HostGroupMappings[hostgroup] {
					channelNames = append(channelNames, channelName)
					found = true
				}
			}
		}
		if !found { // Fall back to default channel
			channelNames = append(channelNames, m.channelMapping.DefaultChannel)
		}
	}

	for _, channelName := range channelNames {
		var event IRCMessage
		event.Messages = append(event.Messages, message)
		event.Channel = channelName
		event.generateID()
		log.WithFields(log.Fields{
			"MsgID":  event.ID,
			"Module": "Icinga2",
		}).Info("Dispatching message to IRC handler")
		m.channel <- event
	}

}

func (m Icinga2Module) GetChannelList() []string {
	var all []string

	for _, v := range m.channelMapping.ExplicitMappings {
		for _, name := range v {
			all = append(all, name)
		}
	}
	for _, v := range m.channelMapping.HostGroupMappings {
		for _, name := range v {
			all = append(all, name)
		}
	}
	all = append(all, m.channelMapping.DefaultChannel)
	return all
}

func (m Icinga2Module) GetHandler() http.HandlerFunc {

	const serviceStateChangeString = "Service \x0312{{ .Service.DisplayName }}\x03 (\x0314{{ .Host.DisplayName }}\x03) transitioned from state {{ .Service.ColoredLastState }} to {{ .Service.ColoredState }}"
	const serviceStateEnteredString = "Service \x0312{{ .Service.DisplayName }}\x03 (\x0314{{ .Host.DisplayName }}\x03) entered state {{ .Service.ColoredState }}"
	const serviceStateString = "Service \x0312{{ .Service.DisplayName }}\x03 (\x0314{{ .Host.DisplayName }}\x03) is still in state {{ .Service.ColoredState }} ({{ .Service.AgoString }})"
	const serviceAckString = "{{ .Author }} acknowledged service \x0312{{ .Service.DisplayName }}\x03 (State {{ .Service.ColoredState }} {{ .Service.AgoString }})"
	const serviceRecoveryString = "Service \x0312{{ .Service.DisplayName }}\x03 (\x0314{{ .Host.DisplayName }}\x03) \x0303recovered\x03 from state {{ .Service.ColoredLastState }}"
	const serviceOutputString = "→ {{ .Service.Output }}"

	const hostStateChangeString = "Host \x0312{{ .Host.DisplayName }}\x03 transitioned from state {{ .Host.ColoredLastState }} to {{ .Host.ColoredState }}"
	const hostStateEnteredString = "Host \x0312{{ .Host.DisplayName }}\x03 entered state {{ .Host.ColoredState }}"
	const hostStateString = "Host \x0312{{ .Host.DisplayName }}\x03 is still in state {{ .Host.ColoredState }} ({{ .Host.AgoString }})"
	const hostAckString = "{{ .Author }} acknowledged host \x0312{{ .Host.DisplayName }}\x03 (State {{ .Host.ColoredState }} {{ .Host.AgoString }})"
	const hostRecoveryString = "Host \x0312{{ .Host.DisplayName }}\x03 \x0303recovered\x03 from state {{ .Host.ColoredLastState }}"
	const hostOutputString = "→ {{ .Host.Output }}"

	serviceStateChangeTemplate, err := template.New("hostOutput").Parse(serviceStateChangeString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceStateChange template: %v", err)
	}

	serviceStateEnteredTemplate, err := template.New("hostOutput").Parse(serviceStateEnteredString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceStateEntered template: %v", err)
	}

	serviceStateTemplate, err := template.New("serviceState").Parse(serviceStateString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceState template: %v", err)
	}

	serviceAckTemplate, err := template.New("serviceState").Parse(serviceAckString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceAck template: %v", err)
	}

	serviceRecoveryTemplate, err := template.New("serviceState").Parse(serviceRecoveryString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceRecovery template: %v", err)
	}

	serviceOutputTemplate, err := template.New("serviceOutput").Parse(serviceOutputString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse serviceOutput template: %v", err)
	}

	hostStateChangeTemplate, err := template.New("hostOutput").Parse(hostStateChangeString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostStateChange template: %v", err)
	}

	hostStateEnteredTemplate, err := template.New("hostOutput").Parse(hostStateEnteredString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostStateEntered template: %v", err)
	}

	hostStateTemplate, err := template.New("hostState").Parse(hostStateString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostState template: %v", err)
	}

	hostAckTemplate, err := template.New("serviceState").Parse(hostAckString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostAck template: %v", err)
	}

	hostRecoveryTemplate, err := template.New("serviceState").Parse(hostRecoveryString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostRecovery template: %v", err)
	}

	hostOutputTemplate, err := template.New("hostOutput").Parse(hostOutputString)
	if err != nil {
		log.Fatalf("[icinga2] Failed to parse hostOutput template: %v", err)
	}

	return func(wr http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		decoder := json.NewDecoder(req.Body)

		var buf bytes.Buffer
		var notification Notification
		if err := decoder.Decode(&notification); err != nil {
			log.Panic(err)
			return
		}

		log.WithFields(log.Fields{
			"event": notification.Target,
		}).Warn("Got a request for the Icinga2Module")

		switch notification.Target {

		case "service":
			if notification.Type == "ACKNOWLEDGEMENT" { // Acknowledge
				err = serviceAckTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Type == "RECOVERY" { // Recovery
				err = serviceRecoveryTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Service.LastStateType != notification.Service.StateType { // State entered
				err = serviceStateEnteredTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
				buf.Reset()
				err = serviceOutputTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Service.LastState == notification.Service.State { // Renotification
				err = serviceStateTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else { // State changed
				err = serviceStateChangeTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
				buf.Reset()
				err = serviceOutputTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			}

		case "host":
			if notification.Type == "ACKNOWLEDGEMENT" { // Acknowledge
				err = hostAckTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Type == "RECOVERY" { // Recovery
				err = hostRecoveryTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Host.LastStateType != notification.Host.StateType { // State entered
				err = hostStateEnteredTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
				buf.Reset()
				err = hostOutputTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else if notification.Host.LastState == notification.Host.State { // Renotification
				err = hostStateTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			} else { // State changed
				err = hostStateChangeTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
				buf.Reset()
				err = hostOutputTemplate.Execute(&buf, &notification)
				m.sendMessage(buf.String(), notification)
			}
		default:
			log.WithFields(log.Fields{
				"event": notification.Target,
			}).Warn("Unknown event")
		}

	}

}
