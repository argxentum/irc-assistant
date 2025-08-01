package events

import (
	"assistant/pkg/api/commands"
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
)

const userCommandRateLimitDuration = 1250 * time.Millisecond

type Handler interface {
	FindMatchingCommand(e *irc.Event) commands.Command
	Handle(e *irc.Event)
}

type handler struct {
	sync.Mutex
	ctx      context.Context
	cfg      *config.Config
	irc      irc.IRC
	registry commands.CommandRegistry
}

func NewHandler(ctx context.Context, cfg *config.Config, irc irc.IRC) Handler {
	eh := &handler{
		ctx:      ctx,
		cfg:      cfg,
		irc:      irc,
		registry: commands.LoadCommandRegistry(ctx, cfg, irc),
	}

	return eh
}

func (eh *handler) FindMatchingCommand(e *irc.Event) commands.Command {
	logger := log.Logger()

	if eh.isUserCommandRateLimited(e) {
		eh.incrementUserRateLimitCounter(e)
		logger.Warningf(e, "ignoring input from %s, rate limit exceeded", e.From)
		return nil
	} else {
		if isUserMask(e.Source) {
			mask := irc.ParseMask(e.Source)
			eh.Lock()
			if _, ok := rateLimitCounter[mask.Host]; ok {
				logger.Debugf(e, "resetting rate limit counter for %s", mask.Host)
				rateLimitCounter[mask.Host] = 0
			}
			eh.Unlock()
		}
	}

	channelDisabled := make([]string, 0)
	if !e.IsPrivateMessage() {
		if ch, _ := repository.GetChannel(e, e.ReplyTarget()); ch != nil {
			channelDisabled = ch.DisabledCommands
		}
	}

	for _, f := range eh.registry.CommandsSortedForProcessing() {
		if !slices.Contains(channelDisabled, f.Name()) && f.CanExecute(e) {
			eh.updateUserCommandHistory(e)
			return f
		}
	}

	return nil
}

func (eh *handler) Handle(e *irc.Event) {
	logger := log.Logger()
	logger.Default(e, e.Raw)
	isPrivate := e.IsPrivateMessage()

	if len(e.Arguments) > 0 {
		substitutions := false
		tokens := commands.Tokens(e.Arguments[len(e.Arguments)-1])
		for i, a := range tokens {
			if v, ok := commandSubstitutions[a]; ok {
				substitutions = true
				logger.Debugf(e, "substituting command %s with %s", a, v)
				tokens[i] = v
			}
		}
		if substitutions {
			e.Arguments[len(e.Arguments)-1] = strings.Join(tokens, " ")
		}
	}

	switch e.Code {
	case irc.CodeQuit:
		if strings.ToLower(e.Arguments[0]) == irc.MessageNetSplit {
			logger.Errorf(e, "net split detected, user leaving: %s (%s)", e.From, e.Source)
		} else if strings.ToLower(e.Arguments[0]) == irc.MessageServerShuttingDown {
			logger.Criticalf(e, "server shutting down, user leaving: %s (%s)", e.From, e.Source)
		}
	case irc.CodeError:
		if strings.HasPrefix(strings.ToLower(e.Arguments[0]), irc.MessageClosingLink) && strings.Contains(strings.ToLower(e.Arguments[0]), irc.MessageServerShuttingDown) {
			logger.Alertf(e, "server shutting down, attempting reconnect in %d seconds", eh.cfg.IRC.ReconnectDelay)
			// todo
		}
	case irc.CodeInvite:
		// if the sender of invite is the owner or an admin, join the channel
		sender, _ := e.Sender()
		if sender == eh.cfg.IRC.Owner || slices.Contains(eh.cfg.IRC.Admins, sender) {
			channel := e.Arguments[1]
			eh.irc.Join(channel)
		}
	case irc.CodeNickChange:
		if e.IsPrivateMessage() {
			logger.Debugf(e, "ignoring nick change event in private message")
			return
		}
		oldMask := e.Mask()
		newMask := e.Mask()
		newMask.Nick = e.Message()
		if err := repository.CreateUserFromNickChange(e, oldMask, newMask); err != nil {
			logger.Errorf(e, "unable to create user for nick change event, %v", err)
		}
	case irc.CodePrivateMessage:
		tokens := commands.Tokens(e.Message())

		if eh.isTemporarilyIgnoredUser(e) {
			logger.Debugf(e, "ignoring message from temporarily ignored user %s", e.Source)
			return
		}

		if !isPrivate {
			eh.resetChannelInactivityTimeout(e)

			bannedWords := eh.bannedWordsInMessage(e, tokens)
			if len(bannedWords) > 0 {
				label := "word"
				if len(bannedWords) > 1 {
					label = "words"
				}
				eh.irc.Kick(e.ReplyTarget(), e.From, fmt.Sprintf("banned %s: %s", label, strings.Join(bannedWords, ", ")))
				return
			}
		}

		if slices.Contains(eh.cfg.Ignore.Users, e.From) {
			logger.Debugf(e, "ignoring message from %s", e.From)
			return
		}

		if f := eh.FindMatchingCommand(e); f != nil {
			f.Authorizer().IsAuthorized(e, e.ReplyTarget(), func(authorized bool) {
				if !authorized {
					logger.Warningf(e, "unauthorized attempt by %s to use %s", e.From, tokens[0])

					if strings.HasPrefix(tokens[0], eh.cfg.Commands.Prefix) {
						f.Replyf(e, "You are not authorized to use %s.", style.Bold(strings.TrimPrefix(tokens[0], eh.cfg.Commands.Prefix)))
					} else {
						f.Replyf(e, "You are not authorized to perform that command.")
					}
					return
				}

				go f.Execute(e)
			})
		} else if !isPrivate && len(e.Message()) > 0 {
			u, err := repository.GetUserByNick(e, e.ReplyTarget(), e.From, true)
			if err != nil {
				logger.Errorf(e, "unable to find or create user in order to update recent user messages, %s", err)
			} else {
				if err = repository.AddRecentUserMessage(e, u); err != nil {
					logger.Errorf(e, "unable to add recent user message, %s", err)
				}
			}
		}
	}
}

var messageHistory = make(map[string]time.Time)
var rateLimitCounter = make(map[string]int)
var temporarilyIgnoredUserMasks = make(map[string]int64)

func (eh *handler) updateUserCommandHistory(e *irc.Event) {
	if isUserMask(e.Source) {
		mask := irc.ParseMask(e.Source)
		eh.Lock()
		messageHistory[mask.Host] = time.Now()
		eh.Unlock()
	}
}

func (eh *handler) isUserCommandRateLimited(e *irc.Event) bool {
	if isUserMask(e.Source) {
		mask := irc.ParseMask(e.Source)
		if c, ok := messageHistory[mask.Host]; ok {
			if time.Since(c) < userCommandRateLimitDuration {
				eh.Lock()
				messageHistory[mask.Host] = time.Now()
				eh.Unlock()
				return true
			}
		}
	}

	return false
}

const maxRateLimitedMessageCount = 5
const temporarilyIgnoredUserDuration = 1 * time.Minute

func (eh *handler) incrementUserRateLimitCounter(e *irc.Event) {
	logger := log.Logger()

	if isUserMask(e.Source) {
		mask := irc.ParseMask(e.Source)
		eh.Lock()
		if _, ok := rateLimitCounter[mask.Host]; !ok {
			rateLimitCounter[mask.Host] = 0
		}
		rateLimitCounter[mask.Host]++

		logger.Debugf(e, "rate limit counter for %s: %d", mask.Host, rateLimitCounter[mask.Host])

		if rateLimitCounter[mask.Host] >= maxRateLimitedMessageCount {
			logger.Debugf(e, "adding temporarily ignored user: %s", mask.Host)
			temporarilyIgnoredUserMasks[mask.Host] = time.Now().Add(temporarilyIgnoredUserDuration).Unix()
		}

		eh.Unlock()
	}
}

func (eh *handler) isTemporarilyIgnoredUser(e *irc.Event) bool {
	logger := log.Logger()

	if !isUserMask(e.Source) {
		return false
	}

	mask := irc.ParseMask(e.Source)

	if t, ok := temporarilyIgnoredUserMasks[mask.Host]; ok {
		if time.Now().Unix() < t {
			logger.Debugf(e, "user %s remains temporarily ignored until %d", mask.Host, t)
			return true
		} else {
			logger.Debugf(e, "user is no longer temporarily ignored: %s", mask.Host)
			eh.Lock()
			delete(temporarilyIgnoredUserMasks, mask.Host)
			delete(rateLimitCounter, mask.Host)
			eh.Unlock()
		}
	}

	return false
}

var userMaskRegex = regexp.MustCompile(`^[^!]+![^@]+@.+$`)

func isUserMask(source string) bool {
	return userMaskRegex.MatchString(source)
}

func (eh *handler) resetChannelInactivityTimeout(e *irc.Event) {
	fs := firestore.Get()
	logger := log.Logger()

	channel, err := fs.Channel(e.ReplyTarget())
	if err != nil {
		logger.Errorf(e, "error retrieving channel, %s", err)
		return
	}

	if channel == nil {
		logger.Errorf(e, "channel %s does not exist", e.ReplyTarget())
		return
	}

	duration, err := elapse.ParseDuration(channel.InactivityDuration)
	if err != nil {
		logger.Errorf(e, "error parsing default inactivity duration, %s", err)
	}

	err = fs.SetPersistentChannelTaskDue(e.ReplyTarget(), models.ChannelInactivityTaskID, duration)
	if err != nil {
		logger.Errorf(e, "error updating persistent channel task, %s", err)
	}
}

func (eh *handler) bannedWordsInMessage(e *irc.Event, tokens []string) []string {
	logger := log.Logger()

	wordMap := make(map[string]bool)

	for _, token := range tokens {
		stripped := strings.TrimFunc(strings.ToLower(token), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if eh.ctx.Session().IsBannedWord(e.ReplyTarget(), stripped) {
			logger.Warningf(e, "banned word detected: %s", stripped)
			wordMap[stripped] = true
		}
	}

	words := make([]string, 0)
	for word := range wordMap {
		words = append(words, word)
	}

	return words
}
