package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/reddit"
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

	domain := domainutil.Domain(data.URL)
	ctx := context.NewContext()

	var messages []string
	var err error

	switch domain {
	case "reddit.com":
		messages, err = reddit.Summarize(ctx, p.cfg, data.URL)
	default:
		return fmt.Errorf("unsupported proxy summary domain: %s", domain)
	}

	if err != nil {
		logger.Errorf(nil, "error summarizing %s: %s", data.URL, err)
		return err
	}

	if len(messages) == 0 {
		logger.Debugf(nil, "no summary content for %s", data.URL)
		return nil
	}

	responseTask := models.NewProxySummaryResponseTask(data.Channel, data.Nick, data.URL, messages)
	return queue.GetDefault().Publish(responseTask)
}
