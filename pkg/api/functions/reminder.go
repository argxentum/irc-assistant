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
	*functionStub
}

func NewReminderFunction(ctx context.Context, cfg *config.Config, irc irc.IRC) Function {
	return &reminderFunction{
		functionStub: defaultFunctionStub(ctx, cfg, irc),
	}
}

func (f *reminderFunction) Name() string {
	return reminderFunctionName
}

func (f *reminderFunction) Description() string {
	return "Creates a reminder message to be delivered after the given duration."
}

func (f *reminderFunction) Triggers() []string {
	return []string{"remind", "reminder"}
}

func (f *reminderFunction) Usages() []string {
	return []string{"%s <duration> <message>"}
}

func (f *reminderFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *reminderFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 2)
}

func (f *reminderFunction) Execute(e *irc.Event) {
	logger := log.Logger()

	tokens := Tokens(e.Message())
	duration := tokens[1]
	message := strings.Join(tokens[2:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] @ %s, %s", f.Name(), e.From, e.ReplyTarget(), duration, message)

	seconds, err := elapse.ParseDuration(duration)
	if err != nil {
		logger.Errorf(e, "error parsing duration, %s", err)
		f.Replyf(e, "invalid duration, see %s for help", style.Bold(fmt.Sprintf("%s%s", f.cfg.Functions.Prefix, registry.Function(reminderFunctionName).Triggers()[0])))
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
