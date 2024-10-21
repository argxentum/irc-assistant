package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"time"
)

const uptimeCommandName = "uptime"

type uptimeCommand struct {
	*commandStub
}

func NewUptimeCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &uptimeCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *uptimeCommand) Name() string {
	return uptimeCommandName
}

func (c *uptimeCommand) Description() string {
	return "Displays uptime information."
}

func (c *uptimeCommand) Triggers() []string {
	return []string{"uptime"}
}

func (c *uptimeCommand) Usages() []string {
	return []string{"%s"}
}

func (c *uptimeCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *uptimeCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *uptimeCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())

	startedAt := c.ctx.Session().StartedAt
	elapsed := time.Since(startedAt)
	years := int(elapsed.Hours() / 24 / 365)
	elapsed -= time.Duration(years) * 24 * 365 * time.Hour
	months := int(elapsed.Hours() / 24 / 30)
	elapsed -= time.Duration(months) * 24 * 30 * time.Hour
	days := int(elapsed.Hours() / 24)
	elapsed -= time.Duration(days) * 24 * time.Hour
	hours := int(elapsed.Hours())
	elapsed -= time.Duration(hours) * time.Hour
	minutes := int(elapsed.Minutes())
	elapsed -= time.Duration(minutes) * time.Minute
	seconds := int(elapsed.Seconds())

	response := ""
	if years > 0 {
		plural := ""
		if years > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d year%s, ", years, plural)
	}
	if months > 0 || years > 0 {
		plural := ""
		if months > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d month%s, ", months, plural)
	}
	if days > 0 || months > 0 || years > 0 {
		plural := ""
		if days > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d day%s, ", days, plural)
	}
	if hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if hours > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d hour%s, ", hours, plural)
	}
	if minutes > 0 || hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if minutes > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d minute%s, ", minutes, plural)
	}
	if seconds > 0 || minutes > 0 || hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if seconds > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d second%s", seconds, plural)
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("Uptime: %s", style.Bold(response)))
}
