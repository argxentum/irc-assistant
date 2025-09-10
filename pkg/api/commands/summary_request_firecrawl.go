package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const firecrawlAPIURL = "https://api.firecrawl.dev/v2/scrape"

type firecrawlRequestBody struct {
	URL     string   `json:"url"`
	Formats []string `json:"formats"`
}

type firecrawlResponseBody struct {
	Data struct {
		Metadata struct {
			Title1 string `json:"title"`
			Title2 string `json:"ogTitle"`
			Title3 string `json:"og:title"`
			Title4 string `json:"twitter:title"`
		}
	}
}

func (c *SummaryCommand) firecrawlRequest(e *irc.Event, doc *retriever.Document) (*summary, error) {
	url := doc.URL
	logger := log.Logger()
	logger.Infof(e, "trying firecrawl for %s", url)

	reqBody := firecrawlRequestBody{
		URL:     url,
		Formats: []string{"json"},
	}

	b, err := json.Marshal(&reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body, %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, firecrawlAPIURL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request, %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authentication", "Bearer "+c.cfg.Firecrawl.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request, %w", err)
	}

	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body, %w", err)
	}

	var respBody *firecrawlResponseBody
	err = json.Unmarshal(b, &respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body, %w", err)
	}

	if respBody == nil {
		return nil, fmt.Errorf("response is nil")
	}

	title := coalesce(respBody.Data.Metadata.Title1, respBody.Data.Metadata.Title2, respBody.Data.Metadata.Title3, respBody.Data.Metadata.Title4)

	if c.isRejectedTitle(title) {
		logger.Debugf(e, "rejected firecrawl title: %s", title)
		return nil, rejectedTitleError
	}

	if len(title) < minimumTitleLength {
		logger.Debugf(e, "firecrawl title too short: %s", title)
		return nil, summaryTooShortError
	}

	return createSummary(style.Bold(title)), nil
}
