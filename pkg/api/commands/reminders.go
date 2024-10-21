package commands

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

const remindersCommandName = "reminders"

const (
	actionCancel = "cancel"
	actionRemove = "remove"
	actionDelete = "delete"
)

type remindersCommand struct {
	*commandStub
}

func NewRemindersCommand(ctx context.Context, cfg *config.Config, ircSvc irc.IRC) Command {
	return &remindersCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircSvc),
	}
}

func (c *remindersCommand) Name() string {
	return remindersCommandName
}

func (c *remindersCommand) Description() string {
	return "Show or cancel reminders."
}

func (c *remindersCommand) Triggers() []string {
	return []string{"reminders"}
}

func (c *remindersCommand) Usages() []string {
	return []string{"reminders", "reminders cancel <number>"}
}

func (c *remindersCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *remindersCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *remindersCommand) Execute(e *irc.Event) {
	tokens := Tokens(e.Message())
	log.Logger().Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), strings.Join(tokens, " "))

	if len(tokens) == 1 {
		c.showReminders(e)
		return
	}

	action := strings.ToLower(strings.TrimSpace(tokens[1]))
	if len(tokens) > 2 && len(action) > 0 && slices.Contains([]string{actionCancel, actionRemove, actionDelete}, action) {
		number, err := strconv.Atoi(strings.ToLower(strings.TrimSpace(tokens[2])))
		if err != nil {
			c.Replyf(e, "Invalid reminder number.")
			return
		}

		c.cancelReminder(e, number)
	} else {
		c.showReminders(e)
	}
}

func (c *remindersCommand) showReminders(e *irc.Event) {
	fs := firestore.Get()

	reminders, err := fs.GetPendingStatusTasks(e.From, e.ReplyTarget(), models.TaskTypeReminder)
	if err != nil {
		log.Logger().Errorf(e, "error getting reminders, %s", err)
		return
	}

	if len(reminders) == 0 {
		if e.IsPrivateMessage() {
			c.Replyf(e, "No reminders found.")
		} else {
			c.Replyf(e, "You have no active reminders in %s.", e.From)
		}
		return
	}

	for i, reminder := range reminders {
		data := reminder.Data.(models.ReminderTaskData)
		c.Replyf(e, "%s: %s (due %s)", style.Bold(fmt.Sprintf("Reminder %d", i+1)), data.Content, elapse.TimeDescription(reminder.DueAt))
	}
}

func (c *remindersCommand) cancelReminder(e *irc.Event, number int) {
	fs := firestore.Get()
	reminders, err := fs.GetPendingStatusTasks(e.From, e.ReplyTarget(), models.TaskTypeReminder)
	if err != nil {
		log.Logger().Errorf(e, "error getting reminders, %s", err)
		return
	}

	if len(reminders) == 0 {
		if e.IsPrivateMessage() {
			c.Replyf(e, "No reminders found.")
		} else {
			c.Replyf(e, "You have no active reminders in %s.", e.From)
		}
		return
	}

	if number < 1 || number > len(reminders) {
		c.Replyf(e, "Invalid reminder number.")
		return
	}

	reminder := reminders[number-1]
	if err := fs.RemoveTask(reminder.ID, models.TaskStatusCancelled); err != nil {
		log.Logger().Errorf(e, "error cancelling reminder, %s", err)
		return
	}

	c.Replyf(e, "Reminder %d cancelled.", number)
}
