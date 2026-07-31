package irc

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"crypto/tls"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	irce "github.com/thoj/go-ircevent"
)

type ChannelStatus string

const (
	ChannelStatusOperator     ChannelStatus = "@"
	ChannelStatusHalfOperator ChannelStatus = "%"
	ChannelStatusVoice        ChannelStatus = "+"
	ChannelStatusNone         ChannelStatus = ""
)

type User struct {
	Mask   *Mask
	Status ChannelStatus
}

type BanEntry struct {
	Mask  string
	SetBy string
	SetAt *time.Time
}

func UserByTrimmingStatusPrefix(u string) *User {
	if strings.HasPrefix(u, string(ChannelStatusOperator)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusOperator))}, Status: ChannelStatusOperator}
	} else if strings.HasPrefix(u, string(ChannelStatusHalfOperator)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusHalfOperator))}, Status: ChannelStatusHalfOperator}
	} else if strings.HasPrefix(u, string(ChannelStatusVoice)) {
		return &User{Mask: &Mask{Nick: strings.TrimPrefix(u, string(ChannelStatusVoice))}, Status: ChannelStatusVoice}
	} else {
		return &User{Mask: &Mask{Nick: u}, Status: ChannelStatusNone}
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
	case ChannelStatusNone:
		return true
	}
	return false
}

type IRC interface {
	Connect(cfg *config.Config, connectCallback func(ctx context.Context, cfg *config.Config, i IRC), joinChannelCallback func(channel string, mask *Mask)) error
	Listen(ech chan *Event)
	Join(channel string)
	Part(channel string)
	SendMessage(target, message string)
	SendMessages(target string, messages []string)
	GetUser(channel, nick string, callback func(user *User))
	ListUsers(channel string, callback func(users []*User))
	ListUsersByMask(channel, mask string, callback func(users []*User))
	Up(channel, nick string)
	Down(channel, nick string)
	Voice(channel, nick string)
	Mute(channel, nick string)
	Kick(channel, nick, reason string)
	Ban(channel, mask string)
	Unban(channel, mask string)
	ListBans(channel string, callback func(bans []*BanEntry))
	GetTopic(channel string, callback func(topic string))
	SetTopic(channel, topic string)
	Disconnect()
}

const maxMessageLength = 400

func NewIRC(ctx context.Context) IRC {
	return &service{
		ctx: ctx,
	}
}

type service struct {
	ctx           context.Context
	cfg           *config.Config
	conn          *irce.Connection
	requests      *ircRequestManager
	ech           chan *Event
	recoverNeeded bool
}

func (s *service) Connect(cfg *config.Config, connectCallback func(ctx context.Context, cfg *config.Config, irc IRC), joinChannelCallback func(channel string, mask *Mask)) error {
	s.cfg = cfg

	s.conn = irce.IRC(cfg.IRC.Nick, cfg.IRC.Username)
	s.conn.RealName = cfg.IRC.RealName
	s.conn.Debug = false
	s.conn.VerboseCallbackHandler = false
	s.requests = newIRCRequestManager(s.conn, s.conn.SendRaw, ircRequestTimeout)

	if cfg.IRC.TLS {
		s.conn.UseTLS = cfg.IRC.TLS
		s.conn.TLSConfig = newTLSConfig(cfg.IRC.Server)
	}

	if len(cfg.IRC.NickServ.Password) > 0 {
		s.respondOnce(CodeNickInUse, func(event *irce.Event) bool {
			log.Logger().Debugf(nil, "nick %s already in use, marking as recover needed", cfg.IRC.Nick)
			s.recoverNeeded = true
			return true
		})

		s.respondOnce(CodeNickReserved, func(event *irce.Event) bool {
			log.Logger().Debugf(nil, "nick %s already in use, need to release", cfg.IRC.Nick)
			s.conn.Privmsgf(cfg.IRC.NickServ.Recipient, cfg.IRC.NickServ.ReleaseCommand, cfg.IRC.Nick, cfg.IRC.NickServ.Password)
			return true
		})

		s.respondOnce(CodeEndOfMotd, func(event *irce.Event) bool {
			if s.recoverNeeded {
				log.Logger().Debugf(nil, "reached end of MOTD and recover is needed, trying to recover %s with NickServ", cfg.IRC.Nick)
				s.conn.Privmsgf(cfg.IRC.NickServ.Recipient, cfg.IRC.NickServ.RecoverCommand, cfg.IRC.Nick, cfg.IRC.NickServ.Password)
			}
			return true
		})

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
			m := ParseMask(e.Source)
			if m == nil {
				log.Logger().Warningf(nil, "ignoring JOIN event with malformed source mask %q", e.Source)
				return
			}
			joinChannelCallback(e.Message(), m)
		})
	}

	err := s.conn.Connect(fmt.Sprintf("%s:%d", cfg.IRC.Server, cfg.IRC.Port))
	if err != nil {
		return err
	}

	return nil
}

func newTLSConfig(serverName string) *tls.Config {
	return &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}
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
	allUsers := make([]*User, 0)
	s.requests.run(requestKey("NAMES", channel), fmt.Sprintf("NAMES %s", channel), map[string]func(*irce.Event) bool{
		CodeNamesReply: func(e *irce.Event) bool {
			if !eventArgumentEquals(e, 2, channel) {
				return false
			}
			for _, name := range strings.Fields(e.Message()) {
				allUsers = append(allUsers, UserByTrimmingStatusPrefix(name))
			}
			return false
		},
		CodeEndOfNames: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, channel)
		},
	}, func(timedOut bool) {
		if timedOut {
			log.Logger().Warningf(nil, "timed out listing users in %s", channel)
			callback([]*User{})
			return
		}
		callback(allUsers)
	})
}

func (s *service) ListUsersByMask(channel, mask string, callback func(users []*User)) {
	matchingUsers := make([]*User, 0)
	m := ParseMask(mask)
	if m == nil {
		log.Logger().Warningf(nil, "cannot list users with invalid mask %q", mask)
		callback(matchingUsers)
		return
	}

	s.requests.run(requestKey("WHO", channel), fmt.Sprintf("WHO %s", channel), map[string]func(*irce.Event) bool{
		CodeWhoReply: func(e *irce.Event) bool {
			if !eventArgumentEquals(e, 1, channel) || len(e.Arguments) < 7 {
				return false
			}

			um := &Mask{
				Nick:   e.Arguments[5],
				UserID: e.Arguments[2],
				Host:   e.Arguments[3],
			}

			if m.Matches(um) {
				flags := e.Arguments[6]
				var status ChannelStatus
				if strings.Contains(flags, string(ChannelStatusOperator)) {
					status = ChannelStatusOperator
				} else if strings.Contains(flags, string(ChannelStatusHalfOperator)) {
					status = ChannelStatusHalfOperator
				} else if strings.Contains(flags, string(ChannelStatusVoice)) {
					status = ChannelStatusVoice
				} else {
					status = ChannelStatusNone
				}

				matchingUsers = append(matchingUsers, &User{Mask: um, Status: status})
			}
			return false
		},
		CodeEndOfWho: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, channel)
		},
	}, func(timedOut bool) {
		if timedOut {
			log.Logger().Warningf(nil, "timed out listing users by mask in %s", channel)
			callback([]*User{})
			return
		}
		callback(matchingUsers)
	})
}

func (s *service) GetUser(channel, nick string, callback func(user *User)) {
	logger := log.Logger()
	var user *User

	s.requests.run(requestKey("WHOIS", nick), fmt.Sprintf("WHOIS %s", nick), map[string]func(*irce.Event) bool{
		CodeWhoIsReply: func(e *irce.Event) bool {
			if !eventArgumentEquals(e, 1, nick) {
				return false
			}
			if len(e.Arguments) < 4 {
				logger.Errorf(nil, "invalid WHOIS reply: %s", e.Raw)
				return false
			}

			user = &User{Mask: &Mask{Nick: nick, UserID: e.Arguments[2], Host: e.Arguments[3]}}
			logger.Debugf(nil, "WHOIS(%s,%s): %s", channel, nick, user.Mask.String())
			return false
		},
		CodeEndOfWhoIs: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, nick)
		},
	}, func(timedOut bool) {
		if timedOut {
			logger.Warningf(nil, "timed out getting user %s in %s", nick, channel)
			callback(nil)
			return
		}
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

func (s *service) Voice(channel, nick string) {
	s.conn.Mode(channel, "+v", nick)
}

func (s *service) Mute(channel, nick string) {
	s.conn.Mode(channel, "-v", nick)
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

func (s *service) ListBans(channel string, callback func(bans []*BanEntry)) {
	bans := make([]*BanEntry, 0)

	s.requests.run(requestKey("MODE+B", channel), fmt.Sprintf("MODE %s +b", channel), map[string]func(*irce.Event) bool{
		CodeBanListReply: func(e *irce.Event) bool {
			if !eventArgumentEquals(e, 1, channel) || len(e.Arguments) < 3 {
				return false
			}

			entry := &BanEntry{Mask: e.Arguments[2]}
			if len(e.Arguments) >= 4 {
				entry.SetBy = e.Arguments[3]
			}
			if len(e.Arguments) >= 5 {
				ts, err := strconv.ParseInt(strings.TrimPrefix(e.Arguments[4], ":"), 10, 64)
				if err == nil {
					t := time.Unix(ts, 0)
					entry.SetAt = &t
				}
			}
			bans = append(bans, entry)
			return false
		},
		CodeEndOfBanList: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, channel)
		},
	}, func(timedOut bool) {
		if timedOut {
			log.Logger().Warningf(nil, "timed out listing bans in %s", channel)
			callback([]*BanEntry{})
			return
		}
		callback(bans)
	})
}

func (s *service) GetTopic(channel string, callback func(topic string)) {
	topic := ""
	s.requests.run(requestKey("TOPIC", channel), fmt.Sprintf("TOPIC %s", channel), map[string]func(*irce.Event) bool{
		CodeTopicReply: func(e *irce.Event) bool {
			if !eventArgumentEquals(e, 1, channel) {
				return false
			}
			topic = e.Message()
			return true
		},
		CodeNoTopic: func(e *irce.Event) bool {
			return eventArgumentEquals(e, 1, channel)
		},
	}, func(timedOut bool) {
		if timedOut {
			log.Logger().Warningf(nil, "timed out getting topic in %s", channel)
		}
		callback(topic)
	})
}

func (s *service) SetTopic(channel, topic string) {
	s.conn.SendRawf("TOPIC %s :%s", channel, topic)
}

func (s *service) Disconnect() {
	s.conn.ClearCallback("*")
	s.conn.Disconnect()
}
