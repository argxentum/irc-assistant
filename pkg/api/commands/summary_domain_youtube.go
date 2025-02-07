package commands

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ytInitialDataRegexp = regexp.MustCompile(`ytInitialData = (.*?);\s*</script>`)

func (c *SummaryCommand) parseYouTube(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/shorts/") {
		return c.parseYouTubeShort(e, url)
	} else {
		return c.parseYouTubeVideo(e, url)
	}
}

func (c *SummaryCommand) parseYouTubeShort(e *irc.Event, url string) (*summary, error) {
	var ytData struct {
		EngagementPanels []struct {
			EngagementPanelSectionListRenderer struct {
				Content struct {
					StructuredDescriptionContentRenderer struct {
						Items []struct {
							VideoDescriptionHeaderRenderer struct {
								ChannelNavigationEndpoint struct {
									BrowseEndpoint struct {
										CanonicalBaseUrl string
									}
								}
								Title struct {
									Runs []struct {
										Text string
									}
								}
								Views struct {
									SimpleText string
								}
							}
						}
					}
				}
			}
		}
	}

	body, err := c.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s: %s", url, err)
	}

	if body == nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s", url)
	}

	html := string(body)
	matches := ytInitialDataRegexp.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("unable to find ytInitialData for %s", url)
	}

	err = json.Unmarshal([]byte(matches[1]), &ytData)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal ytInitialData for %s: %s", url, err)
	}

	if len(ytData.EngagementPanels) < 2 {
		return nil, fmt.Errorf("unable to parse YouTube short data for %s", url)
	}

	items := ytData.EngagementPanels[1].EngagementPanelSectionListRenderer.Content.StructuredDescriptionContentRenderer.Items
	if len(items) == 0 {
		return nil, fmt.Errorf("no YouTube items found for %s", url)
	}

	title := strings.TrimSpace(items[0].VideoDescriptionHeaderRenderer.Title.Runs[0].Text)
	views := strings.TrimSpace(items[0].VideoDescriptionHeaderRenderer.Views.SimpleText)
	views = strings.TrimSuffix(views, " views")
	author := strings.TrimPrefix(items[0].VideoDescriptionHeaderRenderer.ChannelNavigationEndpoint.BrowseEndpoint.CanonicalBaseUrl, "/")

	messages := make([]string, 0)
	if len(title) > 0 && len(views) > 0 && len(author) > 0 {
		messages = append(messages, fmt.Sprintf("%s • %s • %s views", style.Bold(title), author, views))
	} else if len(title) > 0 && len(views) > 0 {
		messages = append(messages, fmt.Sprintf("%s • %s views", style.Bold(title), views))
	} else if len(title) > 0 {
		messages = append(messages, fmt.Sprintf("%s", style.Bold(title)))
	}

	if len(author) > 0 {
		source, err := repository.FindSource(strings.TrimPrefix(author, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if source != nil {
			messages = append(messages, repository.ShortSourceSummary(source))
		}
	}

	return createSummary(messages...), nil
}

func (c *SummaryCommand) parseYouTubeVideo(e *irc.Event, url string) (*summary, error) {
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
				ViewCount struct {
					SimpleText string
				}
				ShortViewCount struct {
					SimpleText string
				}
				ExtraShortViewCount struct {
					SimpleText string
				}
				OriginalViewCount string
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
				NavigationEndpoint struct {
					BrowseEndpoint struct {
						CanonicalBaseUrl string
					}
				}
			}
		}
	}

	body, err := c.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s: %s", url, err)
	} else if body == nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s", url)
	}

	html := string(body)
	matches := ytInitialDataRegexp.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("unable to find ytInitialData for %s", url)
	}

	err = json.Unmarshal([]byte(matches[1]), &ytData)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal ytInitialData for %s: %s", url, err)
	}

	ytResults := ytData.Contents.TwoColumnWatchNextResults.Results.Results.Contents
	if len(ytResults) == 0 {
		return nil, fmt.Errorf("no YouTube results found for %s", url)
	}

	title := ""
	viewCount := ""
	author := ""
	authorHandle := ""
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
			if len(viewCount) == 0 {
				viewCount = primaryInfo.ViewCount.VideoViewCountRenderer.ExtraShortViewCount.SimpleText
			}
			if len(viewCount) == 0 {
				viewCount = primaryInfo.ViewCount.VideoViewCountRenderer.ViewCount.SimpleText
			}
			if len(viewCount) == 0 {
				n, err := strconv.Atoi(primaryInfo.ViewCount.VideoViewCountRenderer.OriginalViewCount)
				if err == nil {
					viewCount = text.DecorateNumberWithCommas(n)
				}
			}
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
			authorHandle = strings.TrimPrefix(secondaryInfo.Owner.VideoOwnerRenderer.NavigationEndpoint.BrowseEndpoint.CanonicalBaseUrl, "/")
		}

		if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 {
			break
		}
	}

	messages := make([]string, 0)
	viewCount = strings.TrimSuffix(viewCount, " views")

	if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 && len(authorHandle) > 0 {
		messages = append(messages, fmt.Sprintf("%s • %s (%s) • %s views", style.Bold(title), author, authorHandle, viewCount))
	} else if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 {
		messages = append(messages, fmt.Sprintf("%s • %s • %s views", style.Bold(title), author, viewCount))
	} else if len(title) > 0 && len(viewCount) > 0 {
		messages = append(messages, fmt.Sprintf("%s • %s views", style.Bold(title), viewCount))
	} else if len(title) > 0 {
		messages = append(messages, fmt.Sprintf("%s", style.Bold(title)))
	}

	if len(author) > 0 {
		authorSource, err := repository.FindSource(strings.TrimPrefix(author, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		authorHandleSource, err := repository.FindSource(strings.TrimPrefix(authorHandle, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if authorSource != nil {
			messages = append(messages, repository.ShortSourceSummary(authorSource))
		} else if authorHandleSource != nil {
			messages = append(messages, repository.ShortSourceSummary(authorHandleSource))
		}
	}

	return createSummary(messages...), nil
}
