package functions

import (
	"assistant/pkg/api/style"
	"encoding/json"
	"fmt"
	"regexp"
)

var ytInitialDataRegexp = regexp.MustCompile(`ytInitialData = (.*?);`)

func parseYouTubeMessage(doc string) string {
	var ytData struct {
		Contents struct {
			TwoColumnWatchNextResults struct {
				Results struct {
					Results struct {
						Contents []any
					}
				}
			}
		}
	}

	var primaryInfo struct {
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

	var secondaryInfo struct {
		Owner struct {
			VideoOwnerRenderer struct {
				Title struct {
					Runs []struct {
						Text string
					}
				}
			}
		}
	}

	matches := ytInitialDataRegexp.FindStringSubmatch(doc)
	if len(matches) < 2 {
		return ""
	}

	err := json.Unmarshal([]byte(matches[1]), &ytData)
	if err != nil {
		return ""
	}

	ytResults := ytData.Contents.TwoColumnWatchNextResults.Results.Results.Contents
	if len(ytResults) == 0 {
		return ""
	}

	title := ""
	viewCount := ""
	author := ""
	for _, result := range ytResults {
		if _, ok := result.(map[string]interface{})["videoPrimaryInfoRenderer"]; ok {
			j, err := json.Marshal(result.(map[string]interface{})["videoPrimaryInfoRenderer"])
			if err != nil {
				continue
			}
			err = json.Unmarshal(j, &primaryInfo)
			if err != nil {
				continue
			}
			titles := primaryInfo.Title.Runs
			if len(titles) == 0 {
				continue
			}
			title = titles[0].Text
			viewCount = primaryInfo.ViewCount.VideoViewCountRenderer.ShortViewCount.SimpleText
		}
		if _, ok := result.(map[string]interface{})["videoSecondaryInfoRenderer"]; ok {
			j, err := json.Marshal(result.(map[string]interface{})["videoSecondaryInfoRenderer"])
			if err != nil {
				continue
			}
			err = json.Unmarshal(j, &secondaryInfo)
			if err != nil {
				continue
			}
			authors := secondaryInfo.Owner.VideoOwnerRenderer.Title.Runs
			if len(authors) == 0 {
				continue
			}
			author = authors[0].Text
		}

		if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 {
			break
		}
	}

	if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 {
		return fmt.Sprintf("YouTube: %s - %s (%s)", style.Bold(title), author, viewCount)
	} else if len(title) > 0 && len(viewCount) > 0 {
		return fmt.Sprintf("YouTube: %s (%s)", style.Bold(title), viewCount)
	} else if len(title) > 0 {
		return fmt.Sprintf("YouTube: %s", style.Bold(title))
	}

	return ""
}
