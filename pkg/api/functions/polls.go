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
	*functionStub
	retriever retriever.DocumentRetriever
}

func NewPollsFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &pollsFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
		retriever:    retriever.NewDocumentRetriever(retriever.NewBodyRetriever()),
	}
}

func (f *pollsFunction) Name() string {
	return pollsFunctionName
}

func (f *pollsFunction) Description() string {
	return "Displays the latest polling data from 270toWin."
}

func (f *pollsFunction) Triggers() []string {
	return []string{"polls"}
}

func (f *pollsFunction) Usages() []string {
	return []string{"%s", "%s <poll>"}
}

func (f *pollsFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *pollsFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *pollsFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s]", f.Name(), e.From, e.ReplyTarget())

	tokens := Tokens(e.Message())
	year := time.Now().Year()
	poll := "national"
	pollInput := poll
	if len(tokens) > 1 {
		pollInput = strings.Join(tokens[1:], " ")
		poll = strings.ToLower(strings.Join(tokens[1:], "-"))
	}

	url := fmt.Sprintf(pollsURL, year, poll)
	doc, err := f.retriever.RetrieveDocumentSelection(e, retriever.DefaultParams(url), "html")
	if err != nil || doc == nil {
		if err != nil {
			logger.Warningf(e, "unable to parse url (%s): %s", url, err)
		} else {
			logger.Warningf(e, "unable to parse url (%s)", url)
		}
		f.Replyf(e, "Unable to retrieve %s polling data", style.Bold(pollInput))
		return
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())

	candidates := make([]string, 0)
	doc.Find("table#polls").First().Find("th").Each(func(i int, s *goquery.Selection) {
		if _, ok := s.Attr("candidate_id"); ok {
			candidates = append(candidates, strings.TrimSpace(s.Text()))
		}
	})

	polls := doc.Find("table#polls").First()

	averages := make([]string, 0)
	average := polls.Find("td.poll_avg").First()
	for range candidates {
		average = average.Next()
		value := strings.TrimSpace(average.Text())
		if len(value) > 0 {
			averages = append(averages, value)
		}
	}

	if len(averages) == 0 {
		average = polls.Find("td.poll_data").First()
		for range candidates {
			averages = append(averages, strings.TrimSpace(average.Text()))
			average = average.Next()
		}
	}

	if len(candidates) != len(averages) || len(candidates) == 0 {
		logger.Warningf(e, "unable to parse polling data, candidates: [%s], averages: [%s]", strings.Join(candidates, ", "), strings.Join(averages, ", "))
		f.Replyf(e, "Unable to parse %s polling data", style.Bold(pollInput))
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
