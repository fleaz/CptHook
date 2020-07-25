package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type GitlabModule struct {
	channelMapping mapping
	channel        chan IRCMessage
	commitLimit    int
}

type mapping struct {
	DefaultChannel   string              `mapstructure:"default"`
	GroupMappings    map[string][]string `mapstructure:"groups"`
	ExplicitMappings map[string][]string `mapstructure:"explicit"`
}

func contains(mapping map[string][]string, entry string) []string {
	for k := range mapping {
		if k == entry {
			return mapping[k]
		}
	}
	return nil
}

func prefixContains(mapping map[string][]string, entry string) []string {
	// Start with -1 so we can also match a single groupname without sub-groups
	length := -1
	var target []string
	for k := range mapping {

		if strings.HasPrefix(entry, k) && len(strings.Split(k, "/")) > length {
			target = mapping[k]
			length = len(strings.Split(k, "/"))
		}
	}
	if len(target) > 0 {
		return target
	} else {
		return nil
	}
}

func (m *GitlabModule) Init(c *viper.Viper, channel *chan IRCMessage) {
	err := c.UnmarshalKey("default", &m.channelMapping.DefaultChannel)
	if err != nil {
		log.Fatal("Failed to unmarshal default-channelmapping into struct")
	}
	err = c.UnmarshalKey("groups", &m.channelMapping.GroupMappings)
	if err != nil {
		log.Fatal("Failed to unmarshal group-channelmapping into struct")
	}
	err = c.UnmarshalKey("explicit", &m.channelMapping.ExplicitMappings)
	if err != nil {
		log.Fatal("Failed to unmarshal explicit-channelmapping into struct")
	}

	m.channel = *channel

	if c.IsSet("commit_limit") {
		commitLimit := c.GetInt("commit_limit")
		if 0 < commitLimit && commitLimit <= 20 {
			m.commitLimit = commitLimit
		} else {
			log.Warn("commit_limit was set to an invalid value. Using default of 3")
			m.commitLimit = 3
		}
	} else {
		m.commitLimit = 3
	}

}

func (m GitlabModule) sendMessage(message string, projectName string, pathWithNamespace string) {
	var channelNames []string

	if list := contains(m.channelMapping.ExplicitMappings, pathWithNamespace); len(list) > 0 { // Check if explizit mapping exists
		for _, channelName := range m.channelMapping.ExplicitMappings[pathWithNamespace] {
			channelNames = append(channelNames, channelName)
		}
	} else if list := prefixContains(m.channelMapping.GroupMappings, pathWithNamespace); len(list) > 0 { // Check if group mapping exists
		for _, channelName := range list {
			channelNames = append(channelNames, channelName)
		}
	} else { // Fall back to default channel
		channelNames = append(channelNames, m.channelMapping.DefaultChannel)
	}

	for _, channelName := range channelNames {
		var event IRCMessage
		event.Messages = append(event.Messages, message)
		event.Channel = channelName
		event.generateID()
		log.WithFields(log.Fields{
			"MsgID":  event.ID,
			"Module": "Gitlab",
		}).Info("Dispatching message to IRC handler")
		m.channel <- event
	}

}

func (m GitlabModule) GetChannelList() []string {
	var all []string

	for _, v := range m.channelMapping.ExplicitMappings {
		for _, name := range v {
			all = append(all, name)
		}
	}
	for _, v := range m.channelMapping.GroupMappings {
		for _, name := range v {
			all = append(all, name)
		}
	}
	all = append(all, m.channelMapping.DefaultChannel)

	return all
}

func (m GitlabModule) GetHandler() http.HandlerFunc {

	const pushCompareString = "[\x0312{{ .Project.Name }}\x03] {{ .UserName }} pushed {{ .TotalCommits }} commits to \x0305{{ .Branch }}\x03 {{ .Project.WebURL }}/compare/{{ .BeforeCommit }}...{{ .AfterCommit }}"
	const pushCommitLogString = "[\x0312{{ .Project.Name }}\x03] {{ .UserName }} pushed {{ .TotalCommits }} commits to \x0305{{ .Branch }}\x03 {{ .Project.WebURL }}/commits/{{ .Branch }}"
	const branchCreateString = "[\x0312{{ .Project.Name }}\x03] {{ .UserName }} created the branch \x0305{{ .Branch }}\x03"
	const branchDeleteString = "[\x0312{{ .Project.Name }}\x03] {{ .UserName }} deleted the branch \x0305{{ .Branch }}\x03"
	const commitString = "\x0315{{ .ShortID }}\x03 (\x0303+{{ .AddedFiles }}\x03|\x0308±{{ .ModifiedFiles }}\x03|\x0304-{{ .RemovedFiles }}\x03) \x0306{{ .Author.Name }}\x03: {{ .Message }}"
	const issueString = "[\x0312{{ .Project.Name }}\x03] {{ .User.Name }} {{ .Issue.Action }} issue \x0308#{{ .Issue.Iid }}\x03: {{ .Issue.Title }} {{ .Issue.URL }}"
	const mergeString = "[\x0312{{ .Project.Name }}\x03] {{ .User.Name }} {{ .Merge.Action }} merge request \x0308#{{ .Merge.Iid }}\x03: {{ .Merge.Title }} {{ .Merge.URL }}"
	const pipelineCreateString = "[\x0312{{ .Project.Name }}\x03] Pipeline for commit {{ .Pipeline.Commit }} {{ .Pipeline.Status }} {{ .Project.WebURL }}/pipelines/{{ .Pipeline.ID }}"
	const pipelineCompleteString = "[\x0312{{ .Project.Name }}\x03] Pipeline for commit {{ .Pipeline.Commit }} {{ .Pipeline.Status }} in {{ .Pipeline.Duration }} seconds {{ .Project.WebURL }}/pipelines/{{ .Pipeline.ID }}"
	const jobCompleteString = "[\x0312{{ .Repository.Name }}\x03] Job \x0308{{ .Name }}\x03 for commit {{ .Commit }} {{ .Status }} in {{ .Duration }} seconds {{ .Repository.Homepage }}/-/jobs/{{ .ID }}"

	JobStatus := map[string]string{
		"pending": "is \x0315pending\x03",
		"created": "was \x0315created\x03",
		"running": "is \x0307running\x03",
		"failed":  "has \x0304failed\x03",
		"success": "has \x0303succeded\x03",
	}

	HookActions := map[string]string{
		"open":   "opened",
		"update": "updated",
		"close":  "closed",
		"reopen": "reopened",
		"merge":  "merged",
	}

	const NullCommit = "0000000000000000000000000000000000000000"

	pushCompareTemplate, err := template.New("push notification").Parse(pushCompareString)
	if err != nil {
		log.Fatalf("Failed to parse pushCompare template: %v", err)
	}

	pushCommitLogTemplate, err := template.New("push to new branch notification").Parse(pushCommitLogString)
	if err != nil {
		log.Fatalf("Failed to parse pushCommitLog template: %v", err)
	}

	branchCreateTemplate, err := template.New("branch creat notification").Parse(branchCreateString)
	if err != nil {
		log.Fatalf("Failed to parse branchDelete template: %v", err)
	}

	branchDeleteTemplate, err := template.New("branch delete notification").Parse(branchDeleteString)
	if err != nil {
		log.Fatalf("Failed to parse branchDelete template: %v", err)
	}

	commitTemplate, err := template.New("commit notification").Parse(commitString)
	if err != nil {
		log.Fatalf("Failed to parse commitString template: %v", err)
	}

	issueTemplate, err := template.New("issue notification").Parse(issueString)
	if err != nil {
		log.Fatalf("Failed to parse issueEvent template: %v", err)
	}

	mergeTemplate, err := template.New("merge notification").Parse(mergeString)
	if err != nil {
		log.Fatalf("Failed to parse mergeEvent template: %v", err)
	}

	pipelineCreateTemplate, err := template.New("pipeline create notification").Parse(pipelineCreateString)
	if err != nil {
		log.Fatalf("Failed to parse pipelineCreateEvent template: %v", err)
	}

	pipelineCompleteTemplate, err := template.New("pipeline complete notification").Parse(pipelineCompleteString)
	if err != nil {
		log.Fatalf("Failed to parse pipelineCompleteEvent template: %v", err)
	}

	jobCompleteTemplate, err := template.New("job complete notification").Parse(jobCompleteString)
	if err != nil {
		log.Fatalf("Failed to parse jobCompleteEvent template: %v", err)
	}

	return func(wr http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		decoder := json.NewDecoder(req.Body)

		var eventType = req.Header.Get("X-Gitlab-Event")
		log.WithFields(log.Fields{
			"EventType": eventType,
		}).Debug("Got a request for the GitlabModule")

		type Project struct {
			Name              string `json:"name"`
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
		}

		type User struct {
			Name string `json:"name"`
		}

		type Issue struct {
			Iid         int    `json:"iid"`
			Action      string `json:"action"`
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
		}

		type Author struct {
			Name string `json:"name"`
		}

		type Commit struct {
			ID       string   `json:"id"`
			Message  string   `json:"message"`
			Added    []string `json:"added"`
			Modified []string `json:"modified"`
			Removed  []string `json:"removed"`
			Author   Author   `json:"author"`
		}

		type PushEvent struct {
			UserName     string   `json:"user_name"`
			BeforeCommit string   `json:"before"`
			AfterCommit  string   `json:"after"`
			Project      Project  `json:"project"`
			Commits      []Commit `json:"commits"`
			TotalCommits int      `json:"total_commits_count"`
			Branch       string   `json:"ref"`
		}

		type IssueEvent struct {
			User    User    `json:"user"`
			Project Project `json:"project"`
			Issue   Issue   `json:"object_attributes"`
		}

		type Merge struct {
			Iid    int    `json:"iid"`
			Action string `json:"action"`
			Title  string `json:"title"`
			URL    string `json:"url"`
		}

		type MergeEvent struct {
			User    User    `json:"user"`
			Project Project `json:"project"`
			Merge   Merge   `json:"object_attributes"`
		}

		type Pipeline struct {
			ID       int     `json:"id"`
			Commit   string  `json:"sha"`
			Status   string  `json:"status"`
			Duration float64 `json:"duration"`
		}

		type PipelineEvent struct {
			Pipeline Pipeline `json:"object_attributes"`
			Project  Project  `json:"project"`
		}

		type Repository struct {
			Name     string `json:"name"`
			Homepage string `json:"homepage"`
			URL      string `json:"url"`
		}

		type JobEvent struct {
			ID         int        `json:"build_id"`
			Name       string     `json:"build_name"`
			Status     string     `json:"build_status"`
			Duration   float64    `json:"build_duration"`
			Commit     string     `json:"sha"`
			Repository Repository `json:"repository"`
		}

		var buf bytes.Buffer

		switch eventType {

		case "Pipeline Hook":
			var pipelineEvent PipelineEvent
			if err := decoder.Decode(&pipelineEvent); err != nil {
				log.Error(err)
				return
			}

			// pending / running
			if pipelineEvent.Pipeline.Status == "pending" {
				log.Printf("Skipping noisy pipeline event with status: %s", pipelineEvent.Pipeline.Status)
				return
			}

			// shorten commit id
			pipelineEvent.Pipeline.Commit = pipelineEvent.Pipeline.Commit[0:7]

			if pipelineEvent.Pipeline.Status == "running" {
				// colorize status
				pipelineEvent.Pipeline.Status = JobStatus[pipelineEvent.Pipeline.Status]

				err = pipelineCreateTemplate.Execute(&buf, &pipelineEvent)
				m.sendMessage(buf.String(), pipelineEvent.Project.Name, pipelineEvent.Project.PathWithNamespace)

			} else if pipelineEvent.Pipeline.Status == "success" || pipelineEvent.Pipeline.Status == "failed" {
				// colorize status
				pipelineEvent.Pipeline.Status = JobStatus[pipelineEvent.Pipeline.Status]

				err = pipelineCompleteTemplate.Execute(&buf, &pipelineEvent)
				m.sendMessage(buf.String(), pipelineEvent.Project.Name, pipelineEvent.Project.PathWithNamespace)
			}

		case "Job Hook":
			var jobEvent JobEvent
			if err := decoder.Decode(&jobEvent); err != nil {
				log.Error(err)
				return
			}

			if jobEvent.Status != "success" && jobEvent.Status != "failed" {
				log.Printf("Skipping noisy job event with status: %s", jobEvent.Status)
				return
			}

			// shorten commit id
			jobEvent.Commit = jobEvent.Commit[0:7]

			// parse namespace from Git URL
			// For some reason the JobEvent doesn't provides the normal
			// namespace and path variables like the other jobs so we
			// have to become creative
			pathWithNamespace := strings.Split(strings.Split(jobEvent.Repository.URL, ":")[1], ".")[0]
			fmt.Println(pathWithNamespace)

			// colorize status
			jobEvent.Status = JobStatus[jobEvent.Status]

			err = jobCompleteTemplate.Execute(&buf, &jobEvent)
			m.sendMessage(buf.String(), jobEvent.Repository.Name, pathWithNamespace)

		case "Merge Request Hook", "Merge Request Event":
			var mergeEvent MergeEvent
			if err := decoder.Decode(&mergeEvent); err != nil {
				log.Error(err)
				return
			}

			mergeEvent.Merge.Action = HookActions[mergeEvent.Merge.Action]

			err = mergeTemplate.Execute(&buf, &mergeEvent)

			m.sendMessage(buf.String(), mergeEvent.Project.Name, mergeEvent.Project.PathWithNamespace)

		case "Issue Hook", "Issue Event":
			var issueEvent IssueEvent
			if err := decoder.Decode(&issueEvent); err != nil {
				log.Error(err)
				return
			}

			issueEvent.Issue.Action = HookActions[issueEvent.Issue.Action]

			err = issueTemplate.Execute(&buf, &issueEvent)

			m.sendMessage(buf.String(), issueEvent.Project.Name, issueEvent.Project.PathWithNamespace)

		case "Push Hook", "Push Event":
			var pushEvent PushEvent
			if err := decoder.Decode(&pushEvent); err != nil {
				log.Error(err)
				return
			}

			pushEvent.Branch = strings.Split(pushEvent.Branch, "/")[2]

			if pushEvent.AfterCommit == NullCommit {
				// Branch was deleted
				var buf bytes.Buffer
				err = branchDeleteTemplate.Execute(&buf, &pushEvent)
				m.sendMessage(buf.String(), pushEvent.Project.Name, pushEvent.Project.PathWithNamespace)
			} else {
				if pushEvent.BeforeCommit == NullCommit {
					// Branch was created
					var buf bytes.Buffer
					err = branchCreateTemplate.Execute(&buf, &pushEvent)
					m.sendMessage(buf.String(), pushEvent.Project.Name, pushEvent.Project.PathWithNamespace)
				}

				if pushEvent.TotalCommits > 0 {
					// when the beforeCommit does not exist, we can't link to a compare without skipping the first commit
					var buf bytes.Buffer
					if pushEvent.BeforeCommit == NullCommit {
						err = pushCommitLogTemplate.Execute(&buf, &pushEvent)
					} else {
						pushEvent.BeforeCommit = pushEvent.BeforeCommit[0:7]
						pushEvent.AfterCommit = pushEvent.AfterCommit[0:7]
						err = pushCompareTemplate.Execute(&buf, &pushEvent)
					}

					m.sendMessage(buf.String(), pushEvent.Project.Name, pushEvent.Project.PathWithNamespace)

					// Limit number of commit meessages to 3
					if pushEvent.TotalCommits > m.commitLimit {
						pushEvent.Commits = pushEvent.Commits[0:m.commitLimit]
					}

					for _, commit := range pushEvent.Commits {
						type CommitContext struct {
							ShortID       string
							Message       string
							Author        Author
							AddedFiles    int
							ModifiedFiles int
							RemovedFiles  int
						}

						context := CommitContext{
							ShortID:       commit.ID[0:7],
							Message:       commit.Message,
							Author:        commit.Author,
							AddedFiles:    len(commit.Added),
							ModifiedFiles: len(commit.Modified),
							RemovedFiles:  len(commit.Removed),
						}

						var buf bytes.Buffer
						err = commitTemplate.Execute(&buf, &context)

						if err != nil {
							log.Printf("ERROR: %v", err)
							return
						}
						m.sendMessage(buf.String(), pushEvent.Project.Name, pushEvent.Project.PathWithNamespace)
					}

					if pushEvent.TotalCommits > m.commitLimit {
						var message = fmt.Sprintf("and %d more commits.", pushEvent.TotalCommits-m.commitLimit)
						m.sendMessage(message, pushEvent.Project.Name, pushEvent.Project.PathWithNamespace)
					}
				}
			}

		default:
			log.WithFields(log.Fields{
				"EventType": eventType,
			}).Warn("Can't handle this event type")
		}

	}

}
