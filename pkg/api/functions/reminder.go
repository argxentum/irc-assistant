package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
	"time"
)

const reminderFunctionName = "reminder"

type reminderFunction struct {
	FunctionStub
}

func NewReminderFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, reminderFunctionName)
	if err != nil {
		return nil, err
	}

	return &reminderFunction{
		FunctionStub: stub,
	}, nil
}

func (f *reminderFunction) MayExecute(e *irc.Event) bool {
	return f.isValid(e, 2)
}

func (f *reminderFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	duration := tokens[1]
	message := strings.Join(tokens[2:], " ")
	logger.Infof(e, "⚡ [%s/%s] reminder @ %s, %s", e.From, e.ReplyTarget(), duration, message)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		f.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, f.cfg.Functions.EnabledFunctions[reminderFunctionName].Triggers[0])))
		return
	}

	task := models.NewReminderTask(time.Now().Add(seconds), e.From, e.ReplyTarget(), message)
	err = firestore.Get().AddTask(task)
	if err != nil {
		logger.Errorf(e, "error adding task, %s", err)
		return
	}

	f.Replyf(e, "reminder set for %s", style.Bold(elapse.TimeDescription(task.DueAt)))
}
