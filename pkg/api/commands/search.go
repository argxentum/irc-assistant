package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const SearchCommandName = "search"

const bingSearchURL = "https://www.bing.com/search?q=%s"
const duckDuckGoSearchURL = "https://html.duckduckgo.com/html?q=%s"
const startPageSearchURL = "https://www.startpage.com/sp/search?q=%s"

const duckDuckGoSearchResultURLPattern = `//duckduckgo.com/l/\?uddg=(.*?)&`

type SearchCommand struct {
	*commandStub
	retriever retriever.DocumentRetriever
}

func NewSearchCommand(ctx context.Context, cfg *config.Config, irc irc.IRC) Command {
	return &SearchCommand{
		commandStub: defaultCommandStub(ctx, cfg, irc),
		retriever:   retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (c *SearchCommand) Name() string {
	return SearchCommandName
}

func (c *SearchCommand) Description() string {
	return "Searches the web for the given query."
}

func (c *SearchCommand) Triggers() []string {
	return []string{"search"}
}

func (c *SearchCommand) Usages() []string {
	return []string{"%s <query>"}
}

func (c *SearchCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *SearchCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

var httpRegex = regexp.MustCompile(`^https?://`)

func (c *SearchCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	input := strings.Join(tokens[1:], " ")

	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), input)

	for _, search := range c.searchChain() {
		s, err := search(e, input)
		if err != nil {
			continue
		}

		if s != nil {
			last := s.messages[len(s.messages)-1]
			if httpRegex.MatchString(last) {
				source, err := repository.FindSource(last)
				if err != nil {
					logger.Errorf(nil, "error finding source, %s", err)
				}

				if source != nil {
					s.messages = append(s.messages, repository.ShortSourceSummary(source))
				}
			}

			c.SendMessages(e, e.ReplyTarget(), s.messages)

			return
		}
	}
}

var ssf []func(e *irc.Event, input string) (*summary, error)

func (c *SearchCommand) searchChain() []func(e *irc.Event, input string) (*summary, error) {
	if ssf == nil {
		ssf = []func(e *irc.Event, input string) (*summary, error){
			c.searchBing,
			c.searchDuckDuckGo,
			c.searchStartPage,
		}
	}

	return ssf
}

func (c *SearchCommand) searchBing(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()
	logger.Debugf(e, "searching bing for %s", input)
	query := url.QueryEscape(input)

	doc, err := c.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve bing search results for %s", input)
		return nil, fmt.Errorf("bing search results doc nil")
	}

	cTitle := strings.TrimSpace(doc.Root.Find("ol#b_results li.b_algo h2").First().Text())
	link := strings.TrimSpace(doc.Root.Find("ol#b_results li.b_algo h2 a").First().AttrOr("href", ""))
	site := strings.TrimSpace(doc.Root.Find("ol#b_results li.b_algo div.tptt").First().Text())

	if len(cTitle) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty bing search results for %s, title: %s, link: %s", input, cTitle, link)
		return nil, errors.New("empty bing search results data")
	}

	title := ""

	if len(cTitle) > 0 && len(site) > 0 {
		if text.MostlyContains(cTitle, site, 0.9) {
			if len(cTitle) > len(site) {
				title = style.Bold(cTitle)
			} else {
				title = style.Bold(site)
			}
		} else {
			title = fmt.Sprintf("%s • %s", style.Bold(cTitle), site)
		}
	} else if len(site) > 0 {
		title = style.Bold(site)
	} else if len(cTitle) > 0 {
		title = style.Bold(cTitle)
	} else {
		return nil, summaryTooShortError
	}

	if len(link) == 0 {
		return nil, summaryTooShortError
	}

	return createSearchResultSummary(e, title, link), nil
}

var searchResultURLRegex = regexp.MustCompile(duckDuckGoSearchResultURLPattern)

func (c *SearchCommand) searchDuckDuckGo(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()

	logger.Infof(e, "searching duckduckgo for %s", input)
	query := url.QueryEscape(input)

	doc, err := c.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(duckDuckGoSearchURL, query)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve duckduckgo search results for %s", input)
		return nil, fmt.Errorf("duckduckgo search results doc nil")
	}

	title := strings.TrimSpace(doc.Root.Find("div.result__body h2.result__title").First().Text())
	linkRaw := strings.TrimSpace(doc.Root.Find("div.result__body h2.result__title a.result__a").First().AttrOr("href", ""))

	match := searchResultURLRegex.FindStringSubmatch(linkRaw)
	if len(match) < 2 {
		logger.Debugf(e, "unable to parse duckduckgo search result link for %s", input)
		return nil, summaryTooShortError
	}

	link, err := url.QueryUnescape(match[1])
	if err != nil {
		logger.Debugf(e, "unable to unescape duckduckgo search result link for %s: %s", input, err)
		return nil, err
	}

	if len(title) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty duckduckgo search results for %s", input)
		return nil, summaryTooShortError
	}

	return createSearchResultSummary(e, style.Bold(title), link), nil
}

func (c *SearchCommand) searchStartPage(e *irc.Event, input string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "searching startpage for %s", input)
	query := url.QueryEscape(input)

	doc, err := c.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(startPageSearchURL, query)))
	if err != nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s: %s", input, err)
		return nil, err
	}

	if doc == nil {
		logger.Debugf(e, "unable to retrieve startpage search results for %s", input)
		return nil, fmt.Errorf("startpage search results doc nil")
	}

	link := doc.Root.Find("section#main a.result-link").First().AttrOr("href", "")
	title := strings.TrimSpace(doc.Root.Find("section#main h2").First().Text())

	if len(title) == 0 || len(link) == 0 {
		logger.Debugf(e, "empty startpage search results for %s", input)
		return nil, summaryTooShortError
	}

	return createSearchResultSummary(e, style.Bold(title), link), nil
}

func createSearchResultSummary(e *irc.Event, title, url string) *summary {
	s := createSummary()

	if sc := registry.Command(SummaryCommandName); sc != nil {
		dsc := sc.(*SummaryCommand)
		if ds, err := dsc.domainSummary(e, url); ds != nil && err == nil {
			s.addMessages(ds.messages...)
		}
	}

	if len(s.messages) == 0 {
		s.addMessage(title)
	}

	s.addMessage(url)

	return s
}
