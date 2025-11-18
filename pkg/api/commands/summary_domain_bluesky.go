package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var blueskyAuthorRegex = regexp.MustCompile(`^(?:(.*?)\s*\(@(.*?)\)|@(.*?))$`)

func (c *SummaryCommand) parseBlueSky(e *irc.Event, url string) (*summary, *models.Source, error) {
	params := retriever.DefaultParams(url)
	params.Impersonate = false

	doc, err := c.docRetriever.RetrieveDocument(e, params)
	if err != nil || doc == nil {
		if err != nil {
			if errors.Is(err, retriever.DisallowedContentTypeError) {
				return nil, nil, fmt.Errorf("disallowed content type for %s", url)
			}
			return nil, nil, fmt.Errorf("unable to retrieve %s: %s", url, err)
		} else {
			return nil, nil, fmt.Errorf("unable to retrieve %s", url)
		}
	}

	atNameAttr, _ := doc.Root.Find("meta[name='article:published_time']").First().Attr("content")
	atPropAttr, _ := doc.Root.Find("meta[property='article:published_time']").First().Attr("content")
	titleAttr, _ := doc.Root.Find("meta[property='og:title']").First().Attr("content")
	title := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Root.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if len(title) == 0 {
		title = strings.TrimSpace(doc.Root.Find("title").First().Text())
	}

	if c.isRejectedTitle(title) {
		return nil, nil, fmt.Errorf("rejected bluesky title: %s", title)
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, nil, fmt.Errorf("title and description too short, title: %s, description: %s", title, description)
	}

	at := ""
	if len(atNameAttr) > 0 {
		if t, err := time.Parse(time.RFC3339, atNameAttr); err == nil {
			at = elapse.PastTimeDescription(t)
		}
	}
	if at == "" && len(atPropAttr) > 0 {
		if t, err := time.Parse(time.RFC3339, atPropAttr); err == nil {
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

	var source *models.Source
	m := blueskyAuthorRegex.FindStringSubmatch(title)
	if len(m) > 3 {
		author := m[1]
		authorHandlePrimary := m[2]
		authorHandleSecondary := m[3]
		authorHandle := authorHandlePrimary
		if len(authorHandle) == 0 {
			authorHandle = authorHandleSecondary
		}

		authorSource, err := repository.FindSource(author)
		if err != nil {
			log.Logger().Errorf(nil, "error finding author source, %s", err)
		}

		authorHandleSource, err := repository.FindSource(authorHandle)
		if err != nil {
			log.Logger().Errorf(nil, "error finding author handle source, %s", err)
		}

		if authorHandleSource != nil {
			source = authorHandleSource
		} else if authorSource != nil {
			source = authorSource
		}
	}

	return createSummary(messages...), source, nil
}
