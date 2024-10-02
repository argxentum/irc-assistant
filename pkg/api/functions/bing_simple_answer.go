package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"net/url"
	"strings"
)

type bingSimpleAnswerFunction struct {
	FunctionStub
	subject   string
	query     string
	reply     string
	footnote  string
	minTokens int
	retriever retriever.DocumentRetriever
}

func NewBingSimpleAnswerFunction(subject, query, reply, footnote string, minTokens int, ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, fmt.Sprintf("bing/simple/%s", subject))
	if err != nil {
		return nil, err
	}

	return &bingSimpleAnswerFunction{
		FunctionStub: stub,
		subject:      subject,
		query:        query,
		reply:        reply,
		footnote:     footnote,
		minTokens:    minTokens,
		retriever:    retriever.NewDocumentRetriever(),
	}, nil
}

func (f *bingSimpleAnswerFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, f.minTokens)
}

func (f *bingSimpleAnswerFunction) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ [%s/%s] bing/simple/%s", e.From, e.ReplyTarget(), f.subject)

	tokens := Tokens(e.Message())
	input := ""
	if len(tokens) > 0 {
		input = strings.Join(tokens[1:], " ")
	}
	query := url.QueryEscape(f.query)
	if len(input) > 0 {
		query = url.QueryEscape(fmt.Sprintf(f.query, input))
	}

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(fmt.Sprintf(bingSearchURL, query)), 3000)
	if err != nil {
		logger.Warningf(e, "error fetching bing search results for %s: %s", input, err)
		f.Replyf(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	node := doc.Find("ol#b_results li.b_ans").First()
	label := node.Find("div.b_focusLabel").First().Text()
	answer1 := node.Find("div.b_focusTextLarge").First().Text()
	answer2 := node.Find("div.b_focusTextMedium").First().Text()
	secondary1 := node.Find("div.b_secondaryFocus").First().Text()
	secondary2 := node.Find("li.b_secondaryFocus").First().Text()

	label = strings.TrimSpace(label)
	answer := strings.TrimSpace(coalesce(answer1, answer2))
	secondary := strings.TrimSpace(coalesce(secondary1, secondary2))

	if len(label) == 0 || len(answer) == 0 || len(secondary) == 0 {
		logger.Warningf(e, "error parsing bing search results for %s", input)
		f.Replyf(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf(f.reply, label, style.Bold(answer), secondary))

	if len(f.footnote) > 0 {
		f.SendMessage(e, e.ReplyTarget(), f.footnote)
	}
}
