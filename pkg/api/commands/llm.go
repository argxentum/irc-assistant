package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"strings"
)

const LLMCommandName = "llm"

type LLMCommand struct {
	*commandStub
}

func NewLLMCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &LLMCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *LLMCommand) Name() string {
	return LLMCommandName
}

func (c *LLMCommand) Description() string {
	return "Chat with an LLM"
}

func (c *LLMCommand) Triggers() []string {
	return []string{"llm"}
}

func (c *LLMCommand) Usages() []string {
	return []string{"%s <prompt>"}
}

func (c *LLMCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *LLMCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *LLMCommand) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	prompt := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), prompt)

	task := models.NewProxyLLMRequestTask(e.ReplyTarget(), e.From, "llm", prompt)
	if err := queue.GetProxy().Publish(task); err != nil {
		logger.Errorf(e, "error publishing LLM request, %s", err)
		return
	}
}
