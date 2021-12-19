package facebot

import (
	"encoding/json"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"log"
	"net/http"
)

// author zhasulan
// created on 16.12.21 18:47

type Webhook struct {
	Port string
}

func (w *Webhook) WebhookServer(bot *Bot) {
	r := router.New()
	r.GET("/webhook", WebhookVerify(bot))
	r.POST("/webhook", WebhookListen(bot))
	log.Panic(fasthttp.ListenAndServe(fmt.Sprintf(":%s", w.Port), r.Handler))
}

func WebhookVerify(bot *Bot) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusOK)

		hubMode := ctx.QueryArgs().Peek("hub.mode")
		hubChallenge := ctx.QueryArgs().Peek("hub.challenge")
		hubVerifyToken := ctx.QueryArgs().Peek("hub.verify_token")

		if string(hubMode) == "subscribe" && hubChallenge != nil {
			if string(hubVerifyToken) != bot.VerifyToken {
				ctx.SetStatusCode(http.StatusForbidden)
				return
			}

			if _, e := ctx.Write(hubChallenge); e != nil {
				zap.S().Error(e)
				ctx.SetStatusCode(http.StatusInternalServerError)
				return
			}
		}
	}
}

type FBChat struct {
	ID string `json:"id"`
}

type SendMessage struct {
	MID  string `json:"mid"`
	Text string `json:"text"`
}

type Read struct {
	Watermark int `json:"watermark"`
}

type Messaging struct {
	Sender    FBChat       `json:"sender"`
	Recipient FBChat       `json:"recipient"`
	Timestamp int          `json:"timestamp"`
	Message   *SendMessage `json:"message"`
	Read      *Read        `json:"read"`
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int         `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Tick struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Button struct {
	Type    string `json:"type,omitempty"`
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type QuickReplies struct {
	ContentType string `json:"content_type,omitempty"`
	Text        string `json:"text,omitempty"`
	Payload     string `json:"payload,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

type Element struct {
	Title    string   `json:"title,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	ItemURL  string   `json:"item_url,omitempty"`
	ImageURL string   `json:"image_url,omitempty"`
	Buttons  []Button `json:"buttons,omitempty"`
}

type Payload struct {
	TemplateType string    `json:"template_type,omitempty"`
	Text         string    `json:"text,omitempty"`
	Elements     []Element `json:"elements,omitempty"`
	Buttons      []Button  `json:"buttons,omitempty"`
	URL          string    `json:"url,omitempty"`
}

type Attachment struct {
	Type    string   `json:"type"`
	Payload *Payload `json:"payload"`
}

type TextMessage struct {
	Text         *string       `json:"text,omitempty"`
	Attachment   *Attachment   `json:"attachment,omitempty"`
	QuickReplies *QuickReplies `json:"quick_replies,omitempty"`
}

type TextResponse struct {
	Recipient     FBChat      `json:"recipient,omitempty"`
	MessagingType *string     `json:"messaging_type,omitempty"`
	Message       TextMessage `json:"message,omitempty"`
}

func WebhookListen(bot *Bot) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var tick Tick
		if e := json.Unmarshal(ctx.PostBody(), &tick); e != nil {
			zap.S().Error(e)
		}

		if tick.Object == "page" {
			for _, entry := range tick.Entry {
				for _, messaging := range entry.Messaging {
					if messaging.Recipient.ID == bot.PageID {
						if messaging.Message != nil {
							TextHandler(bot, messaging.Sender, messaging.Message.Text)
						}
					}
				}
			}
		}

		ctx.SetStatusCode(http.StatusOK)
	}
}

func TextHandler(bot *Bot, recipient FBChat, text string) {
	textHandler, exist := bot.handlers[OnText]
	if exist {
		handler, ok := textHandler.(func(recipient FBChat, message string))
		if ok {
			handler(recipient, text)
			return
		}
	}

}
