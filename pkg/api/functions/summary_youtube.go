package functions

import (
	"assistant/pkg/api/text"
	"encoding/json"
	"fmt"
)

func parseYoutube(doc string) string {
	var responseContext struct {
		Contents struct {
			TwoColumnWatchNextResults struct {
				Results struct {
					Results struct {
						Contents []struct {
							VideoPrimaryInfoRenderer struct {
								Title struct {
									Runs []struct {
										Text string
									}
								}
								ViewCount struct {
									VideoViewCountRenderer struct {
										ShortViewCount struct {
											SimpleText string
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	matches := ytInitialDataRegexp.FindStringSubmatch(doc)
	if len(matches) < 2 {
		return ""
	}

	err := json.Unmarshal([]byte(matches[1]), &responseContext)
	if err != nil {
		return ""
	}

	title := responseContext.Contents.TwoColumnWatchNextResults.Results.Results.Contents[0].VideoPrimaryInfoRenderer.Title.Runs[0].Text
	viewCount := responseContext.Contents.TwoColumnWatchNextResults.Results.Results.Contents[0].VideoPrimaryInfoRenderer.ViewCount.VideoViewCountRenderer.ShortViewCount.SimpleText

	if len(title) > 0 && len(viewCount) > 0 {
		return fmt.Sprintf("YouTube: %s (%s)", text.Bold(title), viewCount)
	} else if len(title) > 0 {
		return fmt.Sprintf("YouTube: %s", text.Bold(title))
	}

	return ""
}
