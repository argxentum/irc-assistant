package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/summary"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const oEmbedRequestTimeout = 5 * time.Second
const youTubeOEmbedURL = "https://www.youtube.com/oembed?url=%s&format=json"
const tikTokOEmbedURL = "https://www.tiktok.com/oembed?url=%s"
const instagramOEmbedURL = "https://www.instagram.com/api/v1/oembed/?url=%s"
const redditOEmbedURL = "https://www.reddit.com/oembed?url=%s"

type oEmbedData struct {
	Title          string `json:"title"`
	AuthorName     string `json:"author_name"`
	AuthorURL      string `json:"author_url"`
	AuthorUniqueID string `json:"author_unique_id"`
	ProviderName   string `json:"provider_name"`
}

func (c *SummaryCommand) oEmbedSummary(e *irc.Event, endpoint, originalURL string) (*summaryResult, *oEmbedData, error) {
	client := &http.Client{Timeout: oEmbedRequestTimeout}
	return c.oEmbedSummaryWithClient(e, client, endpoint, originalURL)
}

func (c *SummaryCommand) oEmbedSummaryWithClient(e *irc.Event, client *http.Client, endpoint, originalURL string) (*summaryResult, *oEmbedData, error) {
	requestURL := fmt.Sprintf(endpoint, url.QueryEscape(originalURL))
	resp, err := client.Get(requestURL)
	if err != nil {
		return nil, nil, fmt.Errorf("oEmbed request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("oEmbed request returned status %d", resp.StatusCode)
	}

	var data oEmbedData
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, nil, fmt.Errorf("unable to decode oEmbed response: %w", err)
	}
	result, err := createOEmbedSummary(data)
	return result, &data, err
}

func findOEmbedSource(e *irc.Event, data *oEmbedData) (*models.Source, error) {
	if data == nil {
		return nil, nil
	}

	logger := log.Logger()
	provider := strings.TrimSpace(data.ProviderName)
	if provider == "" {
		provider = "oEmbed"
	}

	identities := oEmbedSourceIdentities(*data)
	logger.Debugf(e, "%s oEmbed source check: trying identities %v", provider, identities)
	source, identity, err := repository.FindSourceByIdentities(identities)
	if err != nil {
		return nil, fmt.Errorf("error checking %s oEmbed source identities: %w", provider, err)
	}
	if source != nil {
		logger.Debugf(e, "%s oEmbed source check: matched identity %s to source %s", provider, identity, source.ID)
		return source, nil
	}

	logger.Debugf(e, "%s oEmbed source check: no configured source matched", provider)
	return nil, nil
}

func oEmbedSourceIdentities(data oEmbedData) []string {
	identities := make([]string, 0, 4)
	seen := make(map[string]struct{})
	add := func(identity string) {
		identity = strings.TrimSpace(identity)
		if identity == "" {
			return
		}
		key := strings.ToLower(identity)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		identities = append(identities, identity)
	}

	if authorURL, err := url.Parse(strings.TrimSpace(data.AuthorURL)); err == nil && authorURL.Hostname() != "" {
		host := strings.TrimPrefix(strings.ToLower(authorURL.Hostname()), "www.")
		authorPath := strings.Trim(authorURL.Path, "/")
		if authorPath != "" {
			add(host + "/" + authorPath)
			if handle, unescapeErr := url.PathUnescape(path.Base(authorURL.Path)); unescapeErr == nil {
				add(strings.TrimPrefix(handle, "@"))
			}
		}
	}

	add(strings.TrimPrefix(data.AuthorUniqueID, "@"))
	add(data.AuthorName)
	return identities
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
