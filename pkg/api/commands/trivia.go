package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/modes"
	"assistant/pkg/api/style"
	"assistant/pkg/api/trivia"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const TriviaCommandName = "trivia"

type TriviaCommand struct {
	*commandStub
}

func NewTriviaCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &TriviaCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *TriviaCommand) Name() string        { return TriviaCommandName }
func (c *TriviaCommand) Description() string { return "Start a trivia game" }
func (c *TriviaCommand) Triggers() []string  { return []string{"trivia"} }
func (c *TriviaCommand) Usages() []string {
	return []string{"%s", "%s random", "%s <category>", "%s cancel"}
}

func (c *TriviaCommand) AllowedInPrivateMessages() bool { return false }

func (c *TriviaCommand) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	tokens := Tokens(e.Message())
	if len(tokens) > 1 && strings.EqualFold(tokens[1], "cancel") {
		cancelAuth := newCommandAuthorizer(c.ctx, c.cfg, c.irc, RoleAdmin, irc.ChannelStatusHalfOperator)
		cancelAuth.IsAuthorized(e, channel, callback)
		return
	}
	callback(true)
}

func (c *TriviaCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *TriviaCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", c.Name(), e.From, e.ReplyTarget())

	mgr := modes.GetManager()

	if mgr.IsActive(e.ReplyTarget()) {
		c.Replyf(e, "A game is already in progress.")
		return
	}

	cooldown := c.triviaCooldown()
	if remaining := mgr.CooldownRemaining(e.ReplyTarget(), "trivia", cooldown); remaining > 0 {
		c.Replyf(e, "Trivia is on cooldown. Try again in %s.", elapse.FutureTimeDescriptionConcise(time.Now().Add(remaining)))
		return
	}

	tokens := Tokens(e.Message())
	if len(tokens) > 1 && strings.EqualFold(tokens[1], "cancel") {
		if !mgr.IsActive(e.ReplyTarget()) {
			c.Replyf(e, "No active game to cancel.")
			return
		}
		mgr.Deactivate(e.ReplyTarget())
		return
	}

	if len(tokens) > 1 && strings.EqualFold(tokens[1], "random") {
		go c.startWithCategory(e, "", "")
		return
	}

	if len(tokens) > 1 {
		term := strings.Join(tokens[1:], " ")
		categoryID, categoryName := c.matchCategory(term)
		if categoryID != "" {
			go c.startWithCategory(e, categoryID, categoryName)
			return
		}
		c.Replyf(e, "Unknown category \"%s\". Try: %s, or just %s to set up a game.", term, style.Italics("!trivia random"), style.Italics("!trivia"))
		return
	}

	setupURL := c.cfg.Web.ExternalRootURL + "/trivia/" + url.PathEscape(e.ReplyTarget())
	c.SendMessage(e, e.From, fmt.Sprintf("Set up your trivia game: %s", setupURL))
	c.Replyf(e, "Check your DMs for the trivia setup link!")
}

func (c *TriviaCommand) startWithCategory(e *irc.Event, categoryID, categoryName string) {
	logger := log.Logger()

	count := c.cfg.Trivia.DefaultCount
	if count <= 0 {
		count = 5
	}

	questions, err := trivia.FetchQuestions(count, categoryID, "")
	if err != nil {
		logger.Errorf(e, "error fetching trivia questions: %s", err)
		c.Replyf(e, "Failed to fetch trivia questions. Try again later.")
		return
	}

	if len(questions) == 0 {
		c.Replyf(e, "No trivia questions available. Try again later.")
		return
	}

	mode := modes.NewTriviaMode(e.ReplyTarget(), c.irc, c.cfg, questions)
	cooldown := c.triviaCooldown()
	if err := modes.GetManager().Activate(mode, cooldown); err != nil {
		c.Replyf(e, "%s", err)
		return
	}
}

func (c *TriviaCommand) matchCategory(term string) (string, string) {
	lower := strings.ToLower(term)
	for id, name := range trivia.Categories {
		if strings.Contains(strings.ToLower(name), lower) {
			return id, name
		}
	}
	return "", ""
}

func (c *TriviaCommand) triviaCooldown() time.Duration {
	if c.cfg.Trivia.Cooldown != "" {
		if d, err := time.ParseDuration(c.cfg.Trivia.Cooldown); err == nil {
			return d
		}
	}
	return 0
}
