package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
)

const AuthCommandName = "auth"

type AuthCommand struct {
	*commandStub
}

func NewAuthCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &AuthCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusHalfOperator),
	}
}

func (c *AuthCommand) Name() string {
	return AuthCommandName
}

func (c *AuthCommand) Description() string {
	return "Authenticate for web dashboard access"
}

func (c *AuthCommand) Triggers() []string {
	return []string{"auth"}
}

func (c *AuthCommand) Usages() []string {
	return []string{"%s <channel>"}
}

func (c *AuthCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *AuthCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())
	if len(tokens) > 1 {
		channel = tokens[1]
	}
	c.commandStub.authorizer.IsAuthorized(e, channel, callback)
}

func (c *AuthCommand) CanExecute(e *irc.Event) bool {
	if !e.IsPrivateMessage() {
		return false
	}
	return c.isCommandEventValid(c, e, 1)
}

func (c *AuthCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	channel := tokens[1]

	if !irc.IsChannel(channel) {
		c.Replyf(e, "Invalid channel: %s", channel)
		return
	}

	nick, _ := e.Sender()
	logger.Infof(e, "⚡ %s [%s] channel: %s", c.Name(), nick, channel)

	token, err := models.NewAuthToken(nick, channel)
	if err != nil {
		logger.Errorf(e, "error generating auth token: %s", err)
		c.Replyf(e, "Error generating auth token.")
		return
	}

	fs := firestore.Get()
	if err := fs.CreateAuthToken(token); err != nil {
		logger.Errorf(e, "error storing auth token: %s", err)
		c.Replyf(e, "Error generating auth token.")
		return
	}

	url := fmt.Sprintf("%s/dashboard/%s", c.cfg.Web.ExternalRootURL, token.Token)
	c.Replyf(e, "Dashboard access: %s (expires in %s, single use)", url, elapse.FutureTimeDescriptionConcise(token.ExpiresAt))
}
