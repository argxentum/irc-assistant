package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/reddit"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/summary"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"

	"github.com/bobesa/go-domain-util/domainutil"
)

func (p *proxy) handleSummaryProxyRequest(task *models.Task) error {
	data := task.Data.(models.ProxySummaryRequestTaskData)
	logger := log.Logger()
	logger.Debugf(nil, "handling proxy summary request %s for %s in %s", task.ID, data.URL, data.Channel)

	if summary.IsDomainIgnored(data.URL, p.cfg.Ignore.Domains) {
		logger.Debugf(nil, "domain ignored %s", data.URL)
		return nil
	}

	domain := domainutil.Domain(data.URL)
	ctx := context.NewContext()

	var messages []string
	var err error

	switch domain {
	case "reddit.com":
		messages, err = reddit.Summarize(ctx, p.cfg, data.URL)
	default:
		messages, err = p.summarizeURL(data.URL)
	}

	if err != nil {
		logger.Errorf(nil, "error summarizing %s: %s", data.URL, err)
		// Still publish an empty response so the waiter can unblock
		if data.RequestID != "" {
			responseTask := models.NewProxySummaryResponseTaskWithWaiter(data.RequestID, data.Channel, data.Nick, data.URL, nil)
			_ = queue.GetDefault().Publish(responseTask)
		}
		return err
	}

	if len(messages) == 0 {
		logger.Debugf(nil, "no summary content for %s", data.URL)
		if data.RequestID != "" {
			responseTask := models.NewProxySummaryResponseTaskWithWaiter(data.RequestID, data.Channel, data.Nick, data.URL, nil)
			return queue.GetDefault().Publish(responseTask)
		}
		return nil
	}

	var responseTask *models.Task
	if data.RequestID != "" {
		responseTask = models.NewProxySummaryResponseTaskWithWaiter(data.RequestID, data.Channel, data.Nick, data.URL, messages)
	} else {
		responseTask = models.NewProxySummaryResponseTask(data.Channel, data.Nick, data.URL, messages)
	}
	return queue.GetDefault().Publish(responseTask)
}

const maxProxySummaryTitleLength = 256
const maxProxySummaryDescriptionLength = 300

func (p *proxy) summarizeURL(u string) ([]string, error) {
	logger := log.Logger()
	logger.Debugf(nil, "proxy direct summarization for %s", u)

	br := retriever.NewBodyRetriever()
	dr := retriever.NewDocumentRetriever(br)

	doc, err := dr.RetrieveDocument(nil, retriever.DefaultParams(u))
	if err != nil {
		return nil, fmt.Errorf("error retrieving document: %w", err)
	}

	meta := summary.ExtractMetadata(doc.Root)

	if len(meta.Title) == 0 && len(meta.Description) == 0 {
		return nil, nil
	}

	if summary.IsRejectedTitle(meta.Title, p.cfg.Ignore.TitlePrefixes) {
		logger.Debugf(nil, "rejected proxy summary title: %s", meta.Title)
		return nil, nil
	}

	title := meta.Title
	description := meta.Description

	if len(title) > maxProxySummaryTitleLength {
		title = title[:maxProxySummaryTitleLength]
	}
	if len(description) > maxProxySummaryDescriptionLength {
		description = description[:maxProxySummaryDescriptionLength]
	}

	var message string
	if len(title) > 0 && len(description) > 0 {
		message = fmt.Sprintf("%s: %s", style.Bold(title), description)
	} else if len(title) > 0 {
		message = style.Bold(title)
	} else {
		message = description
	}

	return []string{message}, nil
}
