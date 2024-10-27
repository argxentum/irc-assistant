package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"regexp"
	"strings"
)

const karmaSetCommandName = "karmaSet"
const maxKarmaReasonLength = 128

type karmaSetCommand struct {
	*commandStub
}

func NewKarmaSetCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &karmaSetCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *karmaSetCommand) Name() string {
	return karmaSetCommandName
}

func (c *karmaSetCommand) Description() string {
	return "Updates the karma value for the given user."
}

func (c *karmaSetCommand) Triggers() []string {
	return []string{}
}

func (c *karmaSetCommand) Usages() []string {
	return []string{"<user>++ [<reason>]", "<user>-- [<reason>]"}
}

func (c *karmaSetCommand) AllowedInPrivateMessages() bool {
	return false
}

func (c *karmaSetCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0) && (strings.Contains(e.Message(), "++") || strings.Contains(e.Message(), "--"))
}

var karmaRegex = regexp.MustCompile(`(?i)(.*?)\s*(\+\+|--)(?:\s*,?\s+(.*))?`)

func (c *karmaSetCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	matches := karmaRegex.FindStringSubmatch(e.Message())
	if len(matches) < 3 {
		logger.Debugf(e, "invalid karma message: %s", e.Message())
		return
	}

	to := strings.TrimSpace(matches[1])
	if len(to) == 0 {
		logger.Debugf(e, "invalid karma target: %s", e.Message())
		return
	}

	if strings.ToLower(e.From) == strings.ToLower(to) {
		logger.Debugf(e, "cannot update own karma: %s", e.Message())
		c.Replyf(e, "You cannot update your own karma.")
		return
	}

	c.authorizer.GetUser(e.ReplyTarget(), to, func(user *irc.User) {
		if user == nil {
			logger.Debugf(e, "ignoring invalid karma target: %s", to)
			return
		}

		op := strings.TrimSpace(matches[2])
		if len(op) == 0 {
			logger.Debugf(e, "invalid karma operation: %s", e.Message())
			return
		}

		reason := ""
		if len(matches) > 3 {
			reason = strings.TrimSpace(matches[3])
		}
		if len(reason) > maxKarmaReasonLength {
			reason = reason[:maxKarmaReasonLength]
		}

		log.Logger().Infof(e, "⚡ %s [%s/%s] %s %s %s", c.Name(), e.From, e.ReplyTarget(), to, op, reason)

		fs := firestore.Get()
		karma, err := fs.AddKarmaHistory(c.ctx, e.ReplyTarget(), e.From, to, op, reason)
		if err != nil {
			logger.Errorf(e, "error updating karma: %s", err)
			return
		}

		c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s now has a karma of %s.", style.Bold(to), style.Bold(fmt.Sprintf("%d", karma))))
	})
}
