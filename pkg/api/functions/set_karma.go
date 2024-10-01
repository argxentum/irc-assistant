package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"regexp"
	"strings"
)

const setKarmaFunctionName = "setKarma"

type setKarmaFunction struct {
	FunctionStub
}

func NewSetKarmaFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, setKarmaFunctionName)
	if err != nil {
		return nil, err
	}

	return &setKarmaFunction{
		FunctionStub: stub,
	}, nil
}

func (f *setKarmaFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 0) && (strings.Contains(e.Message(), "++") || strings.Contains(e.Message(), "--"))
}

var karmaRegex = regexp.MustCompile(`(?i)(.*?)\s*(\+\+|--)(?:\s*,?\s+(.*))?`)

func (f *setKarmaFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	matches := karmaRegex.FindStringSubmatch(e.Message())
	if len(matches) < 3 {
		logger.Debugf(e, "invalid karma message: %s", e.Message())
		return
	}

	to := strings.TrimSpace(matches[1])
	if len(to) == 0 {
		logger.Debugf(e, "invalid karma target: %s", e.Message())
		return
	}

	if strings.ToLower(e.From) == strings.ToLower(to) {
		logger.Debugf(e, "cannot update own karma: %s", e.Message())
		f.Replyf(e, "You cannot update your own karma.")
		return
	}

	op := strings.TrimSpace(matches[2])
	if len(op) == 0 {
		logger.Debugf(e, "invalid karma operation: %s", e.Message())
		return
	}

	reason := ""
	if len(matches) > 3 {
		reason = strings.TrimSpace(matches[3])
	}

	log.Logger().Infof(e, "⚡ [%s/%s] setKarma %s %s %s", e.From, e.ReplyTarget(), to, op, reason)

	fs := firestore.Get()
	karma, err := fs.AddKarmaHistory(f.ctx, e.ReplyTarget(), e.From, to, op, reason)
	if err != nil {
		logger.Errorf(e, "error updating karma: %s", err)
		return
	}

	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s now has a karma score of %s.", style.Bold(to), style.Bold(fmt.Sprintf("%d", karma))))
}
