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
	"slices"
	"strconv"
	"strings"
)

const remindersFunctionName = "reminders"

const (
	actionCancel = "cancel"
	actionRemove = "remove"
	actionDelete = "delete"
)

type remindersFunction struct {
	*functionStub
}

func NewRemindersFunction(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Function {
	return &remindersFunction{
		functionStub: defaultFunctionStub(ctx, cfg, ircSvc),
	}
}

func (f *remindersFunction) Name() string {
	return remindersFunctionName
}

func (f *remindersFunction) Description() string {
	return "Show or cancel reminders."
}

func (f *remindersFunction) Triggers() []string {
	return []string{"reminders"}
}

func (f *remindersFunction) Usages() []string {
	return []string{"reminders", "reminders cancel <number>"}
}

func (f *remindersFunction) AllowedInPrivateMessages() bool {
	return true
}

func (f *remindersFunction) CanExecute(e *irc.Event) bool {
	return f.isFunctionEventValid(f, e, 0)
}

func (f *remindersFunction) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", f.Name(), e.From, e.ReplyTarget(), strings.Join(tokens, " "))

	if len(tokens) == 1 {
		f.showReminders(e)
		return
	}

	action := strings.ToLower(strings.TrimSpace(tokens[1]))
	if len(tokens) > 2 && len(action) > 0 && slices.Contains([]string{actionCancel, actionRemove, actionDelete}, action) {
		number, err := strconv.Atoi(strings.ToLower(strings.TrimSpace(tokens[2])))
		if err != nil {
			f.Replyf(e, "Invalid reminder number.")
			return
		}

		f.cancelReminder(e, number)
	} else {
		f.showReminders(e)
	}
}

func (f *remindersFunction) showReminders(e *irc.Event) {
	fs := firestore.Get()

	reminders, err := fs.GetPendingStatusTasks(e.From, e.ReplyTarget(), models.TaskTypeReminder)
	if err != nil {
		log.Logger().Errorf(e, "error getting reminders, %s", err)
		return
	}

	if len(reminders) == 0 {
		if e.IsPrivateMessage() {
			f.Replyf(e, "No reminders found.")
		} else {
			f.Replyf(e, "You have no active reminders in %s.", e.From)
		}
		return
	}

	for i, reminder := range reminders {
		data := reminder.Data.(models.ReminderTaskData)
		f.Replyf(e, "%s: %s (due %s)", style.Bold(fmt.Sprintf("Reminder %d", i+1)), data.Content, elapse.TimeDescription(reminder.DueAt))
	}
}

func (f *remindersFunction) cancelReminder(e *irc.Event, number int) {
	fs := firestore.Get()
	reminders, err := fs.GetPendingStatusTasks(e.From, e.ReplyTarget(), models.TaskTypeReminder)
	if err != nil {
		log.Logger().Errorf(e, "error getting reminders, %s", err)
		return
	}

	if len(reminders) == 0 {
		if e.IsPrivateMessage() {
			f.Replyf(e, "No reminders found.")
		} else {
			f.Replyf(e, "You have no active reminders in %s.", e.From)
		}
		return
	}

	if number < 1 || number > len(reminders) {
		f.Replyf(e, "Invalid reminder number.")
		return
	}

	reminder := reminders[number-1]
	if err := fs.RemoveTask(reminder.ID, models.TaskStatusCancelled); err != nil {
		log.Logger().Errorf(e, "error cancelling reminder, %s", err)
		return
	}

	f.Replyf(e, "Reminder %d cancelled.", number)
}
