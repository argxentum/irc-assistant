package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"github.com/gocolly/colly/v2"
	"net/http"
	"strings"
)

const summaryFunctionName = "summary"

var allowedContentTypePrefixes = []string{
	"text/html",
	"text/plain",
	"text/xml",
	"application/xml",
	"application/xhtml",
	"application/rss",
	"application/atom",
	"application/rdf",
	"application/json",
	"application/ld+json",
	"application/vnd.api",
	"application/hal+json",
	"application/vnd.collection",
}

type summaryFunction struct {
	Stub
	xTranslator XTranslator
}

func NewSummaryFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, summaryFunctionName)
	if err != nil {
		return nil, err
	}

	return &summaryFunction{
		Stub:        stub,
		xTranslator: NewXTranslator(),
	}, nil
}

func (f *summaryFunction) MayExecute(e *core.Event) bool {
	if !f.isValid(e, 0) {
		return false
	}

	tokens := Tokens(e.Message())
	return strings.HasPrefix(tokens[0], "https://") || strings.HasPrefix(tokens[0], "http://")
}

func (f *summaryFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ summary\n")
	tokens := Tokens(e.Message())
	url, isX := f.xTranslator.TranslateURL(tokens[0])
	if isX {
		f.handleX(e, url)
	} else {
		f.tryDirect(e, url)
	}
}

func (f *summaryFunction) tryDirect(e *core.Event, url string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		for k, v := range summaryHeaders {
			r.Headers.Set(k, v)
		}
	})

	c.OnHTML("html", func(node *colly.HTMLElement) {
		contentType := node.Response.Headers.Get("Content-Type")
		if !isContentTypeAllowed(contentType) {
			fmt.Printf("⚠️ ignoring invalid content (%s) type for %s\n", contentType, url)
			return
		}

		title := strings.TrimSpace(node.ChildAttr("meta[property='og:title']", "content"))
		description := strings.TrimSpace(node.ChildAttr("meta[property='og:description']", "content"))
		if len(title) > 0 && len(description) > 0 {
			if strings.Contains(description, title) || strings.Contains(title, description) {
				if len(description) > len(title) {
					f.irc.SendMessage(e.ReplyTarget(), description)
					return
				} else {
					f.irc.SendMessage(e.ReplyTarget(), title)
					return
				}
			}
			f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s", title, description))
			return
		}
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}
		if len(description) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), description)
			return
		}

		title = strings.TrimSpace(node.DOM.Find("h1").Text())
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}

		title = strings.TrimSpace(node.DOM.Find("title").Text())
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}

		f.tryBing(e, url)
	})

	err := c.Visit(url)
	if err != nil {
		f.tryBing(e, url)
	}
}
func isContentTypeAllowed(contentType string) bool {
	for _, p := range allowedContentTypePrefixes {
		if strings.HasPrefix(contentType, p) {
			return true
		}
	}
	return false
}

func (f *summaryFunction) tryBing(e *core.Event, url string) {
	fmt.Printf("ℹ trying bing for %s\n", url)

	c := colly.NewCollector()
	c.OnHTML("html", func(node *colly.HTMLElement) {
		title := strings.TrimSpace(node.DOM.Find("ol#b_results").Find("h2").Text())
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}

		f.tryNuggetize(e, url)
	})

	err := c.Visit(fmt.Sprintf("https://www.bing.com/search?q=%s", url))
	if err != nil {
		f.tryNuggetize(e, url)
	}
}

func (f *summaryFunction) tryNuggetize(e *core.Event, url string) {
	fmt.Printf("ℹ trying nuggetize for %s\n", url)

	c := colly.NewCollector()
	c.OnHTML("html", func(node *colly.HTMLElement) {
		title := strings.TrimSpace(node.ChildText("span.title"))
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}

		fmt.Printf("⚠️ unable to summarize %s\n", url)
	})

	err := c.Visit(fmt.Sprintf("https://nug.zip/%s", url))
	if err != nil {
		fmt.Printf("⚠️ summarization failed, error retrieving %s\n", url)
	}
}

func (f *summaryFunction) handleX(e *core.Event, url string) {
	fmt.Printf("ℹ handling x for %s\n", url)

	c := colly.NewCollector()
	c.OnError(func(r *colly.Response, err error) {
		if r.StatusCode == http.StatusFound {
			println("here")
		}
	})

	c.OnHTML("html", func(node *colly.HTMLElement) {
		title := strings.TrimSpace(node.ChildAttr("meta[property='og:title']", "content"))
		description := strings.TrimSpace(node.ChildAttr("meta[property='og:description']", "content"))
		if len(title) > 0 && len(description) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf("%s: %s", title, description))
			return
		}
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}
		if len(description) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), description)
			return
		}

		fmt.Printf("⚠️ unable to summarize %s\n", url)
	})

	err := c.Visit(url)
	if err != nil {
		fmt.Printf("⚠️ summarization failed, error retrieving %s\n", url)
	}
}
