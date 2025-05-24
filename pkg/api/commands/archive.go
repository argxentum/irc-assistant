package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
)

const ArchiveCommandName = "archive"

type ArchiveCommand struct {
	*commandStub
}

func NewArchiveCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ArchiveCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *ArchiveCommand) Name() string {
	return ArchiveCommandName
}

func (c *ArchiveCommand) Description() string {
	return "Submits a URL to archive.today."
}

func (c *ArchiveCommand) Triggers() []string {
	return []string{"archive", "a"}
}

func (c *ArchiveCommand) Usages() []string {
	return []string{"%s <url>"}
}

func (c *ArchiveCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ArchiveCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

const submissionURL = "https://archive.today/submit/?url=%s"

func (c *ArchiveCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	source := tokens[1]
	if !urlRegex.MatchString(source) {
		logger.Debugf(e, "invalid URL: %s", source)
		c.Replyf(e, "Sorry, but I can't archive %s", source)
		return
	}

	redirect := fmt.Sprintf(submissionURL, url.PathEscape(source))
	s, err := repository.GetOrCreateShortcut(source, redirect)
	if err != nil {
		logger.Errorf(e, "failed to create shortcut %s: %v", redirect, err)
		c.Replyf(e, "Sorry, but I can't archive %s", source)
		return
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf(shortcutURLPattern, c.cfg.Web.ExternalRootURL)+s.ID)
}
