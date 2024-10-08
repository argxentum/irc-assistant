package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"errors"
	"fmt"
	"strings"
)

func (f *summaryFunction) parseBlueSky(e *irc.Event, url string) (*summary, error) {
	params := retriever.DefaultParams(url)
	params.Impersonate = false

	doc, err := f.retriever.RetrieveDocument(e, params, retriever.DefaultTimeout)
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

	titleAttr, _ := doc.Find("meta[property='og:title']").First().Attr("content")
	title := strings.TrimSpace(titleAttr)
	descriptionAttr, _ := doc.Find("html meta[property='og:description']").First().Attr("content")
	description := strings.TrimSpace(descriptionAttr)

	if len(description) > maximumDescriptionLength {
		description = description[:maximumDescriptionLength] + "..."
	}

	if len(title) == 0 {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}

	if isRejectedTitle(title) {
		return nil, fmt.Errorf("rejected title: %s", title)
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, fmt.Errorf("title and description too short, title: %s, description: %s", title, description)
	}

	return &summary{text: fmt.Sprintf("%s • %s", style.Bold(description), title)}, nil
}
