package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strconv"
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

	home, err := f.getDocument(e, home270toWinURL, true)
	if err != nil || home == nil {
		if err != nil {
			logger.Warningf(e, "unable to retrieve polling data: %s", err)
		} else {
			logger.Warningf(e, "unable to retrieve polling data")
		}
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

	pollsHome, err := f.getDocument(e, pollsHomeUrl, true)
	if err != nil || pollsHome == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse pollsHomeUrl (%s): %s", pollsHomeUrl, err)
		} else {
			logger.Warningf(e, "unable to parse pollsHomeUrl (%s)", pollsHomeUrl)
		}
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

	pollsDetail, err := f.getDocument(e, fmt.Sprintf(pollsDetailURL, year), true)
	if err != nil || pollsDetail == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse pollsDetailURL (%s): %s", fmt.Sprintf(pollsDetailURL, year), err)
		} else {
			logger.Warningf(e, "unable to parse pollsDetailURL (%s)", fmt.Sprintf(pollsDetailURL, year))
		}
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

	doc, err := f.getDocument(e, pollsURL, true)
	if err != nil || doc == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse pollsURL (%s): %s", pollsURL, err)
		} else {
			logger.Warningf(e, "unable to parse pollsURL (%s)", pollsURL)
		}
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	title := strings.TrimSpace(doc.Find("h3").First().Text())

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

	winningAvg := 0.0
	winningCandidateIndex := 0
	for i, a := range averages {
		avg, err := strconv.ParseFloat(a, 32)
		if err != nil {
			logger.Warningf(e, "unable to parse polling average for %s: %s", candidates[i], a)
			continue
		}
		if avg > winningAvg {
			winningAvg = avg
			winningCandidateIndex = i
		}
	}

	message := ""
	for i, c := range candidates {
		if len(message) > 0 {
			message += ", "
		}
		if i == winningCandidateIndex {
			message += style.ColorForeground(fmt.Sprintf("%s: %s", style.Underline(c), averages[i]), style.ColorGreen)
		} else {
			message += fmt.Sprintf("%s: %s", style.Underline(c), averages[i])
		}
	}
	message = fmt.Sprintf("%s – %s", style.Bold(title), message)

	f.SendMessages(e, e.ReplyTarget(), []string{message, pollsURL})
}
