package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/summary"
	"assistant/pkg/log"
	"errors"
	"slices"

	"github.com/bobesa/go-domain-util/domainutil"
)

func (c *SummaryCommand) directRequest(e *irc.Event, doc *retriever.Document) (*summaryResult, error) {
	logger := log.Logger()
	logger.Infof(e, "direct request for %s", doc.URL)

	if slices.Contains(c.cfg.Summary.DisabledDirectDomains, domainutil.Domain(doc.URL)) {
		logger.Debugf(e, "direct requests disabled for domain: %s", domainutil.Domain(doc.URL))
		return nil, nil
	}

	meta := summary.ExtractMetadata(doc.Root)

	s, err := c.createSummaryFromTitleAndDescription(meta.Title, meta.Description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected direct summary title: %s", meta.Title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "direct summary too short - title: %s, description: %s", meta.Title, meta.Description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "direct summary no content - title: %s, description: %s", meta.Title, meta.Description)
		return nil, err
	}

	logger.Debugf(e, "direct request - title: %s, description: %s", meta.Title, meta.Description)
	return s, nil
}
