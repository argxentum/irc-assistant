package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"strings"
)

const SayCommandName = "say"

type SayCommand struct {
	*commandStub
}

func NewSayCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &SayCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *SayCommand) Name() string {
	return SayCommandName
}

func (c *SayCommand) Description() string {
	return "Sends a message to the specified channel."
}

func (c *SayCommand) Triggers() []string {
	return []string{"say"}
}

func (c *SayCommand) Usages() []string {
	return []string{"%s <channel> <message>"}
}

func (c *SayCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SayCommand) CanExecute(e *irc.Event) bool {
	if !c.isCommandEventValid(c, e, 3) {
		return false
	}

	tokens := Tokens(e.Message())
	return irc.IsChannel(tokens[1])
}

func (c *SayCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	channel := tokens[1]
	message := strings.Join(tokens[2:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s %s", c.Name(), e.From, e.ReplyTarget(), channel, message)

	c.SendMessage(e, channel, message)
}
