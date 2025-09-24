package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
)

const communityNoteMaxLength = 300

func createCommunityNoteOutputMessages(e *irc.Event, n *models.CommunityNote) []string {
	logger := log.Logger()

	if len(n.Content) > communityNoteMaxLength {
		n.Content = n.Content[:communityNoteMaxLength] + "..."
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s %s: %s (via %s, %s • %s)", "\u2139\uFE0F", style.Italics(style.Bold("Factual note")), style.Italics(n.Content), n.Author, elapse.PastTimeDescription(n.NotedAt), n.ID))

	if len(n.CounterSources) > 0 {
		source := n.CounterSources[0]
		messages = append(messages, source)

		s, err := repository.FindSource(source)
		if err != nil {
			logger.Errorf(e, "error finding source, %s", err)
		}

		if s != nil {
			messages = append(messages, repository.ShortSourceSummary(s))
		}
	}

	return messages
}
