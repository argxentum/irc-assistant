package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ytInitialDataRegexp = regexp.MustCompile(`ytInitialData = (.*?);\s*</script>`)
var numberRegexp = regexp.MustCompile(`(\d+(?:,\d{3})*)`)

func (c *SummaryCommand) parseYouTube(e *irc.Event, url string) (*summary, error) {
	body, err := c.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s: %s", url, err)
	}

	if body == nil {
		return nil, fmt.Errorf("unable to retrieve YouTube summary for %s", url)
	}

	html := string(body.Data)
	matches := ytInitialDataRegexp.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("unable to find ytInitialData for %s", url)
	}

	var data ytData
	err = json.Unmarshal([]byte(matches[1]), &data)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal post JSON for %s: %s", url, err)
	}

	if strings.Contains(url, "/post/") {
		s, err := c.parseYouTubePost(e, data)
		if s == nil || err != nil {
			log.Logger().Debugf(e, "error parsing JSON as YouTube post: %v", err)
		} else {
			return s, nil
		}
	}

	return c.parseYouTubeVideo(e, data)
}

func (c *SummaryCommand) parseYouTubeVideo(e *irc.Event, data ytData) (*summary, error) {
	title := ""
	channel := ""
	views := ""
	published := ""
	username := ""

	for _, panel := range data.EngagementPanels {
		if len(panel.EngagementPanelSectionListRenderer.Content.StructuredDescriptionContentRenderer.Items) == 0 {
			continue
		}

		for _, item := range panel.EngagementPanelSectionListRenderer.Content.StructuredDescriptionContentRenderer.Items {
			if len(item.VideoDescriptionHeaderRenderer.Title.Runs) == 0 {
				continue
			}

			title = strings.TrimSpace(item.VideoDescriptionHeaderRenderer.Title.Runs[0].Text)
			channel = strings.TrimSpace(item.VideoDescriptionHeaderRenderer.Channel.SimpleText)
			username = strings.TrimPrefix(item.VideoDescriptionHeaderRenderer.ChannelNavigationEndpoint.BrowseEndpoint.CanonicalBaseUrl, "/")

			views = shortenViewCount(strings.TrimSpace(item.VideoDescriptionHeaderRenderer.Views.SimpleText))
			if len(views) == 0 {
				for _, run := range item.VideoDescriptionHeaderRenderer.Views.Runs {
					if len(views) > 0 && !strings.HasSuffix(views, " ") {
						views += " "
					}
					views += run.Text
				}

				views = strings.TrimSpace(views)
				if len(views) > 0 {
					m := numberRegexp.FindStringSubmatch(views)
					if len(m) > 1 {
						views = strings.Replace(views, m[1], shortenViewCount(m[1]), 1)
					}
				}
			} else if !strings.HasSuffix(views, "views") {
				views = views + " views"
			}

			if len(item.VideoDescriptionHeaderRenderer.Factoid) > 0 {
				for _, factoid := range item.VideoDescriptionHeaderRenderer.Factoid {
					if len(factoid.UploadTimeFactoidRenderer.Factoid.FactoidRenderer.AccessibilityText) > 0 {
						published = strings.TrimSpace(factoid.UploadTimeFactoidRenderer.Factoid.FactoidRenderer.AccessibilityText)
						break
					}
				}
			}

			if len(published) == 0 && len(item.VideoDescriptionHeaderRenderer.PublishDate.SimpleText) > 0 {
				p := strings.TrimSpace(item.VideoDescriptionHeaderRenderer.PublishDate.SimpleText)
				t, err := time.Parse("Jan 2, 2006", p)
				if err == nil && t.Before(time.Now().Add(-24*time.Hour)) {
					now := time.Now()
					from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
					published = elapse.PastTimeDescriptionFromTime(t, from)
				} else {
					published = p
				}
			}
		}

		if len(title) > 0 && len(channel) > 0 {
			break
		}
	}

	messages := make([]string, 0)
	message := ""
	if len(title) > 0 {
		message = style.Bold(title)
	} else {
		return createSummary(), nil
	}

	if len(channel) > 0 {
		message += fmt.Sprintf(" • %s", channel)
	}
	if len(views) > 0 {
		message += fmt.Sprintf(" • %s", views)
	}
	if len(published) > 0 {
		message += fmt.Sprintf(" • %s", published)
	}

	messages = append(messages, message)

	if len(username) > 0 {
		source, err := repository.FindSource(strings.TrimPrefix(username, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if source != nil {
			messages = append(messages, repository.ShortSourceSummary(source))
		}
	}

	return createSummary(messages...), nil
}

func shortenViewCount(input string) string {
	views, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(input, ",", ""), "views")))
	if err != nil {
		return input
	}

	if views < 1000 {
		return fmt.Sprintf("%d", views)
	} else if views < 1000000 {
		return fmt.Sprintf("%.1fK", float64(views)/1000)
	} else if views < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(views)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(views)/1000000000)
}

func (c *SummaryCommand) parseYouTubePost(e *irc.Event, data ytData) (*summary, error) {
	tabs := data.Contents.TwoColumnBrowseResultsRenderer.Tabs
	if len(tabs) == 0 {
		return nil, nil
	}

	sectionListRenderers := tabs[0].TabRenderer.Content.SectionListRenderer.Contents
	if len(sectionListRenderers) == 0 {
		return nil, nil
	}

	itemSectionRenderers := sectionListRenderers[0].ItemSectionRenderer.Contents
	if len(itemSectionRenderers) == 0 {
		return nil, nil
	}

	post := itemSectionRenderers[0].BackstagePostThreadRenderer.Post.BackstagePostRenderer

	author := ""
	if len(post.AuthorText.Runs) > 0 {
		author = strings.TrimSpace(post.AuthorText.Runs[0].Text)
	}

	username := ""
	if len(post.AuthorEndpoint.BrowseEndpoint.CanonicalBaseUrl) > 0 {
		username = strings.TrimPrefix(post.AuthorEndpoint.BrowseEndpoint.CanonicalBaseUrl, "/")
	}

	description := ""
	if len(post.ContentText.Runs) > 0 {
		for _, run := range post.ContentText.Runs {
			if len(description) > 0 && !strings.HasSuffix(description, " ") {
				description += " "
			}
			description += run.Text
		}
	}
	description = strings.TrimSpace(description)

	published := ""
	if len(post.PublishedTimeText.Runs) > 0 {
		published = strings.TrimSpace(post.PublishedTimeText.Runs[0].Text)
	}

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	messages := make([]string, 0)
	message := ""

	if len(description) > 0 {
		message = style.Bold(description)
	} else {
		return nil, nil
	}

	if len(author) > 0 {
		message = fmt.Sprintf("%s • %s", message, author)
	}

	if len(published) > 0 {
		message = fmt.Sprintf("%s • %s", message, published)
	}

	messages = append(messages, message)

	if len(username) > 0 {
		source, err := repository.FindSource(strings.TrimPrefix(username, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}

		if source != nil {
			messages = append(messages, repository.ShortSourceSummary(source))
		}
	}

	return createSummary(messages...), nil
}

type ytData struct {
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
							Channel struct {
								SimpleText string
							}
							Views struct {
								SimpleText string
								Runs       []struct {
									Text string
								}
							}
							PublishDate struct {
								SimpleText string
							}
							Factoid []struct {
								UploadTimeFactoidRenderer struct {
									Factoid struct {
										FactoidRenderer struct {
											AccessibilityText string
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
	Contents struct {
		TwoColumnBrowseResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Content struct {
						SectionListRenderer struct {
							Contents []struct {
								ItemSectionRenderer struct {
									Contents []struct {
										BackstagePostThreadRenderer struct {
											Post struct {
												BackstagePostRenderer struct {
													AuthorText struct {
														Runs []struct {
															Text string
														}
													}
													AuthorEndpoint struct {
														BrowseEndpoint struct {
															CanonicalBaseUrl string
														}
													}
													ContentText struct {
														Runs []struct {
															Text string
														}
													}
													PublishedTimeText struct {
														Runs []struct {
															Text string
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
				}
			}
		}
	}
}
