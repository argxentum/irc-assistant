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
	url, translated := f.xTranslator.TranslateURL(tokens[0])
	if translated {
		f.handleX(e, url)
	} else {
		f.tryDirect(e, url)
	}
}

func (f *summaryFunction) tryDirect(e *core.Event, url string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
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

func (f *summaryFunction) tryBing(e *core.Event, url string) {
	c := colly.NewCollector()
	c.OnHTML("html", func(node *colly.HTMLElement) {
		title := strings.TrimSpace(node.DOM.Find("ol.b_results").Find("h2").Text())
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
	c := colly.NewCollector()
	c.OnHTML("html", func(node *colly.HTMLElement) {
		title := strings.TrimSpace(node.ChildText("span.title"))
		if len(title) > 0 {
			f.irc.SendMessage(e.ReplyTarget(), title)
			return
		}

		f.Reply(e, "Unable to provide a summary")
	})

	err := c.Visit(fmt.Sprintf("https://nug.zip/%s", url))
	if err != nil {
		f.Reply(e, "Unable to provide a summary")
	}
}

func (f *summaryFunction) handleX(e *core.Event, url string) {
	c := colly.NewCollector()

	c.OnError(func(r *colly.Response, err error) {
		if r.StatusCode == http.StatusFound {
			println("here")
		}
	})

	c.OnHTML("html", func(node *colly.HTMLElement) {
		h, _ := node.DOM.Html()
		println(h)

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

		f.Reply(e, "Unable to provide a summary")
	})

	err := c.Visit(url)
	if err != nil {
		f.Reply(e, "Unable to provide a summary")
	}
}
