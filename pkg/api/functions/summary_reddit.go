package functions

import (
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strconv"
	"strings"
)

func parseRedditMessage(url string, doc *goquery.Document) string {
	logger := log.Logger()

	if strings.Contains(strings.ToLower(url), "old.reddit.com") {
		title := doc.Find("meta[property='og:title']").First().AttrOr("content", "")
		if len(title) == 0 {
			logger.Rawf(log.Debug, "could not find og:title meta tag for %s", url)
			return ""
		}

		description := doc.Find("meta[property='og:description']").First().AttrOr("content", "")
		return fmt.Sprintf("%s (%s)", style.Bold(strings.TrimSpace(title)), strings.TrimSpace(description))
	}

	post := doc.Find("shreddit-post")
	if post == nil {
		logger.Rawf(log.Debug, "could not find shreddit-post element for %s", url)
		return ""
	}

	title := post.AttrOr("post-title", "")
	if len(title) == 0 {
		logger.Rawf(log.Debug, "could not find post-title attribute for %s", url)
		return ""
	}

	subreddit := post.AttrOr("subreddit-prefixed-name", "")
	author := post.AttrOr("author", "")
	score, _ := strconv.Atoi(strings.TrimSpace(post.AttrOr("score", "")))
	comments, _ := strconv.Atoi(strings.TrimSpace(post.AttrOr("comment-count", "")))
	description := fmt.Sprintf("Posted in %s by u/%s • %s points and %s comments", subreddit, author, text.DecorateNumberWithCommas(score), text.DecorateNumberWithCommas(comments))
	return fmt.Sprintf("%s (%s)", style.Bold(strings.TrimSpace(title)), strings.TrimSpace(description))
}
