package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"strings"
)

const CommunityNoteEditCommandName = "edit_community_note"

type CommunityNoteEditCommand struct {
	*commandStub
}

func NewCommunityNoteEditCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &CommunityNoteEditCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *CommunityNoteEditCommand) Name() string {
	return CommunityNoteEditCommandName
}

func (c *CommunityNoteEditCommand) Description() string {
	return "Edits a community note."
}

func (c *CommunityNoteEditCommand) Triggers() []string {
	return []string{"cnedit"}
}

func (c *CommunityNoteEditCommand) Usages() []string {
	return []string{"%s [<channel>] <id> <s/c/n> <value>"}
}

func (c *CommunityNoteEditCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *CommunityNoteEditCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 3)
}

const actionEditSource = "s"
const actionEditCounterSource = "c"
const actionEditContent = "n"

func (c *CommunityNoteEditCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channel = tokens[1]
		if len(tokens) < 3 || !irc.IsChannel(channel) {
			c.Replyf(e, "Please specify a channel: %s", style.Italics(fmt.Sprintf("%s <channel> <id>", tokens[0])))
			return
		}

		updatedTokens := make([]string, 0)
		for i, token := range tokens {
			if i != 1 {
				updatedTokens = append(updatedTokens, token)
			}
		}
		tokens = updatedTokens
	}

	id := strings.TrimSpace(tokens[1])
	action := strings.TrimSpace(strings.ToLower(tokens[2]))
	value := strings.TrimSpace(strings.Join(tokens[3:], " "))

	n, err := repository.CommunityNote(e, channel, id)
	if err != nil {
		logger.Errorf(e, "error retrieving community note: %v", err)
	}

	if n == nil {
		c.Replyf(e, "Unable to find community note %s", id)
		return
	}

	logger.Debugf(e, "updating community note %s for action %s with value %s", n.ID, action, value)

	switch action {
	case actionEditSource:
		sourceExists := false
		for _, ss := range n.Sources {
			if ss == value {
				sourceExists = true
				break
			}
		}
		if !sourceExists {
			logger.Debugf(e, "adding source %s for community note %s", value, id)
			n.Sources = append(n.Sources, value)
		} else {
			logger.Debugf(e, "source %s already exists for community note %s", value, id)
		}
	case actionEditCounterSource:
		counterSourceExists := false
		for _, cs := range n.CounterSources {
			if cs == value {
				counterSourceExists = true
				break
			}
		}
		if !counterSourceExists {
			logger.Debugf(e, "adding counter-source %s for community note %s", value, id)
			n.CounterSources = append(n.CounterSources, value)
		} else {
			logger.Debugf(e, "counter-source %s already exists for community note %s", value, id)
		}
	case actionEditContent:
		n.Content = value
	}

	err = repository.UpdateCommunityNote(e, channel, n)
	if err != nil {
		logger.Errorf(e, "error updating community note: %v", err)
		c.Replyf(e, "Sorry, I couldn't update the community note.")
		return
	}

	c.Replyf(e, "Community note %s updated.", style.Bold(n.ID))
}
