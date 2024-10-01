package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapsed"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"math/rand/v2"
)

const getKarmaFunctionName = "getKarma"

type getKarmaFunction struct {
	FunctionStub
}

func NewGetKarmaFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, getKarmaFunctionName)
	if err != nil {
		return nil, err
	}

	return &getKarmaFunction{
		FunctionStub: stub,
	}, nil
}

func (f *getKarmaFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 1)
}

func (f *getKarmaFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	user := tokens[1]
	log.Logger().Infof(e, "⚡ [%s/%s] getKarma %s", e.From, e.ReplyTarget(), user)

	fs := firestore.Get()
	u, err := fs.User(f.ctx, e.ReplyTarget(), user)
	if err != nil {
		log.Logger().Errorf(e, "error getting user, %s", err)
		f.Replyf(e, "unable to get karma for %s.", style.Bold(user))
		return
	}

	if u == nil {
		log.Logger().Infof(e, "user not found")
		f.Replyf(e, "no karma found for %s.", style.Bold(user))
		return
	}

	history, err := fs.KarmaHistory(f.ctx, e.ReplyTarget(), u.ID)
	if err != nil {
		log.Logger().Errorf(e, "error getting karma history, %s", err)
		f.Replyf(e, "unable to get karma for %s.", style.Bold(user))
		return
	}
	if len(history) == 0 {
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma score of %s.", style.Bold(user), style.Bold(fmt.Sprintf("%d", u.Karma))))
		return
	}

	h := history[rand.IntN(len(history))]

	action := "gave"
	if h.Op == firestore.OpDecrement {
		action = "took away"
	}

	elapsedTime := elapsed.ElapsedTimeDescription(h.CreatedAt)

	thanksToPhrases := []string{
		"in small part thanks to",
		"in part due to",
		"partially due to",
		"partially because of",
		"partly because of",
		"partly due to",
		"partially thanks to",
		"in part because of",
		"part of which is due to",
	}
	thanksTo := thanksToPhrases[rand.IntN(len(thanksToPhrases))]

	if len(h.Reason) == 0 {
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma score of %s, %s %s who %s karma %s ago.", style.Bold(user), style.Bold(fmt.Sprintf("%d", u.Karma)), thanksTo, h.From, action, elapsedTime))
		return
	}

	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma score of %s, %s %s who %s karma %s with the reason: %s", style.Bold(user), style.Bold(fmt.Sprintf("%d", u.Karma)), thanksTo, h.From, action, elapsedTime, style.Bold(h.Reason)))
}
