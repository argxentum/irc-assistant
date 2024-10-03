package irc

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"crypto/tls"
	"fmt"
	irce "github.com/thoj/go-ircevent"
	"regexp"
	"strings"
	"time"
)

const (
	Operator     = "@"
	HalfOperator = "%"
	Voice        = "+"
	Normal       = ""
)

type User struct {
	Nick   string
	Status string
}

func StatusName(code string) string {
	switch code {
	case Operator:
		return "operator"
	case HalfOperator:
		return "half-operator"
	case Voice:
		return "voice"
	}
	return "normal"
}

func IsStatusAtLeast(status, required string) bool {
	switch required {
	case Operator:
		return status == Operator
	case HalfOperator:
		return status == Operator || status == HalfOperator
	case Voice:
		return status == Operator || status == HalfOperator || status == Voice
	case Normal:
		return true
	}
	return false
}

type IRC interface {
	Connect(cfg *config.Config, joinCallback func(channel, nick string)) error
	Listen(ech chan *Event)
	Join(channel string)
	Part(channel string)
	SendMessage(target, message string)
	SendMessages(target string, messages []string)
	GetUser(channel, nick string, callback func(user *User))
	GetUsers(channel string, callback func(users []User))
	Up(channel, nick string)
	Down(channel, nick string)
	Kick(channel, nick, reason string)
	Ban(channel, mask string)
	TemporaryBan(channel, nick, reason string, duration time.Duration)
	Disconnect()
}

const maxMessageLength = 400

func NewIRC(ctx context.Context) IRC {
	return &service{
		ctx: ctx,
	}
}

type service struct {
	ctx  context.Context
	cfg  *config.Config
	conn *irce.Connection
	ech  chan *Event
}

func (s *service) Connect(cfg *config.Config, joinCallback func(channel, nick string)) error {
	s.cfg = cfg

	s.conn = irce.IRC(cfg.Connection.Nick, cfg.Connection.Username)
	s.conn.RealName = cfg.Connection.RealName
	s.conn.Debug = false
	s.conn.VerboseCallbackHandler = false

	if cfg.Connection.TLS {
		s.conn.UseTLS = cfg.Connection.TLS
		s.conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if len(cfg.Connection.NickServ.Password) > 0 {
		s.respondOnce(CodeNotice, func(event *irce.Event) bool {
			if strings.Contains(event.Message(), cfg.Connection.NickServ.IdentifyPattern) {
				s.conn.Privmsgf(cfg.Connection.NickServ.Recipient, cfg.Connection.NickServ.IdentifyCommand, cfg.Connection.NickServ.Password)
				return true
			}
			return false
		})
	}

	if len(cfg.Connection.PostConnect.Code) > 0 {
		s.respondOnce(cfg.Connection.PostConnect.Code, func(event *irce.Event) bool {
			for _, command := range cfg.Connection.PostConnect.Commands {
				s.conn.SendRawf(command, cfg.Connection.Nick)
			}
			for _, channel := range cfg.Connection.PostConnect.AutoJoin {
				s.conn.Join(channel)
			}
			return true
		})
	}

	if joinCallback != nil {
		s.conn.AddCallback(CodeJoin, func(e *irce.Event) {
			joinCallback(e.Message(), e.Nick)
		})
	}

	err := s.conn.Connect(fmt.Sprintf("%s:%d", cfg.Connection.Server, cfg.Connection.Port))
	if err != nil {
		return err
	}

	return nil
}

func (s *service) respondOnce(code string, callback func(event *irce.Event) bool) {
	var id int
	id = s.conn.AddCallback(code, func(event *irce.Event) {
		if callback(event) {
			s.conn.RemoveCallback(code, id)
		}
	})
}

func (s *service) respondUntil(eventCode, completionCode string, eventCallback, completionCallback func(event *irce.Event)) {
	var codeID int
	codeID = s.conn.AddCallback(eventCode, func(event *irce.Event) {
		eventCallback(event)
	})

	var completionID int
	completionID = s.conn.AddCallback(completionCode, func(event *irce.Event) {
		s.conn.RemoveCallback(eventCode, codeID)
		s.conn.RemoveCallback(completionCode, completionID)
		completionCallback(event)
	})
}

func (s *service) Listen(ech chan *Event) {
	s.ech = ech

	s.conn.AddCallback("*", func(event *irce.Event) {
		if s.ech != nil {
			s.ech <- createEvent(event)
		}
	})

	s.conn.Loop()
}

func (s *service) Join(channel string) {
	s.conn.Join(channel)
}

func (s *service) Part(channel string) {
	s.conn.Part(channel)
}

var multipleSpacesRegex = regexp.MustCompile(`\s{2,}`)

func (s *service) SendMessage(target, message string) {
	message = strings.ReplaceAll(message, "\r", "")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\t", " ")
	message = strings.TrimSpace(message)
	message = multipleSpacesRegex.ReplaceAllString(message, " ")

	if len(message) < maxMessageLength {
		s.conn.Privmsg(target, message)
		return
	}

	words := strings.Split(message, " ")
	messages := make([]string, 0)
	current := ""

	for _, word := range words {
		if len(current)+len(word) > maxMessageLength {
			messages = append(messages, current)
			current = ""
		}
		if len(current) > 0 {
			current += " "
		}
		current += word
	}

	if len(current) > 0 {
		messages = append(messages, current)
	}

	for _, m := range messages {
		s.conn.Privmsg(target, m)
	}
}

func (s *service) SendMessages(target string, messages []string) {
	go func() {
		for _, message := range messages {
			s.SendMessage(target, message)
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

func (s *service) GetUsers(channel string, callback func(users []User)) {
	s.conn.SendRawf("NAMES %s", channel)

	allUsers := make([]User, 0)

	s.respondUntil(CodeNamesReply, CodeEndOfNames, func(e *irce.Event) {
		users := make([]User, 0)

		results := strings.Split(e.Message(), " ")
		for _, u := range results {
			if strings.HasPrefix(u, Operator) {
				users = append(users, User{Nick: strings.TrimPrefix(u, Operator), Status: Operator})
			} else if strings.HasPrefix(u, HalfOperator) {
				users = append(users, User{Nick: strings.TrimPrefix(u, HalfOperator), Status: HalfOperator})
			} else if strings.HasPrefix(u, Voice) {
				users = append(users, User{Nick: strings.TrimPrefix(u, Voice), Status: Voice})
			} else {
				users = append(users, User{Nick: u, Status: Normal})
			}
		}

		allUsers = append(allUsers, users...)
	}, func(e *irce.Event) {
		callback(allUsers)
	})
}

func (s *service) GetUser(channel, nick string, callback func(user *User)) {
	s.GetUsers(channel, func(users []User) {
		for _, u := range users {
			if u.Nick == nick {
				callback(&u)
				return
			}
		}
		callback(nil)
	})
}

func (s *service) Up(channel, nick string) {
	s.conn.Privmsgf(s.cfg.Connection.ChanServ.Recipient, s.cfg.Connection.ChanServ.UpCommand, channel, nick)
}

func (s *service) Down(channel, nick string) {
	s.conn.Privmsgf(s.cfg.Connection.ChanServ.Recipient, s.cfg.Connection.ChanServ.DownCommand, channel, nick)
}

func (s *service) Kick(channel, nick, reason string) {
	s.GetUser(channel, s.cfg.Connection.Nick, func(status *User) {
		if status != nil && (status.Status == Operator || status.Status == HalfOperator) {
			s.conn.Kick(nick, channel, reason)
		}
	})
}

func (s *service) Ban(channel, mask string) {
	s.conn.Mode(channel, "+b", mask)
}

func (s *service) TemporaryBan(channel, nick, reason string, duration time.Duration) {
	go func() {
		s.Kick(channel, nick, reason)
		time.Sleep(100 * time.Millisecond)
		s.conn.Mode(channel, "+b", nick)
	}()
}

func (s *service) Disconnect() {
	s.conn.ClearCallback("*")
	s.conn.Disconnect()
}
