package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/rand/v2"
)

const karmaGetFunctionName = "karmaGet"

type karmaGetFunction struct {
	*functionStub
}

func NewKarmaGetFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &karmaGetFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
	}
}

func (f *karmaGetFunction) Name() string {
	return karmaGetFunctionName
}

func (f *karmaGetFunction) Description() string {
	return "Displays the given user's karma."
}

func (f *karmaGetFunction) Triggers() []string {
	return []string{"karma"}
}

func (f *karmaGetFunction) Usages() []string {
	return []string{"%s <user>"}
}

func (f *karmaGetFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *karmaGetFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 1)
}

func (f *karmaGetFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	nick := tokens[1]
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), nick)

	fs := firestore.Get()
	u, err := fs.User(f.ctx, e.ReplyTarget(), nick)
	if err != nil && status.Code(err) != codes.NotFound {
		log.Logger().Errorf(e, "error getting user, %s", err)
		f.Replyf(e, "unable to get karma for %s.", style.Bold(nick))
		return
	}

	if u == nil || status.Code(err) == codes.NotFound {
		log.Logger().Infof(e, "user not found")
		f.Replyf(e, "no karma found for %s.", style.Bold(nick))
		return
	}

	history, err := fs.KarmaHistory(f.ctx, e.ReplyTarget(), u.Nick)
	if err != nil {
		log.Logger().Errorf(e, "error getting karma history, %s", err)
		f.Replyf(e, "unable to get karma for %s.", style.Bold(nick))
		return
	}
	if len(history) == 0 {
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma of %s.", style.Bold(nick), style.Bold(fmt.Sprintf("%d", u.Karma))))
		return
	}

	h := history[rand.IntN(len(history))]

	action := "giving"
	if h.Op == firestore.OpSubtract {
		action = "taking away"
	}

	elapsedTime := elapse.TimeDescription(h.CreatedAt)

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
		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma of %s, %s %s %s karma %s.", style.Bold(nick), style.Bold(fmt.Sprintf("%d", u.Karma)), thanksTo, h.From, action, elapsedTime))
		return
	}

	f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s has a karma of %s, %s %s %s karma %s with the reason: %s", style.Bold(nick), style.Bold(fmt.Sprintf("%d", u.Karma)), thanksTo, h.From, action, elapsedTime, style.Bold(h.Reason)))
}
