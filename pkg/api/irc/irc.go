package irc

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"crypto/tls"
	"fmt"
	irce "github.com/thoj/go-ircevent"
	"regexp"
	"strings"
	"time"
)

type ChannelStatus string

const (
	ChannelStatusOperator     ChannelStatus = "@"
	ChannelStatusHalfOperator ChannelStatus = "%"
	ChannelStatusVoice        ChannelStatus = "+"
	ChannelStatusNormal       ChannelStatus = ""
)

type User struct {
	Mask   *Mask
	Status ChannelStatus
}

func UserByTrimmingStatusPrefix(u string) *User {
	if strings.HasPrefix(u, string(ChannelStatusOperator)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusOperator))}, Status: ChannelStatusOperator}
	} else if strings.HasPrefix(u, string(ChannelStatusHalfOperator)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusHalfOperator))}, Status: ChannelStatusHalfOperator}
	} else if strings.HasPrefix(u, string(ChannelStatusVoice)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusVoice))}, Status: ChannelStatusVoice}
	} else {
		return &User{Mask: &Mask{Nick: u}, Status: ChannelStatusNormal}
	}
}

func StatusName(status ChannelStatus) string {
	switch status {
	case ChannelStatusOperator:
		return "operator"
	case ChannelStatusHalfOperator:
		return "half-operator"
	case ChannelStatusVoice:
		return "voice"
	}
	return "normal"
}

func IsStatusAtLeast(status, required ChannelStatus) bool {
	switch required {
	case ChannelStatusOperator:
		return status == ChannelStatusOperator
	case ChannelStatusHalfOperator:
		return status == ChannelStatusOperator || status == ChannelStatusHalfOperator
	case ChannelStatusVoice:
		return status == ChannelStatusOperator || status == ChannelStatusHalfOperator || status == ChannelStatusVoice
	case ChannelStatusNormal:
		return true
	}
	return false
}

type IRC interface {
	Connect(cfg *config.Config, connectCallback func(ctx context.Context, cfg *config.Config, i IRC), joinChannelCallback func(channel, nick string)) error
	Listen(ech chan *Event)
	Join(channel string)
	Part(channel string)
	SendMessage(target, message string)
	SendMessages(target string, messages []string)
	GetUser(channel, nick string, callback func(user *User))
	ListUsers(channel string, callback func(users []*User))
	Up(channel, nick string)
	Down(channel, nick string)
	Kick(channel, nick, reason string)
	Ban(channel, mask string)
	Unban(channel, mask string)
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

func (s *service) Connect(cfg *config.Config, connectCallback func(ctx context.Context, cfg *config.Config, irc IRC), joinChannelCallback func(channel, nick string)) error {
	s.cfg = cfg

	s.conn = irce.IRC(cfg.IRC.Nick, cfg.IRC.Username)
	s.conn.RealName = cfg.IRC.RealName
	s.conn.Debug = false
	s.conn.VerboseCallbackHandler = false

	if cfg.IRC.TLS {
		s.conn.UseTLS = cfg.IRC.TLS
		s.conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if len(cfg.IRC.NickServ.Password) > 0 {
		s.respondOnce(CodeNotice, func(event *irce.Event) bool {
			if strings.Contains(event.Message(), cfg.IRC.NickServ.IdentifyPattern) {
				s.conn.Privmsgf(cfg.IRC.NickServ.Recipient, cfg.IRC.NickServ.IdentifyCommand, cfg.IRC.NickServ.Password)
				return true
			}
			return false
		})
	}

	if len(cfg.IRC.PostConnect.Code) > 0 {
		s.respondOnce(cfg.IRC.PostConnect.Code, func(event *irce.Event) bool {
			for _, command := range cfg.IRC.PostConnect.Commands {
				s.conn.SendRawf(command, cfg.IRC.Nick)
			}
			for _, channel := range cfg.IRC.PostConnect.AutoJoin {
				s.conn.Join(channel)
			}
			connectCallback(s.ctx, s.cfg, s)
			return true
		})
	}

	if joinChannelCallback != nil {
		s.conn.AddCallback(CodeJoin, func(e *irce.Event) {
			joinChannelCallback(e.Message(), e.Nick)
		})
	}

	err := s.conn.Connect(fmt.Sprintf("%s:%d", cfg.IRC.Server, cfg.IRC.Port))
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

func (s *service) ListUsers(channel string, callback func(users []*User)) {
	s.conn.SendRawf("NAMES %s", channel)

	allUsers := make([]*User, 0)

	s.respondUntil(CodeNamesReply, CodeEndOfNames, func(e *irce.Event) {
		users := make([]*User, 0)
		results := strings.Split(e.Message(), " ")
		for _, u := range results {
			users = append(users, UserByTrimmingStatusPrefix(u))
		}
		allUsers = append(allUsers, users...)
	}, func(e *irce.Event) {
		callback(allUsers)
	})
}

func (s *service) GetUser(channel, nick string, callback func(user *User)) {
	logger := log.Logger()
	s.conn.SendRawf("WHOIS %s", nick)

	var user *User

	s.respondUntil(CodeWhoIsReply, CodeEndOfWhoIs, func(e *irce.Event) {
		if len(e.Arguments) < 4 {
			logger.Errorf(nil, "invalid WHOIS reply: %s", e.Raw)
			return
		}

		id := e.Arguments[2]
		host := e.Arguments[3]
		user = &User{Mask: &Mask{Nick: nick, UserID: id, Host: host}}
		logger.Debugf(nil, "WHOIS(%s,%s): %s", channel, nick, user.Mask.String())
	}, func(e *irce.Event) {
		if user == nil {
			callback(nil)
			return
		}

		s.ListUsers(channel, func(users []*User) {
			for _, u := range users {
				if u.Mask.Nick == nick {
					user.Status = u.Status
					callback(user)
					return
				}
			}
			callback(nil)
		})
	})
}

func (s *service) Up(channel, nick string) {
	s.conn.Privmsgf(s.cfg.IRC.ChanServ.Recipient, s.cfg.IRC.ChanServ.UpCommand, channel, nick)
}

func (s *service) Down(channel, nick string) {
	s.conn.Privmsgf(s.cfg.IRC.ChanServ.Recipient, s.cfg.IRC.ChanServ.DownCommand, channel, nick)
}

func (s *service) Kick(channel, nick, reason string) {
	s.conn.Kick(nick, channel, reason)
}

func (s *service) Ban(channel, mask string) {
	s.conn.Mode(channel, "+b", mask)
}

func (s *service) Unban(channel, mask string) {
	s.conn.Mode(channel, "-b", mask)
}

func (s *service) Disconnect() {
	s.conn.ClearCallback("*")
	s.conn.Disconnect()
}
