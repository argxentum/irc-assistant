package core

import (
	"fmt"
	irce "github.com/thoj/go-ircevent"
	"strings"
)

type EntityType string

const (
	EntityTypeUser    EntityType = "user"
	EntityTypeChannel EntityType = "channel"
)

type Event struct {
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

func mapEvent(e *irce.Event) *Event {
	return &Event{
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

func LogEvent(e *Event) {
	entities := ""
	if len(e.From) > 0 {
		from, fromType := e.Sender()
		to, toType := e.Recipient()

		if e.Code == CodePrivateMessage && len(from) > 0 && len(e.Source) > 0 {
			entities = fmt.Sprintf(" %s::%s(%s) -> %s::%s --", fromType, from, e.Source, toType, to)
		} else if e.Code == CodePrivateMessage && len(from) > 0 {
			entities = fmt.Sprintf(" %s::%s -> %s::%s --", fromType, from, toType, to)
		} else if len(from) > 0 && len(e.Source) > 0 {
			entities = fmt.Sprintf(" %s::%s(%s) --", fromType, from, e.Source)
		} else {
			entities = fmt.Sprintf(" %s --", e.From)
		}
	}

	fmt.Printf("[%s]%s %s\n", e.Code, entities, e.Message())
}
