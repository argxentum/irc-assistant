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

const karmaSetFunctionName = "karmaSet"
const maxKarmaReasonLength = 128

type karmaSetFunction struct {
	*functionStub
}

func NewKarmaSetFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &karmaSetFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
	}
}

func (f *karmaSetFunction) Name() string {
	return karmaSetFunctionName
}

func (f *karmaSetFunction) Description() string {
	return "Updates the karma value for the given user."
}

func (f *karmaSetFunction) Triggers() []string {
	return []string{}
}

func (f *karmaSetFunction) Usages() []string {
	return []string{"<user>++ [<reason>]", "<user>-- [<reason>]"}
}

func (f *karmaSetFunction) AllowedInPrivateMessages() bool {
	return false
}

func (f *karmaSetFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0) && (strings.Contains(e.Message(), "++") || strings.Contains(e.Message(), "--"))
}

var karmaRegex = regexp.MustCompile(`(?i)(.*?)\s*(\+\+|--)(?:\s*,?\s+(.*))?`)

func (f *karmaSetFunction) Execute(e *irc.Event) {
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

	f.authorizer.UserStatus(e.ReplyTarget(), to, func(user *irc.User) {
		if user == nil {
			logger.Debugf(e, "ignoring invalid karma target: %s", to)
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
		if len(reason) > maxKarmaReasonLength {
			reason = reason[:maxKarmaReasonLength]
		}

		log.Logger().Infof(e, "⚡ %s [%s/%s] %s %s %s", f.Name(), e.From, e.ReplyTarget(), to, op, reason)

		fs := firestore.Get()
		karma, err := fs.AddKarmaHistory(f.ctx, e.ReplyTarget(), e.From, to, op, reason)
		if err != nil {
			logger.Errorf(e, "error updating karma: %s", err)
			return
		}

		f.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s now has a karma of %s.", style.Bold(to), style.Bold(fmt.Sprintf("%d", karma))))
	})
}
