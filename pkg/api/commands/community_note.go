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

func createCommunityNoteOutputMessages(e *irc.Event, n *models.CommunityNote, includeCounterSourceURL bool) []string {
	logger := log.Logger()

	if len(n.Content) > communityNoteMaxLength {
		n.Content = n.Content[:communityNoteMaxLength] + "..."
	}

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s %s: %s (via %s, %s • %s)", "\u2139\uFE0F", style.Italics(style.Bold("Factual note")), style.Italics(n.Content), n.Author, elapse.PastTimeDescription(n.NotedAt), n.ID))

	if includeCounterSourceURL && len(n.CounterSources) > 0 {
		counterSource := n.CounterSources[0]
		messages = append(messages, counterSource)

		cs, err := repository.FindSource(counterSource)
		if err != nil {
			logger.Errorf(e, "error finding counter-source, %s", err)
		}

		if cs != nil {
			messages = append(messages, repository.ShortSourceSummary(cs))
		}
	}

	return messages
}
