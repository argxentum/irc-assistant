package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"strings"
)

const SourceCommandName = "source"

type SourceCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewSourceCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &SourceCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *SourceCommand) Name() string {
	return SourceCommandName
}

func (c *SourceCommand) Description() string {
	return "Displays source bias and credibility information."
}

func (c *SourceCommand) Triggers() []string {
	return []string{"source", "bias"}
}

func (c *SourceCommand) Usages() []string {
	return []string{"%s <name>", "%s <url>"}
}

func (c *SourceCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SourceCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *SourceCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")

	source, err := repository.FindSource(input)
	if err != nil {
		log.Logger().Errorf(e, "error finding source: %s", err)
		c.Replyf(e, "Unable to find source details for %s", style.Bold(input))
		return
	}

	if source != nil {
		SendSource(c.commandStub, e, source)
		return
	} else {
		c.Replyf(e, "No source details found for %s", style.Bold(input))
		return
	}
}

func SendSource(cs *commandStub, e *irc.Event, source *models.Source) {
	messages := make([]string, 0)
	messages = append(messages, repository.FullSourceSummary(source))

	if len(source.Reviews) > 0 {
		messages = append(messages, source.Reviews[0])
	}

	cs.SendMessages(e, e.ReplyTarget(), messages)
}
