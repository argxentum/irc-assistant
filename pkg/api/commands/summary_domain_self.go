package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/log"
	"fmt"
	"regexp"
	"strings"
)

const shortcutURLPattern = "%s/s/"

var urlRegex = regexp.MustCompile(`(https?://\S+)`)

func (c *SummaryCommand) parseShortcut(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()
	logger.Infof(e, "performing shortcut URL lookup: %s", url)

	id := strings.TrimPrefix(url, fmt.Sprintf(shortcutURLPattern, c.cfg.Web.ExternalRootURL))
	source, err := repository.GetShortcutSource(id)
	if err != nil {
		return nil, err
	}

	if len(source) == 0 {
		return nil, fmt.Errorf("can't find shortcut with source %s", id)
	}

	if !urlRegex.MatchString(source) {
		return nil, fmt.Errorf("invalid URL: %s", source)
	}

	if len(e.Arguments) < 2 {
		return nil, fmt.Errorf("missing URL argument")
	}

	e.Arguments[1] = strings.ReplaceAll(e.Arguments[1], url, source)

	registry.Command(SummaryCommandName).Execute(e)
	return nil, nil
}
