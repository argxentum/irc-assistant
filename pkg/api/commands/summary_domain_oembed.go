package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/summary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const oEmbedRequestTimeout = 5 * time.Second
const youTubeOEmbedURL = "https://www.youtube.com/oembed?url=%s&format=json"
const tikTokOEmbedURL = "https://www.tiktok.com/oembed?url=%s"

type oEmbedData struct {
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
}

func (c *SummaryCommand) oEmbedSummary(e *irc.Event, endpoint, originalURL string) (*summaryResult, string, error) {
	client := &http.Client{Timeout: oEmbedRequestTimeout}
	return c.oEmbedSummaryWithClient(e, client, endpoint, originalURL)
}

func (c *SummaryCommand) oEmbedSummaryWithClient(e *irc.Event, client *http.Client, endpoint, originalURL string) (*summaryResult, string, error) {
	requestURL := fmt.Sprintf(endpoint, url.QueryEscape(originalURL))
	resp, err := client.Get(requestURL)
	if err != nil {
		return nil, "", fmt.Errorf("oEmbed request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("oEmbed request returned status %d", resp.StatusCode)
	}

	var data oEmbedData
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, "", fmt.Errorf("unable to decode oEmbed response: %w", err)
	}
	result, err := createOEmbedSummary(data)
	return result, strings.TrimSpace(data.AuthorName), err
}

func createOEmbedSummary(data oEmbedData) (*summaryResult, error) {
	title := summary.Sanitize(data.Title)
	author := summary.Sanitize(data.AuthorName)
	if title == "" {
		return nil, noContentError
	}

	message := style.Bold(title)
	if author != "" {
		message += " • " + author
	}
	return createSummaryResult(message), nil
}
