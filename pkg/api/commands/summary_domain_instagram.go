package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
)

const instagramEmbedDomain = "eeinstagram.com"
const discordBotUserAgent = "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)"

func (c *SummaryCommand) parseInstagram(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
	logger := log.Logger()

	embedURL := strings.Replace(url, "instagram.com", instagramEmbedDomain, 1)
	logger.Debugf(e, "instagram embed request for %s", embedURL)

	params := retriever.DefaultParams(embedURL)
	params.Impersonate = false
	params.Headers = map[string]string{
		"User-Agent": discordBotUserAgent,
	}

	doc, err := c.docRetriever.RetrieveDocument(e, params)
	if err != nil || doc == nil {
		return nil, nil, fmt.Errorf("unable to retrieve instagram embed for %s: %v", url, err)
	}

	titleAttr, _ := doc.Root.Find("meta[name='twitter:title']").First().Attr("content")
	author := strings.TrimSpace(titleAttr)

	descAttr, _ := doc.Root.Find("meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descAttr)

	if len(author) == 0 && len(description) == 0 {
		return nil, nil, fmt.Errorf("no instagram content found for %s", url)
	}

	if len(description) > extendedMaximumDescriptionLength {
		description = description[:extendedMaximumDescriptionLength] + "..."
	}

	var content string
	if len(description) > 0 && len(author) > 0 {
		content = fmt.Sprintf("%s • %s", style.Bold(description), author)
	} else if len(description) > 0 {
		content = style.Bold(description)
	} else {
		content = author
	}

	return createSummaryResult(content), nil, nil
}
