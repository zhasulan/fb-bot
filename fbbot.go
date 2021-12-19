package facebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// author zhasulan
// created on 15.12.21 23:00

type Chat struct {
	ID int64
}

type Message struct {
	Chat
}

const (
	Text = "\atext"
)

type Settings struct {
	PageAccessToken string
	VerifyToken     string
	PageID          string
}

func NewBot(settings Settings) (*Bot, error) {
	return &Bot{
		PageAccessToken: settings.PageAccessToken,
		VerifyToken:     settings.VerifyToken,
		PageID:          settings.PageID,
		handlers:        make(map[string]interface{}),
	}, nil
}

type Bot struct {
	PageAccessToken string
	VerifyToken     string
	PageID          string

	handlers map[string]interface{}
}

func (f *Bot) Handle(endpoint, handler interface{}) {
	switch end := endpoint.(type) {
	case string:
		f.handlers[end] = handler
	default:
		panic("fbbot: unsupported endpoint")
	}
}

type ReplyButton struct {
	Text string `json:"text"`
}

type ReplyMarkup struct {
	ReplyKeyboard [][]ReplyButton `json:"keyboard,omitempty"`
}

type SendOptions struct {
	ReplyMarkup         ReplyMarkup
	ParseMode           interface{}
	DisableNotification interface{}
}

func (f *Bot) Send(chat *Chat, what interface{}, options ...interface{}) (*Message, error) {

	var text string
	switch w := what.(type) {
	case string:
		text = w
	}

	var elements []Element
	for _, option := range options {
		switch o := option.(type) {
		case SendOptions:
			for _, replyButtons := range o.ReplyMarkup.ReplyKeyboard {
				var buttons []Button

				for _, replyButton := range replyButtons {
					buttons = append(buttons, Button{
						Type:    "postback",
						Title:   replyButton.Text,
						Payload: "",
					})
				}

				elements = append(elements, Element{
					Title:   "➡",
					Buttons: buttons,
				})
			}
		}
	}

	textResponse := TextResponse{
		Recipient: FBChat{ID: strconv.FormatInt(chat.ID, 10)},
		Message: TextMessage{
			Text: &text,
			Attachment: &Attachment{
				Type: "template",
				Payload: &Payload{
					TemplateType: "generic",
					Elements:     elements,
				},
			},
		},
	}

	body, err := json.Marshal(textResponse)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v2.6/me/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	query := request.URL.Query()
	query.Add("access_token", f.PageAccessToken)
	request.URL.RawQuery = query.Encode()

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() {
		if e := response.Body.Close(); e != nil {
			log.Panic(e)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook return error status code: %d", response.StatusCode)
	}

	return nil, nil // todo return message id
}

func (f *Bot) Start() {
	webhook := Webhook{Port: "8080"}
	webhook.WebhookServer(f)
}
