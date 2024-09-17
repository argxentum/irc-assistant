package core

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"crypto/tls"
	"fmt"
	irce "github.com/thoj/go-ircevent"
	"strings"
	"time"
)

const (
	Operator     = "@"
	HalfOperator = "%"
	Voice        = "+"
	Normal       = ""
)

func ChannelStatusName(status string) string {
	switch status {
	case Operator:
		return "operator"
	case HalfOperator:
		return "half-operator"
	case Voice:
		return "voice"
	}
	return "normal"
}

func IsChannelStatusAtLeast(status, required string) bool {
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

const (
	CodeNotice         = "NOTICE"
	CodeJoin           = "JOIN"
	CodeInvite         = "INVITE"
	CodePrivateMessage = "PRIVMSG"
)

type IRC interface {
	Connect(cfg *config.Config, autoJoinCallback func(channel string)) error
	Listen(ech chan *Event)
	Join(channel string)
	Part(channel string)
	SendMessage(target, message string)
	SendMessages(target string, messages []string)
	GetUserStatus(channel, user string, callback func(status string))
	Kick(channel, user, reason string)
	Ban(channel, user, reason string)
	TemporaryBan(channel, user, reason string, duration time.Duration)
	Disconnect()
}

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

func (s *service) Connect(cfg *config.Config, autoJoinCallback func(channel string)) error {
	s.cfg = cfg

	s.conn = irce.IRC(cfg.Connection.Nick, cfg.Connection.Nick)
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

	s.respondOnce(CodeJoin, func(event *irce.Event) bool {
		if autoJoinCallback != nil {
			autoJoinCallback(event.Message())
		}
		return true
	})

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

func (s *service) Listen(ech chan *Event) {
	s.ech = ech

	s.conn.AddCallback("*", func(event *irce.Event) {
		if s.ech != nil {
			s.ech <- mapEvent(event)
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

func (s *service) SendMessage(target, message string) {
	s.conn.Privmsg(target, message)
}

func (s *service) SendMessages(target string, messages []string) {
	go func() {
		for i, message := range messages {
			// rate limit every third message to channels
			if IsChannel(target) && (i%3) == 0 && i > 0 {
				time.Sleep(500 * time.Millisecond)
			}

			s.SendMessage(target, message)
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

func (s *service) GetUserStatus(channel, user string, callback func(status string)) {
	s.conn.SendRawf("NAMES %s", channel)

	s.respondOnce(s.cfg.Connection.NamesResponseCode, func(e *irce.Event) bool {
		users := strings.Split(e.Message(), " ")
		for _, u := range users {
			if u == fmt.Sprintf("@%s", user) {
				callback(Operator)
				return true
			}
			if u == fmt.Sprintf("%%%s", user) {
				callback(HalfOperator)
				return true
			}
			if u == fmt.Sprintf("+%s", user) {
				callback(Voice)
				return true
			}
			if u == user {
				callback(Normal)
				return true
			}
		}
		return false
	})
}

func (s *service) Kick(channel, user, reason string) {
	s.GetUserStatus(channel, s.cfg.Connection.Nick, func(status string) {
		if status == Operator || status == HalfOperator {
			s.conn.Kick(user, channel, reason)
		}
	})
}

func (s *service) Ban(channel, user, reason string) {
	go func() {
		s.Kick(channel, user, reason)
		time.Sleep(250 * time.Millisecond)
		s.conn.Mode(channel, "+b", user)
	}()
}

func (s *service) TemporaryBan(channel, user, reason string, duration time.Duration) {
	go func() {
		s.Kick(channel, user, reason)
		time.Sleep(250 * time.Millisecond)
		s.conn.Mode(channel, "+b", user)
	}()
}

func (s *service) Disconnect() {
	s.conn.ClearCallback("*")
	s.conn.Disconnect()
}
