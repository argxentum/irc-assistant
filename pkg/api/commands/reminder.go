package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
	"time"
)

const reminderCommandName = "reminder"

type reminderCommand struct {
	*commandStub
}

func NewReminderCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &reminderCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
	}
}

func (c *reminderCommand) Name() string {
	return reminderCommandName
}

func (c *reminderCommand) Description() string {
	return "Creates a reminder message to be delivered after the given duration."
}

func (c *reminderCommand) Triggers() []string {
	return []string{"reminder"}
}

func (c *reminderCommand) Usages() []string {
	return []string{"%s <duration> <message>"}
}

func (c *reminderCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *reminderCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

func (c *reminderCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	duration := tokens[1]
	message := strings.Join(tokens[2:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] @ %s, %s", c.Name(), e.From, e.ReplyTarget(), duration, message)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		c.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", c.cfg.Commands.Prefix, registry.Command(reminderCommandName).Triggers()[0])))
		return
	}

	task := models.NewReminderTask(time.Now().Add(seconds), e.From, e.ReplyTarget(), message)
	err = firestore.Get().AddTask(task)
	if err != nil {
		logger.Errorf(e, "error adding task, %s", err)
		return
	}

	c.Replyf(e, "reminder set for %s", style.Bold(elapse.TimeDescription(task.DueAt)))
}
