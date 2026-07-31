package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/api/style"
	"assistant/pkg/api/summary"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const youTubeMaxRetries = 3
const youTubeRetryDelay = 2 * time.Second

var ytInitialDataAssignmentRegexp = regexp.MustCompile(`(?:(?:var\s+)?ytInitialData|window(?:\s*\[\s*["']ytInitialData["']\s*\]|\s*\.ytInitialData))\s*=\s*`)
var numberRegexp = regexp.MustCompile(`(\d+(?:,\d{3})*)`)
var errYouTubeInitialDataNotFound = errors.New("ytInitialData assignment not found")

func (c *SummaryCommand) parseYouTube(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
	logger := log.Logger()
	logger.Debugf(e, "YouTube summary path: trying oEmbed for %s", url)
	if result, author, err := c.oEmbedSummary(e, youTubeOEmbedURL, url); err == nil && result != nil {
		logger.Debugf(e, "YouTube summary path: oEmbed succeeded for %s", url)
		var src *models.Source
		if author != "" {
			src, err = repository.FindSource(author)
			if err != nil {
				logger.Debugf(e, "YouTube error finding optional source for author %s: %v", author, err)
			}
		}
		return result, src, nil
	} else if err != nil {
		logger.Debugf(e, "YouTube summary path: oEmbed failed for %s: %s; falling back to page parsing", url, err)
	} else {
		logger.Debugf(e, "YouTube summary path: oEmbed returned no usable summary for %s; falling back to page parsing", url)
	}

	logger.Debugf(e, "YouTube summary path: trying page parsing for %s", url)
	var data ytData
	var pageData []byte

	for attempt := 1; attempt <= youTubeMaxRetries; attempt++ {
		logger.Debugf(e, "YouTube summary path: fetching page for %s (attempt %d/%d)", url, attempt, youTubeMaxRetries)
		body, err := c.bodyRetriever.RetrieveBody(e, retriever.DefaultParams(url))
		if err != nil {
			logger.Debugf(e, "YouTube summary path: page retrieval failed for %s (attempt %d/%d): %s", url, attempt, youTubeMaxRetries, err)
		} else if body == nil {
			logger.Debugf(e, "YouTube summary path: page retrieval returned no body for %s (attempt %d/%d)", url, attempt, youTubeMaxRetries)
		} else {
			pageData = body.Data
			if err = decodeYouTubeInitialData(body.Data, &data); errors.Is(err, errYouTubeInitialDataNotFound) {
				logger.Debugf(e, "YouTube summary path: ytInitialData not found for %s (attempt %d/%d)", url, attempt, youTubeMaxRetries)
			} else if err != nil {
				logger.Debugf(e, "YouTube summary path: ytInitialData decoding failed for %s (attempt %d/%d): %s", url, attempt, youTubeMaxRetries, err)
			} else {
				logger.Debugf(e, "YouTube summary path: ytInitialData parsed for %s (attempt %d/%d)", url, attempt, youTubeMaxRetries)
				break
			}

			if fallback, fallbackErr := c.parseYouTubeMetadata(body.Data); fallbackErr == nil && fallback != nil {
				logger.Debugf(e, "YouTube summary path: page metadata fallback succeeded for %s (attempt %d/%d)", url, attempt, youTubeMaxRetries)
				return fallback, nil, nil
			} else if fallbackErr != nil {
				logger.Debugf(e, "YouTube summary path: page metadata fallback failed for %s (attempt %d/%d): %s", url, attempt, youTubeMaxRetries, fallbackErr)
			}
		}

		if attempt < youTubeMaxRetries {
			logger.Debugf(e, "YouTube summary path: retrying page parsing for %s after attempt %d/%d", url, attempt, youTubeMaxRetries)
			time.Sleep(youTubeRetryDelay)
		} else {
			logger.Debugf(e, "YouTube summary path: all page parsing attempts failed for %s", url)
			return nil, nil, fmt.Errorf("unable to retrieve YouTube summary for %s after %d attempts", url, youTubeMaxRetries)
		}
	}

	if strings.Contains(url, "/post/") {
		s, src, err := c.parseYouTubePost(e, data)
		if s == nil || err != nil {
			logger.Debugf(e, "YouTube summary path: ytInitialData post parser failed for %s: %v", url, err)
		} else {
			logger.Debugf(e, "YouTube summary path: ytInitialData post parser succeeded for %s", url)
			return s, src, nil
		}
	}

	s, src, err := c.parseYouTubeVideo(e, data)
	if err == nil && s != nil && len(s.messages) > 0 {
		logger.Debugf(e, "YouTube summary path: ytInitialData video parser succeeded for %s", url)
		return s, src, nil
	}
	logger.Debugf(e, "YouTube summary path: ytInitialData video parser produced no usable summary for %s: %v", url, err)
	if fallback, fallbackErr := c.parseYouTubeMetadata(pageData); fallbackErr == nil && fallback != nil {
		logger.Debugf(e, "YouTube summary path: page metadata fallback succeeded for %s", url)
		return fallback, src, nil
	} else if fallbackErr != nil {
		logger.Debugf(e, "YouTube summary path: page metadata fallback failed for %s: %s", url, fallbackErr)
	}
	logger.Debugf(e, "YouTube summary path: no approach produced a summary for %s", url)
	return s, src, err
}

func decodeYouTubeInitialData(body []byte, data *ytData) error {
	assignments := ytInitialDataAssignmentRegexp.FindAllIndex(body, -1)
	if len(assignments) == 0 {
		return errYouTubeInitialDataNotFound
	}

	var lastErr error
	for _, assignment := range assignments {
		var candidate ytData
		decoder := json.NewDecoder(bytes.NewReader(body[assignment[1]:]))
		if err := decoder.Decode(&candidate); err != nil {
			lastErr = err
			continue
		}
		*data = candidate
		return nil
	}
	return lastErr
}

func (c *SummaryCommand) parseYouTubeMetadata(body []byte) (*summaryResult, error) {
	if len(body) == 0 {
		return nil, noContentError
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	metadata := summary.ExtractMetadata(doc)
	return c.createSummaryFromTitleAndDescription(metadata.Title, metadata.Description)
}

func (c *SummaryCommand) parseYouTubeVideo(e *irc.Event, data ytData) (*summaryResult, *models.Source, error) {
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
			} else if !strings.HasSuffix(views, "views") && !strings.HasSuffix(views, "view") {
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

	var source *models.Source
	if len(username) > 0 {
		var err error
		source, err = repository.FindSource(strings.TrimPrefix(username, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}
	}

	messages := make([]string, 0)
	message := ""
	if len(title) > 0 {
		message = style.Bold(title)
	} else {
		return createSummaryResult(), source, nil
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

	return createSummaryResult(messages...), source, nil
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

func (c *SummaryCommand) parseYouTubePost(e *irc.Event, data ytData) (*summaryResult, *models.Source, error) {
	tabs := data.Contents.TwoColumnBrowseResultsRenderer.Tabs
	if len(tabs) == 0 {
		return nil, nil, nil
	}

	sectionListRenderers := tabs[0].TabRenderer.Content.SectionListRenderer.Contents
	if len(sectionListRenderers) == 0 {
		return nil, nil, nil
	}

	itemSectionRenderers := sectionListRenderers[0].ItemSectionRenderer.Contents
	if len(itemSectionRenderers) == 0 {
		return nil, nil, nil
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

	var source *models.Source
	if len(username) > 0 {
		var err error
		source, err = repository.FindSource(strings.TrimPrefix(username, "@"))
		if err != nil {
			log.Logger().Errorf(nil, "error finding source, %s", err)
		}
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
		return nil, source, nil
	}

	if len(author) > 0 {
		message = fmt.Sprintf("%s • %s", message, author)
	}

	if len(published) > 0 {
		message = fmt.Sprintf("%s • %s", message, published)
	}

	messages = append(messages, message)
	return createSummaryResult(messages...), source, nil
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
