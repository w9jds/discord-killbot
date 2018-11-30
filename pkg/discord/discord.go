package discord

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Attachment Discord attachment object
type Attachment struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	TTS       bool    `json:"tts,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// Embed Discord embed object
type Embed struct {
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	URL         string    `json:"url,omitempty"`
	Timestamp   string    `json:"timestamp,omitempty"`
	Color       uint64    `json:"color,omitempty"`
	Footer      Footer    `json:"footer,omitempty"`
	Image       Image     `json:"image,omitempty"`
	Thumbnail   Image     `json:"thumbnail,omitempty"`
	Provider    Reference `json:"provider,omitempty"`
	Author      Author    `json:"author,omitempty"`
	Fields      []Field   `json:"fields,omitempty"`
}

// Footer Discord footer object
type Footer struct {
	Text         string `json:"text,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// Reference Discord reference object
type Reference struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Author Discord author object
type Author struct {
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// Image Discord image object
type Image struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int16  `json:"height,omitempty"`
	Width    int16  `json:"width,omitempty"`
}

// Field Discord field object
type Field struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

type rateLimited struct {
	Message    string `json:"message"`
	RetryAfter uint16 `json:"retry_after"`
	Global     bool   `json:"global"`
}

// Discord base structure
type Discord struct {
	client *http.Client
}

// NewAPI creates a new instance of Discord with passed in http client
func NewAPI(httpClient *http.Client) *Discord {
	return &Discord{client: httpClient}
}

// PushWebhook publish an attachment to a webhook on Discord
func (discord *Discord) PushWebhook(webhook string, buffer []byte) {
	request, error := http.NewRequest("POST", webhook, bytes.NewBuffer(buffer))
	if error != nil {
		log.Println("Error creating discord : ", error)
	}

	request.Header.Add("Content-Type", "application/json")

	response, error := discord.client.Do(request)
	if error != nil {
		log.Println("Error pushing webhook: ", error)
	}

	if response.StatusCode == 429 {
		body, error := ioutil.ReadAll(response.Body)
		if error != nil {
			log.Println(error)
		}

		var rate rateLimited
		error = json.Unmarshal(body, &rate)
		if error != nil {
			log.Println(error)
		}

		time.Sleep(time.Duration(rate.RetryAfter) * time.Millisecond)
		discord.PushWebhook(webhook, buffer)
	}

	if response.StatusCode != 204 && response.StatusCode != 429 {
		body, error := ioutil.ReadAll(response.Body)
		if error != nil {
			log.Println(error)
		}

		log.Println(string(body))
	}
}
