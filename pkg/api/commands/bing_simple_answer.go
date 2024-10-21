package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

type bingSimpleAnswerCommand struct {
	*commandStub
	triggers    []string
	usages      []string
	description string
	subject     string
	query       string
	reply       string
	footnote    string
	minTokens   int
	retriever   retriever.DocumentRetriever
}

func NewBingSimpleAnswerCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC, triggers []string, usages []string, description, subject, query, reply, footnote string, minTokens int) Command {
	return &bingSimpleAnswerCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleUnprivileged, irc.ChannelStatusNormal),
		triggers:    triggers,
		usages:      usages,
		description: description,
		subject:     subject,
		query:       query,
		reply:       reply,
		footnote:    footnote,
		minTokens:   minTokens,
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *bingSimpleAnswerCommand) Name() string {
	return fmt.Sprintf("bing/simple/%s", c.subject)
}

func (c *bingSimpleAnswerCommand) Description() string {
	return c.description
}

func (c *bingSimpleAnswerCommand) Triggers() []string {
	return c.triggers
}

func (c *bingSimpleAnswerCommand) Usages() []string {
	return c.usages
}

func (c *bingSimpleAnswerCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *bingSimpleAnswerCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, c.minTokens)
}

func (c *bingSimpleAnswerCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), c.subject)

	tokens := Tokens(e.Message())
	input := ""
	if len(tokens) > 0 {
		input = strings.Join(tokens[1:], " ")
	}
	query := url.QueryEscape(c.query)
	if len(input) > 0 {
		query = url.QueryEscape(fmt.Sprintf(c.query, input))
	}

	doc, err := c.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), 3000)
	if err != nil {
		logger.Warningf(e, "error fetching bing search results for %s: %s", input, err)
		c.Replyf(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	node := doc.Find("ol#b_results li.b_ans").First()
	label := node.Find("div.b_focusLabel").First().Text()
	answer1 := node.Find("div.b_focusTextLarge").First().Text()
	answer2 := node.Find("div.b_focusTextMedium").First().Text()
	secondary1 := node.Find("div.b_secondaryFocus").First().Text()
	secondary2 := node.Find("li.b_secondaryFocus").First().Text()

	label = strings.TrimSpace(label)
	answer := strings.TrimSpace(coalesce(answer1, answer2))
	secondary := strings.TrimSpace(coalesce(secondary1, secondary2))

	if len(label) == 0 || len(answer) == 0 || len(secondary) == 0 {
		logger.Warningf(e, "error parsing bing search results for %s", input)
		c.Replyf(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf(c.reply, label, style.Bold(answer), secondary))

	if len(c.footnote) > 0 {
		c.SendMessage(e, e.ReplyTarget(), c.footnote)
	}
}
