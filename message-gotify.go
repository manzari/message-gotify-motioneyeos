package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var config GotifyConfig
var message GotifyMessage
var configPath = "/data/etc/gotify.json"

type GotifyConfig struct {
	ApiKey                 string
	ServerUrl              string
	MotionDetectedTitle    string
	MotionDetectedMessage  string
	MotionDetectedPriority int
	MediaUploadedTitle     string
	MediaUploadedMessage   string
	MediaUploadedPriority  int
}

type GotifyMessage struct {
	Message  string `json:"message"`
	Priority int    `json:"priority"`
	Title    string `json:"title"`
}

func main() {
	if os.Getenv("MGDEBUG") == "1" {
		configPath = "./gotify.json"
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := GotifyConfig{
			ApiKey:                 "4p1k3y",
			ServerUrl:              "https://gotify.urhost.example",
			MotionDetectedTitle:    "{cN} Motion",
			MotionDetectedMessage:  "Motion was detected by {cN}",
			MotionDetectedPriority: 2,
			MediaUploadedTitle:     "Media Uploaded",
			MediaUploadedMessage:   "{cN} uploaded file {f}",
			MediaUploadedPriority:  2,
		}
		defaultConfigJson, _ := json.Marshal(defaultConfig)
		err = ioutil.WriteFile(configPath, defaultConfigJson, 0600)
		handleError("json marshal failed: ", err)
		println("Default config generated at " + configPath)
		os.Exit(0)
	}
	configFile, err := ioutil.ReadFile(configPath)
	handleError("Failed to read config file: ", err)

	err = json.Unmarshal(configFile, &config)
	handleError("Failed to parse config file: ", err)

	flag.Parse()
	event := flag.Arg(0)
	cameraName := flag.Arg(1)
	filename := flag.Arg(2)
	if event != "motion" && (event != "media" && filename != "") && cameraName != "" {
		println("usage: message-gotify <motion|media> cameraname filename")
		os.Exit(1)
	}

	if event == "motion" {
		message = createMessage(config.MotionDetectedTitle, config.MotionDetectedMessage, config.MotionDetectedPriority, cameraName, filename)
	} else {
		message = createMessage(config.MediaUploadedTitle, config.MediaUploadedMessage, config.MediaUploadedPriority, cameraName, filename)
	}

	jsonMessage, err := json.Marshal(&message)
	handleError("json marshal failed: ", err)
	req, err := http.NewRequest("POST", config.ServerUrl+"/message", bytes.NewBuffer(jsonMessage))
	handleError("Failed to create request: ", err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", config.ApiKey)
	resp, err := http.DefaultClient.Do(req)
	handleError("HTTP POST failed: ", err)
	if resp.StatusCode != 200 {
		handleError("", errors.New("received a non 200 response"))
	}
}

func handleError(message string, err error) {
	if err != nil {
		_, err2 := fmt.Fprint(os.Stderr, message, err)
		if err2 != nil {
			println(err)
			println(err2)
		}
		os.Exit(1)
	}
}

func createMessage(messageTitle string, messageText string, priority int, cameraName string, filename string) GotifyMessage {
	messageTitle = strings.Replace(messageTitle, "{f}", filename, -1)
	messageTitle = strings.Replace(messageTitle, "{cN}", cameraName, -1)
	messageText = strings.Replace(messageText, "{f}", filename, -1)
	messageText = strings.Replace(messageText, "{cN}", cameraName, -1)
	return GotifyMessage{
		Title:    messageTitle,
		Message:  messageText,
		Priority: priority,
	}
}
