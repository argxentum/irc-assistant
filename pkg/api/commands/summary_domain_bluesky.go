package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var blueskyAuthorRegex = regexp.MustCompile(`^(.*?)\s*\(@(.*?)\)$`)

func (c *SummaryCommand) parseBlueSky(e *irc.Event, url string) (*summary, error) {
	params := retriever.DefaultParams(url)
	params.Impersonate = false

	doc, err := c.docRetriever.RetrieveDocument(e, params, retriever.DefaultTimeout)
	if err != nil || doc == nil {
		if err != nil {
			if errors.Is(err, retriever.DisallowedContentTypeError) {
				return nil, fmt.Errorf("disallowed content type for %s", url)
			}
			return nil, fmt.Errorf("unable to retrieve %s: %s", url, err)
		} else {
			return nil, fmt.Errorf("unable to retrieve %s", url)
		}
	}

	atAttr, _ := doc.Find("meta[name='article:published_time']").First().Attr("content")
	titleAttr, _ := doc.Find("meta[property='og:title']").First().Attr("content")
	title := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if len(title) == 0 {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}

	if c.isRejectedTitle(title) {
		return nil, fmt.Errorf("rejected title: %s", title)
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, fmt.Errorf("title and description too short, title: %s, description: %s", title, description)
	}

	at := ""
	if len(atAttr) > 0 {
		if t, err := time.Parse(time.RFC3339, atAttr); err == nil {
			at = elapse.PastTimeDescription(t)
		}
	}

	content := ""
	if len(description) > 0 {
		content = fmt.Sprintf("%s • %s", style.Bold(description), title)
	} else {
		content = title
	}

	if len(at) > 0 {
		content = fmt.Sprintf("%s • %s", content, at)
	}

	messages := make([]string, 0)
	messages = append(messages, content)

	m := blueskyAuthorRegex.FindStringSubmatch(title)
	if len(m) > 2 {
		author := m[1]
		authorHandle := m[2]

		authorSource, err := repository.FindSource(author)
		if err != nil {
			log.Logger().Errorf(nil, "error finding author source, %s", err)
		}

		authorHandleSource, err := repository.FindSource(authorHandle)
		if err != nil {
			log.Logger().Errorf(nil, "error finding author handle source, %s", err)
		}

		if authorHandleSource != nil {
			messages = append(messages, repository.SourceShortDescription(authorHandleSource))
		} else if authorSource != nil {
			messages = append(messages, repository.SourceShortDescription(authorSource))
		}
	}

	return createSummary(messages...), nil
}
