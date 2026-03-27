package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/reddit"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
)

func (p *proxy) handleRedditSearchProxyRequest(task *models.Task) error {
	data := task.Data.(models.ProxyRedditSearchRequestTaskData)
	logger := log.Logger()
	logger.Debugf(nil, "handling proxy reddit search request %s for r/%s %s [sort: %s] in %s", task.ID, data.Subreddit, data.Query, data.Sort, data.Channel)

	ctx := context.NewContext()

	var posts []reddit.PostWithTopComment
	var err error

	switch data.Sort {
	case models.RedditSearchSortNew:
		posts, err = reddit.SearchNewSubredditPosts(ctx, p.cfg, data.Subreddit, data.Query)
	default:
		posts, err = reddit.SearchRelevantSubredditPosts(ctx, p.cfg, data.Subreddit, data.Query)
	}

	if err != nil {
		logger.Errorf(nil, "error searching reddit: %s", err)
		return err
	}

	postsAny := make([]any, len(posts))
	for i, post := range posts {
		postsAny[i] = post
	}

	responseTask := models.NewProxyRedditSearchResponseTask(data.Channel, data.Nick, data.Subreddit, data.Query, postsAny)
	return queue.GetDefault().Publish(responseTask)
}
