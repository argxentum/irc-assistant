package reddit

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/summary"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const webURL = "https://www.reddit.com"

var completeDomainPattern = regexp.MustCompile(`https?://((?:.*?\.)?reddit\.com)/`)
var mediaPattern = regexp.MustCompile(`https://(?:www\.)?reddit\.com/media\?url=https.+`)

// Summarize resolves a Reddit URL (including shortlinks and media links)
// and returns formatted summary messages for the post and top comment.
func Summarize(ctx context.Context, cfg *config.Config, url string) ([]string, error) {
	if strings.Contains(url, "/s/") {
		return summarizeShortlink(ctx, cfg, url)
	}

	if mediaPattern.MatchString(url) {
		return summarizeMediaLink(ctx, cfg, url)
	}

	return summarizePost(ctx, cfg, url)
}

func summarizePost(ctx context.Context, cfg *config.Config, url string) ([]string, error) {
	if strings.HasPrefix(url, "https://old.reddit.com/") {
		url = strings.Replace(url, "old.reddit.com", "reddit.com", 1)
	}

	match := completeDomainPattern.FindStringSubmatch(url)
	if len(match) < 2 {
		return nil, fmt.Errorf("unable to parse reddit domain from URL %s", url)
	}

	domain := match[1]
	url = strings.Replace(url, domain, "api.reddit.com", 1)

	post, err := GetPostWithTopComment(ctx, cfg, url)
	if err != nil {
		return nil, err
	}

	if post == nil {
		return nil, errors.New("post not found")
	}

	title := summary.Sanitize(post.Post.Title)
	if len(title) == 0 {
		return nil, nil
	}

	messages := make([]string, 0)
	messages = append(messages, post.Post.FormattedTitle())

	if post.Comment != nil {
		messages = append(messages, post.Comment.FormattedBody())
	}

	return messages, nil
}

func summarizeShortlink(ctx context.Context, cfg *config.Config, url string) ([]string, error) {
	logger := log.Logger()
	logger.Debugf(nil, "reddit shortlink request for %s", url)

	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	// <shreddit-post permalink="/r/Weird/comments/1kqbrm3/..." ... >
	post := doc.Find("shreddit-post").First()
	permalink := strings.TrimSpace(post.AttrOr("permalink", ""))

	return Summarize(ctx, cfg, webURL+permalink)
}

func summarizeMediaLink(ctx context.Context, cfg *config.Config, url string) ([]string, error) {
	logger := log.Logger()
	logger.Debugf(nil, "reddit media request for %s", url)

	if err := Login(ctx, cfg); err != nil {
		return nil, err
	}

	doc, err := fetchDocument(url)
	if err != nil {
		return nil, err
	}

	// <post-bottom-bar permalink="/r/funny/comments/1kpzega/..." ...>
	bottomBar := doc.Find("post-bottom-bar").First()
	permalink := strings.TrimSpace(bottomBar.AttrOr("permalink", ""))

	updatedURL := webURL + permalink
	messages, err := Summarize(ctx, cfg, updatedURL)
	if messages != nil {
		messages = append(messages, updatedURL)
	}

	return messages, err
}

func fetchDocument(url string) (*goquery.Document, error) {
	logger := log.Logger()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	rhs := retriever.RandomHeaderSet()
	for k, v := range rhs {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Debugf(nil, "error fetching %s, %s", url, err)
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("no response")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response for %s: %w", url, err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("unable to parse document for %s: %w", url, err)
	}

	if doc == nil {
		return nil, fmt.Errorf("document nil for %s", url)
	}

	return doc, nil
}
