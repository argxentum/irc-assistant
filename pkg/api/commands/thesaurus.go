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

const ThesaurusCommandName = "thesaurus"

const thesaurusAPIURL = "https://dictionaryapi.com/api/v3/references/thesaurus/json/%s?key=%s"
const thesaurusPublicURL = "https://www.merriam-webster.com/thesaurus/%s"

type ThesaurusCommand struct {
	*commandStub
}

func NewThesaurusCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ThesaurusCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *ThesaurusCommand) Name() string {
	return ThesaurusCommandName
}

func (c *ThesaurusCommand) Description() string {
	return "Retrieves the synonyms and antonyms of a word."
}

func (c *ThesaurusCommand) Triggers() []string {
	return []string{"thesaurus"}
}

func (c *ThesaurusCommand) Usages() []string {
	return []string{"%s <word>"}
}

func (c *ThesaurusCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ThesaurusCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *ThesaurusCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())
	word := strings.Join(tokens[1:], " ")

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), word)

	snakeWord := strings.ReplaceAll(word, " ", "_")

	resp, err := http.Get(fmt.Sprintf(thesaurusAPIURL, url.QueryEscape(snakeWord), c.cfg.MerriamWebster.ThesaurusAPIKey))
	if err != nil {
		logger.Errorf(e, "error fetching thesaurus entry: %v", err)
		c.Replyf(e, "Error fetching thesaurus entry for %s.", style.Bold(word))
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf(e, "bad http response when fetching thesaurus entry: %s", resp.Status)
		c.Replyf(e, "Error fetching thesaurus entry for %s.", style.Bold(word))
		return
	}

	var response []thesaurusResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf(e, "error reading response body: %v", err)
		c.Replyf(e, "Error reading thesaurus entry for %s.", style.Bold(word))
		return
	}

	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&response); err != nil {
		logger.Errorf(e, "error decoding response: %v", err)

		var alternatives []string
		if err := json.NewDecoder(bytes.NewReader(b)).Decode(&alternatives); err != nil {
			logger.Errorf(e, "error decoding alternative response: %v", err)
			c.Replyf(e, "Error parsing thesaurus entry for %s.", style.Bold(word))
			return
		}

		if len(alternatives) == 0 {
			logger.Warningf(e, "no thesaurus entries found for %s", word)
			c.Replyf(e, "No thesaurus entries found for %s.", style.Bold(word))
			return
		}

		resp2, err := http.Get(fmt.Sprintf(thesaurusAPIURL, url.QueryEscape(alternatives[0]), c.cfg.MerriamWebster.ThesaurusAPIKey))
		if err != nil {
			logger.Errorf(e, "error fetching thesaurus entry: %v", err)
			c.Replyf(e, "Error finding a thesaurus entry for %s.", style.Bold(word))
			return
		}

		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			logger.Errorf(e, "bad http response when fetching alternative thesaurus entry: %s", resp2.Status)
			c.Replyf(e, "Error finding a thesaurus entry for %s.", style.Bold(word))
			return
		}

		if err := json.NewDecoder(resp2.Body).Decode(&response); err != nil {
			logger.Errorf(e, "error decoding alternative response: %v", err)
			c.Replyf(e, "Error parsing thesaurus entry for %s.", style.Bold(word))
			return
		}
	}

	if len(response) == 0 {
		logger.Warningf(e, "no thesaurus entry found for %s", word)
		c.Replyf(e, "No thesaurus entries found for %s.", style.Bold(word))
		return
	}

	entry := response[0]
	wordFound := strings.Split(entry.Meta.ID, ":")[0]

	if len(entry.Meta.Synonyms) == 0 && len(entry.Meta.Antonyms) == 0 {
		logger.Warningf(e, "no thesaurus synonyms or antonyms found for %s", word)
		c.Replyf(e, "No thesaurus entries found for %s.", style.Bold(word))
		return
	}

	message := fmt.Sprintf("%s", style.Bold(style.Underline(wordFound))) + ": "

	if len(entry.Type) > 0 {
		message += fmt.Sprintf("(%s) ", style.Italics(entry.Type))
	}

	s := ""
	if len(entry.Meta.Synonyms) > 0 {
		for _, syn := range entry.Meta.Synonyms[0] {
			if len(s) > 0 {
				s += ", "
			}
			if len(syn) > 0 {
				s += syn
			}
		}
	}

	a := ""
	if len(entry.Meta.Antonyms) > 0 {
		for _, ant := range entry.Meta.Antonyms[0] {
			if len(a) > 0 {
				a += ", "
			}
			if len(ant) > 0 {
				a += ant
			}
		}
	}

	if len(s) > 0 {
		message += fmt.Sprintf("%s: %s", style.Bold("synonyms"), s)
	}

	if len(a) > 0 {
		if len(s) > 0 {
			message += " | "
		}
		message += fmt.Sprintf("%s: %s", style.Bold("antonyms"), a)
	}

	if len(message) == 0 {
		logger.Warningf(e, "no thesaurus entry text found for %s", word)
		c.Replyf(e, "No thesaurus entries found for %s.", style.Bold(word))
		return
	}

	url := fmt.Sprintf(thesaurusPublicURL, strings.ReplaceAll(wordFound, " ", "_"))

	if len(message) > extendedMaximumDescriptionLength {
		message = message[:extendedMaximumDescriptionLength] + "..."
	}

	c.SendMessages(e, e.ReplyTarget(), []string{message, url})
}

type thesaurusResponse struct {
	Meta struct {
		ID       string
		Synonyms [][]string `json:"syns"`
		Antonyms [][]string `json:"ants"`
	}
	Type string `json:"fl"`
}
