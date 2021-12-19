package facebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// author zhasulan
// created on 15.12.21 23:00

type Chat struct {
	ID int64 `json:"id"`

	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Message struct {
	Chat *Chat  `json:"chat"`
	Text string `json:"text"`

	Contact *Contact `json:"contact"`
	Voice   *Voice   `json:"voice"`
}

const (
	OnText    = "\atext"
	OnContact = "\acontact"
	OnVoice   = "\avoice"
	OnAudio   = "\aaudio"
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

func (b *Bot) ChatByID(id string) (*Chat, error) {
	return nil, nil
}

func (b *Bot) GetFile(file *File) (io.ReadCloser, error) {
	return nil, nil
}

func (b *Bot) Handle(endpoint, handler interface{}) {
	switch end := endpoint.(type) {
	case string:
		b.handlers[end] = handler
	default:
		panic("fbbot: unsupported endpoint")
	}
}

type ReplyButton struct {
	Text     string `json:"text"`
	Contact  bool
	Location bool
}

type ReplyMarkup struct {
	ReplyKeyboard       [][]ReplyButton `json:"keyboard,omitempty"`
	ResizeReplyKeyboard bool            `json:"resize_reply_keyboard"`
	ReplyKeyboardRemove bool            `json:"reply_keyboard_remove"`

	InlineKeyboard [][]InlineButton `json:"inline_keyboard,omitempty"`
}

func (r *ReplyMarkup) Text(text string) Btn {
	return Btn{Text: text}
}

func (r *ReplyMarkup) Contact(text string) Btn {
	return Btn{Contact: true, Text: text}
}

func (r *ReplyMarkup) Data(text, unique string, data ...string) Btn {
	return Btn{
		Unique: unique,
		Text:   text,
		Data:   strings.Join(data, "|"),
	}
}

func (r *ReplyMarkup) Row(many ...Btn) Row {
	return many
}

func (r *ReplyMarkup) Reply(rows ...Row) {
	replyKeys := make([][]ReplyButton, 0, len(rows))
	for i, row := range rows {
		keys := make([]ReplyButton, 0, len(row))
		for j, btn := range row {
			btn := btn.Reply()
			if btn == nil {
				panic(fmt.Sprintf(
					"telebot: button row %d column %d is not a reply button",
					i, j))
			}
			keys = append(keys, *btn)
		}
		replyKeys = append(replyKeys, keys)
	}

	r.ReplyKeyboard = replyKeys
}

type SendOptions struct {
	ReplyMarkup         *ReplyMarkup
	ParseMode           interface{}
	DisableNotification interface{}
}

type Btn struct {
	Unique   string
	Text     string
	URL      string
	Data     string
	Contact  bool
	Location bool
}

func (b Btn) Reply() *ReplyButton {
	if b.Unique != "" {
		return nil
	}

	return &ReplyButton{
		Text:     b.Text,
		Contact:  b.Contact,
		Location: b.Location,
	}
}

func (b *Btn) CallbackUnique() string {
	if b.Unique != "" {
		return "\f" + b.Unique
	}
	return b.Text
}

type Row []Btn

func (b *Bot) SendAlbum(chat *Chat, album Album, options ...interface{}) ([]Message, error) {
	return nil, nil
}

func (b *Bot) Send(chat *Chat, what interface{}, options ...interface{}) (*Message, error) {

	var text string
	switch w := what.(type) {
	case string:
		text = w
	}

	var elements []Element
	for _, option := range options {
		switch o := option.(type) {
		case *SendOptions:
			var buttons []Button
			for _, replyButtons := range (*o.ReplyMarkup).ReplyKeyboard {
				for _, replyButton := range replyButtons {
					buttons = append(buttons, Button{
						Type:    "postback",
						Title:   replyButton.Text,
						Payload: replyButton.Text,
					})
				}

				if len(buttons) == 3 {
					elements = append(elements, Element{
						Title:   "➡",
						Buttons: buttons,
					})
					buttons = []Button{}
				}
			}

			if len(buttons) != 3 && len(buttons) != 0 {
				elements = append(elements, Element{
					Title:   "➡",
					Buttons: buttons,
				})
				buttons = []Button{}
			}
		}
	}

	textResponse := TextResponse{
		Recipient: FBChat{ID: strconv.FormatInt(chat.ID, 10)},
		Message: TextMessage{
			Text: &text,
		},
	}

	body, err := json.Marshal(textResponse)
	if err != nil {
		return nil, err
	}

	// Send Text
	if e := b.facebookRequest(body); e != nil {
		return nil, e
	}

	buttonsResponse := TextResponse{
		Recipient: FBChat{ID: strconv.FormatInt(chat.ID, 10)},
		Message: TextMessage{
			Attachment: &Attachment{
				Type: "template",
				Payload: &Payload{
					TemplateType: "generic",
					Elements:     elements,
				},
			},
		},
	}

	body, err = json.Marshal(buttonsResponse)
	if err != nil {
		return nil, err
	}

	if e := b.facebookRequest(body); e != nil {
		return nil, e
	}

	return nil, err
}

func (b *Bot) facebookRequest(body []byte) error {
	request, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v2.6/me/messages", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	query := request.URL.Query()
	query.Add("access_token", b.PageAccessToken)
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

	return nil // todo return message id
}

func (b *Bot) Start() {
	webhook := Webhook{Port: "8080"}
	webhook.WebhookServer(b)
}

// File object represents any sort of file.
type File struct {
	FileID   string `json:"file_id"`
	UniqueID string `json:"file_unique_id"`
	FileSize int    `json:"file_size"`

	// file on telegram server https://core.telegram.org/bots/api#file
	FilePath string `json:"file_path"`

	// file on local file system.
	FileLocal string `json:"file_local"`

	// file on the internet
	FileURL string `json:"file_url"`

	// file backed with io.Reader
	FileReader io.Reader `json:"-"`

	fileName string
}

type ParseMode = string

const (
	ModeDefault    ParseMode = ""
	ModeMarkdown   ParseMode = "Markdown"
	ModeMarkdownV2 ParseMode = "MarkdownV2"
	ModeHTML       ParseMode = "HTML"
)

func FromReader(reader io.Reader) File {
	return File{FileReader: reader}
}

func (r *ReplyMarkup) Inline(rows ...Row) {
	inlineKeys := make([][]InlineButton, 0, len(rows))
	for i, row := range rows {
		keys := make([]InlineButton, 0, len(row))
		for j, btn := range row {
			btn := btn.Inline()
			if btn == nil {
				panic(fmt.Sprintf(
					"telebot: button row %d column %d is not an inline button",
					i, j))
			}
			keys = append(keys, *btn)
		}
		inlineKeys = append(inlineKeys, keys)
	}

	r.InlineKeyboard = inlineKeys
}

// InlineButton represents a button displayed in the message.
type InlineButton struct {
	// Unique slagish name for this kind of button,
	// try to be as specific as possible.
	//
	// It will be used as a callback endpoint.
	Unique string `json:"unique,omitempty"`

	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
	Data string `json:"callback_data,omitempty"`
}

func (b Btn) Inline() *InlineButton {
	return &InlineButton{
		Unique: b.Unique,
		Text:   b.Text,
		URL:    b.URL,
		Data:   b.Data,
	}
}
