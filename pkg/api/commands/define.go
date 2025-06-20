package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const DefineCommandName = "define"

const defineAPIURL = "https://dictionaryapi.com/api/v3/references/collegiate/json/%s?key=%s"
const definePublicURL = "https://www.merriam-webster.com/dictionary/%s"

type DefineCommand struct {
	*commandStub
}

func NewDefineCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DefineCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *DefineCommand) Name() string {
	return DefineCommandName
}

func (c *DefineCommand) Description() string {
	return "Retrieves the definition of a word."
}

func (c *DefineCommand) Triggers() []string {
	return []string{"define"}
}

func (c *DefineCommand) Usages() []string {
	return []string{"%s <word>"}
}

func (c *DefineCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DefineCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *DefineCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())
	word := strings.Join(tokens[1:], " ")

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), word)

	snakeWord := strings.ReplaceAll(word, " ", "_")

	resp, err := http.Get(fmt.Sprintf(defineAPIURL, url.QueryEscape(snakeWord), c.cfg.MerriamWebster.DictionaryAPIKey))
	if err != nil {
		logger.Errorf(e, "error fetching definition: %v", err)
		c.Replyf(e, "Error fetching definition for %s.", style.Bold(word))
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf(e, "bad http response when fetching definition: %s", resp.Status)
		c.Replyf(e, "Error fetching definition for %s.", style.Bold(word))
		return
	}

	var response []defineResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf(e, "error reading response body: %v", err)
		c.Replyf(e, "Error reading definition for %s.", style.Bold(word))
		return
	}

	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&response); err != nil {
		logger.Errorf(e, "error decoding response: %v", err)

		var alternatives []string
		if err := json.NewDecoder(bytes.NewReader(b)).Decode(&alternatives); err != nil {
			logger.Errorf(e, "error decoding alternative response: %v", err)
			c.Replyf(e, "Error parsing definition for %s.", style.Bold(word))
			return
		}

		if len(alternatives) == 0 {
			logger.Warningf(e, "no definitions found for %s", word)
			c.Replyf(e, "No definitions found for %s.", style.Bold(word))
			return
		}

		resp2, err := http.Get(fmt.Sprintf(defineAPIURL, url.QueryEscape(alternatives[0]), c.cfg.MerriamWebster.DictionaryAPIKey))
		if err != nil {
			logger.Errorf(e, "error fetching definition: %v", err)
			c.Replyf(e, "Error finding a definition for %s.", style.Bold(word))
			return
		}

		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			logger.Errorf(e, "bad http response when fetching alternative definition: %s", resp2.Status)
			c.Replyf(e, "Error finding a definition for %s.", style.Bold(word))
			return
		}

		if err := json.NewDecoder(resp2.Body).Decode(&response); err != nil {
			logger.Errorf(e, "error decoding alternative response: %v", err)
			c.Replyf(e, "Error parsing definition for %s.", style.Bold(word))
			return
		}
	}

	if len(response) == 0 {
		logger.Warningf(e, "no definitions found for %s", word)
		c.Replyf(e, "No definitions found for %s.", style.Bold(word))
		return
	}

	definition := response[0]
	wordFound := strings.Split(definition.Meta.ID, ":")[0]

	if len(definition.Shortdef) == 0 {
		logger.Warningf(e, "no short definitions found for %s", word)
		c.Replyf(e, "No definitions found for %s.", style.Bold(word))
		return
	}

	message := fmt.Sprintf("%s", style.Bold(style.Underline(wordFound))) + ": "

	if len(definition.Type) > 0 {
		message += fmt.Sprintf("(%s) ", style.Italics(definition.Type))
	}

	ds := ""
	for _, def := range definition.Shortdef {
		if len(ds) > 0 {
			ds += "; "
		}
		if len(def) > 0 {
			ds += def
		}
	}

	message += ds

	if len(message) == 0 {
		logger.Warningf(e, "no definition text found for %s", word)
		c.Replyf(e, "No definitions found for %s.", style.Bold(word))
		return
	}

	url := fmt.Sprintf(definePublicURL, strings.ReplaceAll(wordFound, " ", "_"))

	if len(message) > extendedMaximumDescriptionLength {
		message = message[:extendedMaximumDescriptionLength] + "..."
	}

	c.SendMessages(e, e.ReplyTarget(), []string{message, url})
}

type defineResponse struct {
	Meta struct {
		ID string
	}
	Type     string `json:"fl"`
	Shortdef []string
}
