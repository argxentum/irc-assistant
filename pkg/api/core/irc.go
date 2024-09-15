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
		for _, message := range messages {
			s.SendMessage(target, message)
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

func (s *service) Disconnect() {
	s.conn.ClearCallback("*")
	s.conn.Disconnect()
}
