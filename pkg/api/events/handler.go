package events

import (
	"assistant/pkg/api/commands"
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type Handler interface {
	FindMatchingCommand(e *irc.Event) commands.Command
	Handle(e *irc.Event)
}

type handler struct {
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
	for _, f := range eh.registry.CommandsSortedForProcessing() {
		if f.CanExecute(e) {
			return f
		}
	}
	return nil
}

func (eh *handler) Handle(e *irc.Event) {
	logger := log.Logger()
	logger.Default(e, e.Raw)

	switch e.Code {
	case irc.CodeInvite:
		// if the sender of invite is the owner or an admin, join the channel
		sender, _ := e.Sender()
		if sender == eh.cfg.IRC.Owner || slices.Contains(eh.cfg.IRC.Admins, sender) {
			channel := e.Arguments[1]
			eh.irc.Join(channel)
		}
	case irc.CodePrivateMessage:
		tokens := commands.Tokens(e.Message())

		if !e.IsPrivateMessage() {
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
		}
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
