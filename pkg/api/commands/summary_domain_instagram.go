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
	logger.Debugf(e, "Instagram summary path: trying oEmbed for %s", url)
	if result, metadata, err := c.oEmbedSummary(e, instagramOEmbedURL, url); err == nil && result != nil {
		logger.Debugf(e, "Instagram summary path: oEmbed succeeded for %s", url)
		source, sourceErr := findOEmbedSource(e, metadata)
		if sourceErr != nil {
			logger.Errorf(e, "Instagram oEmbed source check failed for %s: %v", url, sourceErr)
		}
		return result, source, nil
	} else if err != nil {
		logger.Debugf(e, "Instagram summary path: oEmbed failed for %s: %s; falling back to %s", url, err, instagramEmbedDomain)
	} else {
		logger.Debugf(e, "Instagram summary path: oEmbed returned no usable summary for %s; falling back to %s", url, instagramEmbedDomain)
	}

	logger.Debugf(e, "Instagram summary path: trying %s metadata for %s", instagramEmbedDomain, url)
	return c.parseInstagramEmbed(e, url)
}

func (c *SummaryCommand) parseInstagramEmbed(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
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

	logger.Debugf(e, "Instagram summary path: %s metadata succeeded for %s", instagramEmbedDomain, url)
	return createSummaryResult(content), nil, nil
}
