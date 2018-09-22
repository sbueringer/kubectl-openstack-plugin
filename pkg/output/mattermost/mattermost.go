/*
Copyright 2016 Skippbox, Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mattermost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var mattermostColors = map[string]string{
	"Normal":  "#00FF00",
	"Warning": "#FFFF00",
	"Danger":  "#FF0000",
}

// Mattermost handler implements handler.Handler interface,
// Notify event to Mattermost channel
type Mattermost struct {
	Channel  string
	Url      string
	Username string
}

type MattermostMessage struct {
	Channel      string                         `json:"channel"`
	Username     string                         `json:"username"`
	IconUrl      string                         `json:"icon_url"`
	Text         string                         `json:"text"`
	Attachements []MattermostMessageAttachement `json:"attachments"`
}

type MattermostMessageAttachement struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	Color string `json:"color"`
}

// Init prepares Mattermost configuration
func New() *Mattermost {
	channel := os.Getenv("KUBECTL_OS_MATTERMOST_CHANNEL")
	url := os.Getenv("KUBECTL_OS_MATTERMOST_URL")
	username := os.Getenv("KUBECTL_OS_MATTERMOST_USERNAME")

	m := &Mattermost{
		Channel:  channel,
		Url:      url,
		Username: username,
	}

	err := checkMissingMattermostVars(m)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *Mattermost) SendMessage(text string) {
	mattermostMessage := prepareMattermostMessage(m, text)

	err := postMessage(m.Url, mattermostMessage)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to channel %s at %s", m.Channel, time.Now())
}

func checkMissingMattermostVars(s *Mattermost) error {
	if s.Channel == "" || s.Url == "" || s.Username == "" {
		return fmt.Errorf("missing Mattermost channel, url or username")
	}

	return nil
}

func prepareMattermostMessage(m *Mattermost, msg string) *MattermostMessage {
	return &MattermostMessage{
		Channel:  m.Channel,
		Username: m.Username,
		IconUrl:  "https://raw.githubusercontent.com/sbueringer/kubectl-openstack-plugin/master/openstack-logo.png",
		Text:     msg,
		// move msg to attachment as soon as Mattermost supports it without errors on our Mattermost: https://github.com/mattermost/mattermost-server/pull/7707
		//Attachements: []MattermostMessageAttachement{
		//	{
		//		Title: msg,
		//		Text:  msg,
		//		Color: mattermostColors["Normal"],
		//	},
		//},
	}
}

func postMessage(url string, mattermostMessage *MattermostMessage) error {
	message, err := json.Marshal(mattermostMessage)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
