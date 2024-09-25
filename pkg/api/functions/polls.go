package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"regexp"
	"strings"
)

const pollsFunctionName = "polls"
const home270toWinURL = "https://www.270towin.com"
const pollsDetailURL = "https://www.270towin.com/polls/php/get-polls-list-v2.php?election_year=%s&election_types[]=P&election_subtypes[]=GE&election_subtypes[]=GR&limit=40&base_location_url=/2024-presidential-election-polls/STATE"

type pollsFunction struct {
	FunctionStub
}

func NewPollsFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, pollsFunctionName)
	if err != nil {
		return nil, err
	}

	return &pollsFunction{
		FunctionStub: stub,
	}, nil
}

func (f *pollsFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

var detailFunctionRegexp = regexp.MustCompile(`get_polls\((\d+)`)

func (f *pollsFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] polls", e.From, e.ReplyTarget())

	home, err := getDocument(home270toWinURL, true)
	if err != nil || home == nil {
		logger.Warningf(e, "unable to retrieve polling data: %s", err)
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	pollsHomeUrl := ""
	home.Find("ul.navbar-nav").First().Find("a.dropdown-item").Each(func(i int, s *goquery.Selection) {
		if len(pollsHomeUrl) > 0 {
			return
		}
		if strings.TrimSpace(strings.ToLower(s.Text())) == "most recent general election polls" {
			pollsHomeUrl = s.AttrOr("href", "")
		}
	})

	if len(pollsHomeUrl) == 0 {
		logger.Warningf(e, "unable to retrieve polling data")
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	if !strings.HasPrefix(pollsHomeUrl, home270toWinURL) {
		pollsHomeUrl = home270toWinURL + pollsHomeUrl
	}

	pollsHome, err := getDocument(pollsHomeUrl, true)
	if err != nil || pollsHome == nil {
		logger.Warningf(e, "unable to parse pollsHomeUrl (%s): %s", pollsHomeUrl, err)
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	matcher := detailFunctionRegexp.FindStringSubmatch(pollsHome.Text())
	if len(matcher) != 2 {
		logger.Warningf(e, "unable to find get_polls(<year>) javascript call")
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}
	year := matcher[1]

	pollsDetail, err := getDocument(fmt.Sprintf(pollsDetailURL, year), true)
	if err != nil || pollsDetail == nil {
		logger.Warningf(e, "unable to parse pollsDetailURL (%s): %s", fmt.Sprintf(pollsDetailURL, year), err)
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	pollsURL := ""
	pollsDetail.Find("table#polls-list").First().Find("a").Each(func(i int, s *goquery.Selection) {
		if len(pollsURL) > 0 {
			return
		}
		if strings.TrimSpace(strings.ToLower(s.Text())) == "national" {
			pollsURL = s.AttrOr("href", "")
		}
	})

	if !strings.HasPrefix(pollsURL, home270toWinURL) {
		pollsURL = home270toWinURL + pollsURL
	}

	if len(pollsURL) == 0 {
		logger.Warningf(e, "unable to determine pollsURL")
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	doc, err := getDocument(pollsURL, true)
	if err != nil || doc == nil {
		logger.Warningf(e, "unable to parse pollsURL (%s): %s", pollsURL, err)
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	subtitle := strings.TrimSpace(doc.Find("h3").First().Text())

	candidates := make([]string, 0)
	doc.Find("table#polls").First().Find("th").Each(func(i int, s *goquery.Selection) {
		if _, ok := s.Attr("candidate_id"); ok {
			candidates = append(candidates, strings.TrimSpace(s.Text()))
		}
	})

	averages := make([]string, 0)
	average := doc.Find("table#polls").First().Find("td.poll_avg").First()
	for range candidates {
		average = average.Next()
		averages = append(averages, strings.TrimSpace(average.Text()))
	}

	if len(candidates) != len(averages) || len(candidates) == 0 {
		logger.Warningf(e, "unable to parse polling data, candidates: [%s], averages: [%s]", strings.Join(candidates, ", "), strings.Join(averages, ", "))
		f.Replyf(e, "Unable to parse polling data")
		return
	}

	t := createDefaultTable()
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},
		{Number: 2, Align: text.AlignRight},
	})

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s: %s", style.Bold(title), subtitle))
	for i, c := range candidates {
		t.AppendRow([]any{style.Bold(c), averages[i]})
	}
	messages = append(messages, strings.Split(t.Render(), "\n")...)
	messages = append(messages, pollsURL)

	f.SendMessages(e, e.ReplyTarget(), messages)
}
