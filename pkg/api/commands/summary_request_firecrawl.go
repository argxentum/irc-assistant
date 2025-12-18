package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"bytes"
	"encoding/json"
	"errors"
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
			Title1       string `json:"cXenseParse:recs:wsj-headline"`
			Title2       string `json:"article.headline"`
			Title3       string `json:"article.origheadline"`
			Title4       string `json:"title"`
			Title5       string `json:"ogTitle"`
			Title6       string `json:"twitter:title"`
			Title7       string `json:"twitter:image:alt"`
			Description1 string `json:"cXenseParse:recs:wsj-summary"`
			Description2 string `json:"article.summary"`
			Description3 string `json:"twitter:description"`
			Description4 string `json:"ogDescription"`
			Description5 string `json:"description"`
		} `json:"metadata"`
	} `json:"data"`
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
		logger.Debugf(e, "error marshalling firecrawl request body, %v", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, firecrawlAPIURL, bytes.NewReader(b))
	if err != nil {
		logger.Debugf(e, "failed to create firecrawl http request, %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.Firecrawl.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Debugf(e, "failed to send firecrawl http request, %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Debugf(e, "failed to read firecrawl response body, %v", err)
		return nil, err
	}

	var respBody *firecrawlResponseBody
	err = json.Unmarshal(b, &respBody)
	if err != nil {
		logger.Debugf(e, "failed to unmarshal firecrawl response body, %v", err)
		return nil, err
	}

	if respBody == nil {
		logger.Debug(e, "firecrawl response is nil")
		return nil, fmt.Errorf("response is nil")
	}

	title := coalesce(
		respBody.Data.Metadata.Title1,
		respBody.Data.Metadata.Title2,
		respBody.Data.Metadata.Title3,
		respBody.Data.Metadata.Title4,
		respBody.Data.Metadata.Title5,
		respBody.Data.Metadata.Title6,
		respBody.Data.Metadata.Title7,
	)

	description := coalesce(
		respBody.Data.Metadata.Description1,
		respBody.Data.Metadata.Description2,
		respBody.Data.Metadata.Description3,
		respBody.Data.Metadata.Description4,
		respBody.Data.Metadata.Description5,
	)

	s, err := c.createSummaryFromTitleAndDescription(title, description)
	if errors.Is(err, rejectedTitleError) {
		logger.Debugf(e, "rejected firecrawl summary title: %s", title)
		return nil, err
	}
	if errors.Is(err, summaryTooShortError) {
		logger.Debugf(e, "firecrawl summary too short - title: %s, description: %s", title, description)
		return nil, err
	}
	if errors.Is(err, noContentError) {
		logger.Debugf(e, "firecrawl summary no content - title: %s, description: %s", title, description)
		return nil, err
	}

	logger.Debugf(e, "firecrawl search request - title: %s, description: %s", title, description)
	return s, nil
}
