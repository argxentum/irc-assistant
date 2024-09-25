package irc

import (
	"github.com/google/uuid"
	irce "github.com/thoj/go-ircevent"
	"strings"
)

type EntityType string

const (
	EntityTypeUser    EntityType = "user"
	EntityTypeChannel EntityType = "channel"
)

type Event struct {
	ID        string
	Raw       string
	Code      string
	From      string
	Source    string
	Arguments []string
}

func (e *Event) Message() string {
	if len(e.Arguments) == 0 {
		return ""
	}
	return e.Arguments[len(e.Arguments)-1]
}

func (e *Event) IsPrivateMessage() bool {
	_, t := e.Recipient()
	return e.Code == CodePrivateMessage && t == EntityTypeUser
}

func (e *Event) Sender() (string, EntityType) {
	if IsChannel(e.From) {
		return e.From, EntityTypeChannel
	}
	return e.From, EntityTypeUser
}

func (e *Event) Recipient() (string, EntityType) {
	if len(e.Arguments) > 0 && IsChannel(e.Arguments[0]) {
		return e.Arguments[0], EntityTypeChannel
	}
	return e.Arguments[0], EntityTypeUser
}

func (e *Event) ReplyTarget() string {
	target := ""
	if e.IsPrivateMessage() {
		sender, _ := e.Sender()
		target = sender
	} else {
		recipient, _ := e.Recipient()
		target = recipient
	}
	return target
}

func createEvent(e *irce.Event) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Raw:       e.Raw,
		Code:      e.Code,
		From:      e.Nick,
		Source:    e.Source,
		Arguments: e.Arguments,
	}
}

func IsChannel(target string) bool {
	return strings.HasPrefix(target, "#") || strings.HasPrefix(target, "&")
}
