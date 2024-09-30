package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strconv"
	"strings"
	"time"
)

const pollsFunctionName = "polls"

const pollsURL = "https://www.270towin.com/%d-presidential-election-polls/%s"

type pollsFunction struct {
	FunctionStub
	retriever retriever.DocumentRetriever
}

func NewPollsFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, pollsFunctionName)
	if err != nil {
		return nil, err
	}

	return &pollsFunction{
		FunctionStub: stub,
		retriever:    retriever.NewDocumentRetriever(),
	}, nil
}

func (f *pollsFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0)
}

func (f *pollsFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] polls", e.From, e.ReplyTarget())

	tokens := Tokens(e.Message())
	year := time.Now().Year()
	poll := "national"
	if len(tokens) > 1 {
		poll = strings.ToLower(tokens[1])
	}

	url := fmt.Sprintf(pollsURL, year, poll)
	doc, err := f.retriever.RetrieveDocumentSelection(e, retriever.DefaultParams(url), "html")
	if err != nil || doc == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse url (%s): %s", url, err)
		} else {
			logger.Warningf(e, "unable to parse url (%s)", url)
		}
		f.Replyf(e, "Unable to retrieve polling data")
		return
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())

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
		avg, err := strconv.ParseFloat(strings.Replace(a, "%", "", -1), 32)
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

	f.SendMessages(e, e.ReplyTarget(), []string{message, url})
}
