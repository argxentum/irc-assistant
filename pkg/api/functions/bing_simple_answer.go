package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"assistant/pkg/api/text"
	"fmt"
	"net/url"
	"strings"
)

const bingSearchUrl = "https://www.bing.com/search?q=%s"

type bingSimpleAnswerFunction struct {
	FunctionStub
	subject   string
	query     string
	reply     string
	footnote  string
	minTokens int
}

// when is the next election day

func NewBingSimpleAnswerFunction(subject, query, reply, footnote string, minTokens int, ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
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
	}, nil
}

func (f *bingSimpleAnswerFunction) MayExecute(e *core.Event) bool {
	return f.isValid(e, f.minTokens)
}

func (f *bingSimpleAnswerFunction) Execute(e *core.Event) {
	fmt.Printf("⚡ bing/simple/%s\n", f.subject)
	tokens := Tokens(e.Message())
	input := ""
	if len(tokens) > 0 {
		input = strings.Join(tokens[1:], " ")
	}
	query := url.QueryEscape(f.query)
	if len(input) > 0 {
		query = url.QueryEscape(fmt.Sprintf(f.query, input))
	}

	doc, err := getDocument(fmt.Sprintf(bingSearchUrl, query), true)
	if err != nil {
		f.Reply(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	node := doc.Find("ol#b_results li.b_ans").First()
	label := node.Find("div.b_focusLabel").First().Text()
	answer1 := node.Find("div.b_focusTextLarge").First().Text()
	answer2 := node.Find("div.b_focusTextMedium").First().Text()
	secondary1 := node.Find("div.b_secondaryFocus").First().Text()
	secondary2 := node.Find("li.b_secondaryFocus").First().Text()

	answer := coalesce(answer1, answer2)
	secondary := coalesce(secondary1, secondary2)

	if len(label) == 0 || len(answer) == 0 || len(secondary) == 0 {
		f.Reply(e, "Sorry, something went wrong and I couldn't find an answer.")
		return
	}

	f.irc.SendMessage(e.ReplyTarget(), fmt.Sprintf(f.reply, label, text.Bold(answer), secondary))

	if len(f.footnote) > 0 {
		f.irc.SendMessage(e.ReplyTarget(), f.footnote)
	}
}
