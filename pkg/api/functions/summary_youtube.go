package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ytInitialDataRegexp = regexp.MustCompile(`ytInitialData = (.*?);`)

func (f *summaryFunction) parseYouTube(e *irc.Event, url string) (*summary, error) {
	if strings.Contains(url, "/shorts/") {
		return f.parseYouTubeShort(e, url)
	} else {
		return f.parseYouTubeVideo(e, url)
	}
}

func (f *summaryFunction) parseYouTubeShort(e *irc.Event, url string) (*summary, error) {
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

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s: %s", url, err)
	}

	if doc == nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s", url)
	}

	html := doc.Text()
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

	if len(title) > 0 && len(views) > 0 && len(author) > 0 {
		return &summary{fmt.Sprintf("%s • %s • %s views", style.Bold(title), author, views)}, nil
	} else if len(title) > 0 && len(views) > 0 {
		return &summary{fmt.Sprintf("%s • %s views", style.Bold(title), views)}, nil
	} else if len(title) > 0 {
		return &summary{fmt.Sprintf("%s", style.Bold(title))}, nil
	}

	return nil, nil
}

func (f *summaryFunction) parseYouTubeVideo(e *irc.Event, url string) (*summary, error) {
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

	doc, err := f.retriever.RetrieveDocument(e, retriever.DefaultParams(url), retriever.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s: %s", url, err)
	} else if doc == nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s", url)
	}

	html := doc.Text()
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

	viewCount = strings.TrimSuffix(viewCount, " views")

	if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 && len(authorHandle) > 0 {
		return &summary{fmt.Sprintf("%s • %s (%s) • %s views", style.Bold(title), author, authorHandle, viewCount)}, nil
	} else if len(title) > 0 && len(viewCount) > 0 && len(author) > 0 {
		return &summary{fmt.Sprintf("%s • %s • %s views", style.Bold(title), author, viewCount)}, nil
	} else if len(title) > 0 && len(viewCount) > 0 {
		return &summary{fmt.Sprintf("%s • %s views", style.Bold(title), viewCount)}, nil
	} else if len(title) > 0 {
		return &summary{fmt.Sprintf("%s", style.Bold(title))}, nil
	}

	return nil, nil
}
