package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

const pollsFunctionName = "polls"
const home270toWinURL = "https://www.270towin.com"
const pollsDetailURL = "https://www.270towin.com/polls/php/get-polls-list-v2.php?election_year=%s&election_types[]=P&election_subtypes[]=GE&election_subtypes[]=GR&limit=40&base_location_url=/2024-presidential-election-polls/STATE"

type pollsFunction struct {
	FunctionStub
}

func NewPollsFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, pollsFunctionName)
	if err != nil {
		return nil, err
	}

	return &pollsFunction{
		FunctionStub: stub,
	}, nil
}

func (f *pollsFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, 0)
}

var detailFunctionRegexp = regexp.MustCompile(`get_polls\((\d+)`)

func (f *pollsFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ polls\n")

	home, err := getDocument(home270toWinURL, true)
	if err != nil || home == nil {
		f.Reply(e, "Unable to retrieve polling data")
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
		f.Reply(e, "Unable to retrieve polling data")
		return
	}

	if !strings.HasPrefix(pollsHomeUrl, home270toWinURL) {
		pollsHomeUrl = home270toWinURL + pollsHomeUrl
	}

	pollsHome, err := getDocument(pollsHomeUrl, true)
	if err != nil || pollsHome == nil {
		f.Reply(e, "Unable to retrieve polling data")
		return
	}

	matcher := detailFunctionRegexp.FindStringSubmatch(pollsHome.Text())
	if len(matcher) != 2 {
		f.Reply(e, "Unable to retrieve polling data")
		return
	}
	year := matcher[1]

	pollsDetail, err := getDocument(fmt.Sprintf(pollsDetailURL, year), true)
	if err != nil || pollsDetail == nil {
		f.Reply(e, "Unable to retrieve polling data")
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
		f.Reply(e, "Unable to retrieve polling data")
		return
	}

	doc, err := getDocument(pollsURL, true)
	if err != nil || doc == nil {
		f.Reply(e, "Unable to retrieve polling data")
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
		f.Reply(e, "Unable to parse polling data")
		return
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s: %s", title, text.Bold(subtitle)))
	for i, c := range candidates {
		messages = append(messages, fmt.Sprintf("%s: %s", text.Bold(text.Underline(c)), text.Bold(averages[i])))
	}
	messages = append(messages, pollsURL)

	f.irc.SendMessages(e.ReplyTarget(), messages)
}
