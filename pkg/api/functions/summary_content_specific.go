package functions

import (
	"assistant/pkg/api/irc"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

var csf map[string]func(e *irc.Event, doc *goquery.Document) (*summary, error)

func (f *summaryFunction) contentSpecificSummarization() map[string]func(e *irc.Event, doc *goquery.Document) (*summary, error) {
	if csf == nil {
		csf = map[string]func(e *irc.Event, doc *goquery.Document) (*summary, error){
			"https://joinmastodon.org/apps": f.parseMastodon,
		}
	}

	return csf
}

func (f *summaryFunction) contentSpecificSummarizer(doc *goquery.Document) func(e *irc.Event, doc *goquery.Document) (*summary, error) {
	if doc == nil {
		return nil
	}

	for content, fn := range f.contentSpecificSummarization() {
		if strings.Contains(doc.Text(), content) {
			return fn
		}
	}

	return nil
}
