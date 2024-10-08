package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

func (f *summaryFunction) parseMastodon(_ *irc.Event, doc *goquery.Document) (*summary, error) {
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
