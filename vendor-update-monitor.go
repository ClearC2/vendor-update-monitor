package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Commit struct {
	ID       string   `json:"id"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

type GitEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Commits []Commit `json:"commits"`
	Compare string   `json:"compare"`
}

type Config struct {
	Patterns     []string    `json:"patterns"`
	Ref          string      `json:"ref"`
	SlackWebhook string      `json:"slack"`
	Port         json.Number `json:"port"`
}

type WebhookResponse struct {
	Files       []string `json:"files"`
	ShouldAlert bool     `json:"shouldAlert"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackBlock struct {
	Type string    `json:"type"`
	Text SlackText `json:"text"`
}

type SlackBlocks struct {
	Blocks []SlackBlock `json:"blocks"`
}

func main() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// reread config so we don't have to restart the service for changes to take affect
		config, configErr := getConfig()
		if configErr != nil {
			fmt.Println("Could not parse config")
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println("Error reading request body:", err)
			return
		}
		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "push":
			var pushEvent GitEvent
			err := json.Unmarshal(body, &pushEvent)
			if err != nil {
				fmt.Println("Error unmarshaling push event:", err)
				return
			}
			ref := pushEvent.Ref
			if ref == config.Ref {
				fmt.Println("New commit on branch")
				files := getViolatingFiles(config, &pushEvent)
				w.Header().Set("Content-Type", "application/json")
				shouldAlert := len(files) > 0

				if shouldAlert {
					sendSlackMessage(config, files, pushEvent.Compare)
				}
				payload, _ := json.Marshal(WebhookResponse{Files: files, ShouldAlert: shouldAlert})
				w.Write(payload)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	config, configErr := getConfig()
	if configErr != nil {
		fmt.Println("Could not parse config")
		return
	}
	err := http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}

func getConfig() (*Config, error) {
	file, fileErr := os.ReadFile(os.Args[1])
	if fileErr != nil {
		return nil, fileErr
	}
	var config Config
	err := json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func getViolatingFiles(config *Config, pushEvent *GitEvent) []string {
	patterns := getPatterns(config)
	files := make([]string, 0)

	for _, commit := range pushEvent.Commits {
		files = append(files, getMatchingFiles(patterns, commit.Added)...)
		files = append(files, getMatchingFiles(patterns, commit.Modified)...)
		files = append(files, getMatchingFiles(patterns, commit.Removed)...)
	}

	return removeDuplicates(files)
}

func getPatterns(config *Config) []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, len(config.Patterns))

	for i, p := range config.Patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			panic(err)
		}
		patterns[i] = r
	}
	return patterns
}

func getMatchingFiles(patterns []*regexp.Regexp, filenames []string) []string {
	matches := make([]string, 0)
	for _, filename := range filenames {
		for _, pattern := range patterns {
			if pattern.MatchString(filename) {
				matches = append(matches, filename)
				break
			}
		}
	}
	return matches
}

func removeDuplicates(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, str := range strs {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

func sendSlackMessage(config *Config, files []string, compare string) {
	blocks := []SlackBlock{{
		Type: "header",
		Text: SlackText{
			Type: "plain_text",
			Text: "Someone is changing flagged files",
		},
	}}
	blocks = append(blocks, SlackBlock{
		Type: "section",
		Text: SlackText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("```%s```", strings.Join(files, "\n")),
		},
	})
	blocks = append(blocks, SlackBlock{
		Type: "section",
		Text: SlackText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("<%s|Link to github>", compare),
		},
	})
	jsonStr, _ := json.Marshal(SlackBlocks{Blocks: blocks})
	_, sendErr := http.Post(
		config.SlackWebhook,
		"application/json",
		bytes.NewBuffer(jsonStr),
	)
	if sendErr != nil {
		fmt.Println("Could not send to slack")
	}
}
