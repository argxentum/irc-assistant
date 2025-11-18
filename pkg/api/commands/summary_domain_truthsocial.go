package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const truthSocialAPIURL = "https://truthsocial.com/api/v1/statuses/%s"

var truthSocialURLRegex = regexp.MustCompile(`^https?://(?:.*?\.)?truthsocial\.com/(.*?)/(\d+)$`)

func (c *SummaryCommand) parseTruthSocial(e *irc.Event, url string) (*summary, *models.Source, error) {
	urlComponents := truthSocialURLRegex.FindStringSubmatch(url)
	if len(urlComponents) < 3 {
		return nil, nil, fmt.Errorf("invalid Truth Social URL: %s", url)
	}

	username := urlComponents[1]
	postID := urlComponents[2]

	params := retriever.DefaultParams(fmt.Sprintf(truthSocialAPIURL, postID))
	params.Headers = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Encoding":           "gzip, deflate, br, zstd",
		"Accept-Language":           "en-US,en;q=0.9",
		"Cookie":                    "__cf_bm=B0JY410q_70vsINDvYJdHPgLbXPPYvoTyjmV7Yng.ak-1749840654-1.0.1.1-Vzs.TTdLmYPWb1yDxdRAv26BR0UwImYa1Yg51pi37pc.fk3WVDISxIYq4yfvsePC9IS9qWR3IlXyuX6bjaAbn06193otKTW_WjTbDc6E4Xc; __cflb=0H28vTPqhjwKvpvovPVDebuz7TcmnBpbTHAZKvot1TM; _cfuvid=dN7vNGEKQKQv9B_ZR8MklnAdbbTQ7_NgRY.GyaNRiLo-1749840654684-0.0.1.1-604800000",
		"Priority":                  "u=0, i",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
	}

	r := retriever.NewBodyRetriever()
	body, err := r.RetrieveBody(e, params)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve Truth Social post %s: %s", postID, err)
	}

	if body.Response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unable to retrieve Truth Social post %s: %s", postID, body.Response.StatusCode)
	}

	var post truthSocialPost
	if err := json.NewDecoder(bytes.NewReader(body.Data)).Decode(&post); err != nil {
		return nil, nil, fmt.Errorf("unable to decode Truth Social post %s: %s", postID, err)
	}

	content := ""
	if len(post.Content) > 0 {
		content = fmt.Sprintf("%s • %s (%s)", style.Bold(post.Content), post.Account.DisplayName, username)
	} else {
		return nil, nil, fmt.Errorf("post content is empty for post %s", postID)
	}

	if len(post.CreatedAt) > 0 {
		at := ""
		if t, err := time.Parse(time.RFC3339, post.CreatedAt); err == nil {
			at = elapse.PastTimeDescription(t)
		}
		if len(at) > 0 {
			content = fmt.Sprintf("%s • %s", content, at)
		}
	}

	return createSummary(content), nil, nil
}

type truthSocialPost struct {
	ID             string
	CreatedAt      string `json:"created_at"`
	URL            string
	Content        string
	FavoritesCount int `json:"favourites_count"`
	UpvotesCount   int `json:"upvotes_count"`
	RepliesCount   int `json:"replies_count"`
	ReblogsCount   int `json:"reblogs_count"`
	Account        struct {
		ID             string
		Username       string
		DisplayName    string `json:"display_name"`
		FollowersCount int    `json:"followers_count"`
	}
}
