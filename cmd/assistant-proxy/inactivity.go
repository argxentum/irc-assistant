package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/reddit"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
)

func (p *proxy) handleInactivityProxyRequest(task *models.Task) error {
	data := task.Data.(models.ProxyInactivityRequestTaskData)
	logger := log.Logger()
	logger.Debugf(nil, "handling proxy inactivity request %s for r/%s/%s in %s", task.ID, data.Subreddit, data.Category, data.Channel)

	ctx := context.NewContext()
	posts, err := reddit.SubredditCategoryPostsWithTopComment(ctx, p.cfg, data.Subreddit, data.Category, data.Count)
	if err != nil {
		logger.Errorf(nil, "error getting subreddit posts: %s", err)
		return err
	}

	if len(posts) == 0 {
		logger.Debugf(nil, "no posts found for r/%s/%s", data.Subreddit, data.Category)
		return nil
	}

	// Convert to []any for JSON serialization
	postsAny := make([]any, len(posts))
	for i, post := range posts {
		postsAny[i] = post
	}

	responseTask := models.NewProxyInactivityResponseTask(data.Channel, postsAny)
	return queue.GetDefault().Publish(responseTask)
}
