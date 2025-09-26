package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strconv"
	"strings"
)

const DataManagementCommandName = "data_management"

type commandObject string

const (
	commandObjectSource commandObject = "source"
)

type commandAction string

const (
	commandActionCopy   commandAction = "copy"
	commandActionEdit   commandAction = "edit"
	commandActionDelete commandAction = "delete"
)

type commandField string

const (
	commandFieldNone              commandField = ""
	commandFieldSourceBias        commandField = "bias"
	commandFieldSourceCredibility commandField = "credibility"
	commandFieldSourceFactuality  commandField = "factuality"
	commandFieldSourceIdentity    commandField = "identity"
	commandFieldSourceKeyword     commandField = "keyword"
	commandFieldSourcePaywall     commandField = "paywall"
	commandFieldSourceReference   commandField = "reference"
	commandFieldSourceTitle       commandField = "title"
)

type DataManagementCommand struct {
	*commandStub
}

func NewDataManagementCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &DataManagementCommand{
		commandStub: newCommandStub(ctx, cfg, ircs, RoleAdmin, irc.ChannelStatusNone),
	}
}

func (c *DataManagementCommand) Name() string {
	return DataManagementCommandName
}

func (c *DataManagementCommand) Description() string {
	return fmt.Sprintf("Administrative data management command.")
}

func (c *DataManagementCommand) Triggers() []string {
	return []string{"data"}
}

func (c *DataManagementCommand) Usages() []string {
	return []string{"%s [<channel>] <object> <action> <id> <field> <value>"}
}

func (c *DataManagementCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *DataManagementCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 3)
}

func (c *DataManagementCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	channel := e.ReplyTarget()
	if e.IsPrivateMessage() {
		channel = tokens[1]
		if len(tokens) < 3 || !irc.IsChannel(channel) {
			c.Replyf(e, "Please specify a channel: %s", style.Italics(fmt.Sprintf("%s <channel> <id>", tokens[0])))
			return
		}

		updatedTokens := make([]string, 0)
		for i, token := range tokens {
			if i != 1 {
				updatedTokens = append(updatedTokens, token)
			}
		}
		tokens = updatedTokens
	}

	object := commandObject(strings.TrimSpace(tokens[1]))
	action := commandAction(strings.TrimSpace(tokens[2]))
	id := strings.TrimSpace(tokens[3])

	field := commandFieldNone
	if len(tokens) > 4 {
		field = commandField(strings.TrimSpace(tokens[4]))
	}

	value := ""
	if len(tokens) > 5 {
		value = strings.TrimSpace(strings.Join(tokens[5:], " "))
	}

	logger.Debugf(e, "Performing data management operation in %s: %s %s on %s: %s -> %s", channel, object, action, id, field, value)

	if err := c.manageData(e, channel, object, action, id, field, value); err != nil {
		logger.Errorf(e, "failed to perform data management operation: %s", err)
	}
}

type commandObjectNotFoundError error
type commandActionNotFoundError error
type commandFieldNotFoundError error

func (c *DataManagementCommand) manageData(e *irc.Event, channel string, object commandObject, action commandAction, id string, field commandField, value string) error {
	switch object {
	case commandObjectSource:
		return c.manageSourceData(e, action, id, field, value)
	default:
		return commandObjectNotFoundError(fmt.Errorf("unknown object: %s", object))
	}
}

func (c *DataManagementCommand) manageSourceData(e *irc.Event, action commandAction, id string, field commandField, value string) error {
	switch action {
	case commandActionCopy:
		source, err := c.copySourceData(e, id, field, value)
		if err != nil {
			if source != nil {
				c.Replyf(e, "Something went wrong while creating source copy %s from original %s", style.Bold(source.ID), style.Bold(id))
			} else {
				c.Replyf(e, "Failed to create source copy from original %s", style.Bold(id))
			}
			return err
		}
		c.Replyf(e, "Created source copy %s from original %s", style.Bold(source.ID), style.Bold(id))
		return nil
	case commandActionEdit:
		err := c.editSourceData(e, id, field, value)
		if err != nil {
			c.Replyf(e, "Failed to edit source %s", style.Bold(id))
			return err
		}
		c.Replyf(e, "Edited source %s", style.Bold(id))
		return nil
	case commandActionDelete:
		err := c.deleteSourceData(e, id, field)
		if err != nil {
			c.Replyf(e, "Delete operation on source %s failed", style.Bold(id))
			return err
		}
		c.Replyf(e, "Deleted source %s", style.Bold(id))
		return nil
	default:
		return commandActionNotFoundError(fmt.Errorf("unknown action: %s", action))
	}
}

func (c *DataManagementCommand) copySourceData(e *irc.Event, id string, field commandField, value any) (*models.Source, error) {
	logger := log.Logger()
	fs := firestore.Get()
	orig, err := fs.GetSource(id)
	if err != nil {
		return nil, err
	}

	switch field {
	case commandFieldSourceBias:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Bias = fmt.Sprintf("%s", value)
		return source, fs.SetSource(source)
	case commandFieldSourceCredibility:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Credibility = fmt.Sprintf("%s", value)
		return source, fs.SetSource(source)
	case commandFieldSourceFactuality:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Factuality = fmt.Sprintf("%s", value)
		return source, fs.SetSource(source)
	case commandFieldSourceIdentity:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.URLs = append(source.URLs, fmt.Sprintf("%s", value))
		return source, fs.SetSource(source)
	case commandFieldSourceKeyword:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Keywords = append(source.Keywords, fmt.Sprintf("%s", value))
		return source, fs.SetSource(source)
	case commandFieldSourcePaywall:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		if parseBool, err := strconv.ParseBool(fmt.Sprintf("%s", value)); err == nil {
			source.Paywall = parseBool
			return source, fs.SetSource(source)
		}
		logger.Warningf(e, "Could not parse boolean value %s for field %s", value, field)
		return source, nil
	case commandFieldSourceReference:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Reviews = append(source.Reviews, fmt.Sprintf("%s", value))
		return source, fs.SetSource(source)
	case commandFieldSourceTitle:
		source := models.NewSource("", orig.Bias, orig.Factuality, orig.Credibility, "", []string{}, []string{})
		source.Title = fmt.Sprintf("%s", value)
		return source, fs.SetSource(source)
	default:
		return nil, commandFieldNotFoundError(fmt.Errorf("unknown field: %s", field))
	}
}

func (c *DataManagementCommand) editSourceData(e *irc.Event, id string, field commandField, value any) error {
	fs := firestore.Get()
	source, err := fs.GetSource(id)
	if err != nil {
		return err
	}

	switch field {
	case commandFieldSourceBias:
		return fs.UpdateSource(id, map[string]any{"bias": value})
	case commandFieldSourceCredibility:
		return fs.UpdateSource(id, map[string]any{"credibility": value})
	case commandFieldSourceFactuality:
		return fs.UpdateSource(id, map[string]any{"factuality": value})
	case commandFieldSourceIdentity:
		identities := make([]string, 0)
		if value != nil {
			for _, v := range source.URLs {
				identities = append(identities, v)
			}
			identities = append(identities, fmt.Sprintf("%s", value))
		}
		return fs.UpdateSource(id, map[string]any{"urls": identities})
	case commandFieldSourceKeyword:
		keywords := make([]string, 0)
		if value != nil {
			for _, v := range source.Keywords {
				keywords = append(keywords, v)
			}
			keywords = append(keywords, fmt.Sprintf("%s", value))
		}
		return fs.UpdateSource(id, map[string]any{"keywords": keywords})
	case commandFieldSourcePaywall:
		return fs.UpdateSource(id, map[string]any{"paywall": value})
	case commandFieldSourceReference:
		references := make([]string, 0)
		if value != nil {
			for _, v := range source.Reviews {
				references = append(references, v)
			}
			references = append(references, fmt.Sprintf("%s", value))
		}
		return fs.UpdateSource(id, map[string]any{"reviews": references})
	case commandFieldSourceTitle:
		return fs.UpdateSource(id, map[string]any{"title": value})
	default:
		return commandFieldNotFoundError(fmt.Errorf("unknown field: %s", field))
	}
}

func (c *DataManagementCommand) deleteSourceData(e *irc.Event, id string, field commandField) error {
	fs := firestore.Get()

	if field == commandFieldNone {
		return fs.DeleteSource(id)
	} else {
		return c.editSourceData(e, id, field, nil)
	}
}
