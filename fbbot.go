package fb_bot

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

// author zhasulan
// created on 15.12.21 23:00

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

func (f *Bot) Send(body []byte) error {
	request, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v2.6/me/messages", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	query := request.URL.Query()
	query.Add("access_token", f.PageAccessToken)
	request.URL.RawQuery = query.Encode()

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer func() {
		if e := response.Body.Close(); e != nil {
			log.Panic(e)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("facebook return error status code: %d", response.StatusCode)
	}

	return nil
}

func (f *Bot) Start() {
	webhook := Webhook{Port: "8080"}
	webhook.WebhookServer(f)
}
