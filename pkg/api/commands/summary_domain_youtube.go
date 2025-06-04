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
							}
						}
					}
				}
			}
		}
	}

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

	err = json.Unmarshal([]byte(matches[1]), &ytData)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal ytInitialData for %s: %s", url, err)
	}

	title := ""
	channel := ""
	views := ""
	published := ""
	username := ""

	for _, panel := range ytData.EngagementPanels {
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
					if len(views) > 0 {
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
			} else {
				views = views + " views"
			}

			p := strings.TrimSpace(item.VideoDescriptionHeaderRenderer.PublishDate.SimpleText)
			t, err := time.Parse("Jan 2, 2006", p)
			if err == nil {
				published = elapse.PastTimeDescription(t)
			} else {
				published = p
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
