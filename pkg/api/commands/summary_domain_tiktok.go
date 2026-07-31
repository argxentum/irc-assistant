package commands

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const tikTokItemDetailURL = "https://www.tiktok.com/api/customtdk/item/?itemId=%s&odinId=%s"
const tikTokRequestTimeout = 5 * time.Second

var tikTokVideoURLRegex = regexp.MustCompile(`^https://(?:www\.)?tiktok.com/(.*?)/video/(\d+)`)
var jsonDataRegex = regexp.MustCompile(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">(.*?)</script>`)

func (c *SummaryCommand) parseTikTok(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
	logger := log.Logger()
	logger.Debugf(e, "TikTok summary path: trying oEmbed for %s", url)
	if result, metadata, err := c.oEmbedSummary(e, tikTokOEmbedURL, url); err == nil && result != nil {
		logger.Debugf(e, "TikTok summary path: oEmbed succeeded for %s", url)
		src, sourceErr := findOEmbedSource(e, metadata)
		if sourceErr != nil {
			logger.Errorf(e, "TikTok oEmbed source check failed for %s: %v", url, sourceErr)
		}
		return result, src, nil
	} else if err != nil {
		logger.Debugf(e, "TikTok summary path: oEmbed failed for %s: %s; falling back to page parsing", url, err)
	} else {
		logger.Debugf(e, "TikTok summary path: oEmbed returned no usable summary for %s; falling back to page parsing", url)
	}

	logger.Debugf(e, "TikTok summary path: trying page parsing for %s", url)
	return c.parseTikTokPage(e, url)
}

func (c *SummaryCommand) parseTikTokPage(e *irc.Event, url string) (*summaryResult, *models.Source, error) {
	logger := log.Logger()

	if !tikTokVideoURLRegex.MatchString(url) {
		return nil, nil, fmt.Errorf("tiktok url does not match expected pattern: %s", url)
	}

	m := tikTokVideoURLRegex.FindStringSubmatch(url)
	if len(m) < 3 {
		return nil, nil, fmt.Errorf("tiktok url pattern unexpected matches (%d) for: %s", len(m), url)
	}

	author := strings.TrimPrefix(m[1], "@")
	itemID := m[2]

	logger.Debugf(e, "tiktok author %s, itemID %s for: %s", author, itemID, url)

	client := &http.Client{Timeout: tikTokRequestTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error for %s, %v", url, err)
	}
	if resp == nil {
		return nil, nil, fmt.Errorf("tiktok nil response for %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("tiktok invalid status code %d for %s", resp.StatusCode, url)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error reading body content for %s, %v", url, err)
	}
	html := string(b)

	m = jsonDataRegex.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil, nil, fmt.Errorf("tiktok json data not found for %s", url)
	}
	videoJson := m[1]

	var videoData tikTokVideoData
	err = json.Unmarshal([]byte(videoJson), &videoData)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error unmarshaling video data for %s, %v", url, err)
	}
	logger.Debugf(e, "TikTok summary path: embedded page data parsed for %s", url)

	detailURL := fmt.Sprintf(tikTokItemDetailURL, itemID, videoData.DefaultScope.AppContext.OdinID)
	var itemData tikTokItemData
	logger.Debugf(e, "TikTok summary path: trying optional item-detail enrichment for %s", url)
	if detailResp, detailErr := client.Get(detailURL); detailErr != nil {
		logger.Debugf(e, "TikTok summary path: optional item-detail enrichment failed for %s: %v", url, detailErr)
	} else if detailResp != nil {
		defer detailResp.Body.Close()
		if detailResp.StatusCode != http.StatusOK {
			logger.Debugf(e, "TikTok summary path: optional item-detail enrichment returned status %d for %s", detailResp.StatusCode, url)
		} else if detailBody, readErr := io.ReadAll(detailResp.Body); readErr != nil {
			logger.Debugf(e, "TikTok summary path: optional item-detail enrichment could not be read for %s: %v", url, readErr)
		} else if unmarshalErr := json.Unmarshal(detailBody, &itemData); unmarshalErr != nil {
			logger.Debugf(e, "TikTok summary path: optional item-detail enrichment could not be decoded for %s: %v", url, unmarshalErr)
		} else {
			logger.Debugf(e, "TikTok summary path: optional item-detail enrichment succeeded for %s", url)
		}
	}

	src, err := repository.FindSource(author)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error finding source for author %s: %v", author, err)
	}

	result, err := c.createTikTokSummary(videoData, itemData)
	if err != nil {
		logger.Debugf(e, "TikTok summary path: page parser failed for %s: %v", url, err)
	} else if result != nil {
		logger.Debugf(e, "TikTok summary path: page parser succeeded for %s", url)
	}
	return result, src, err
}

func (c *SummaryCommand) createTikTokSummary(videoData tikTokVideoData, itemData tikTokItemData) (*summaryResult, error) {
	title := itemData.Item.Title
	if title == "" {
		title = videoData.DefaultScope.VideoDetail.ShareMeta.Title
	}

	description := itemData.Item.Description
	if description == "" {
		description = videoData.DefaultScope.VideoDetail.ItemInfo.Item.Description
	}
	if description == "" {
		description = videoData.DefaultScope.VideoDetail.ShareMeta.Description
	}
	if description == "" {
		description = videoData.DefaultScope.VideoDetail.ShareMeta.LegacyDescription
	}
	if title == "" {
		title = description
	}
	if description == title {
		description = ""
	}

	if len(title) > maximumTitleLength {
		title = title[:maximumTitleLength] + "..."
	}

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if c.isRejectedTitle(title) {
		return nil, rejectedTitleError
	}

	message := ""
	if title != "" {
		message = style.Bold(title)
	}
	if description != "" {
		if message == "" {
			message = style.Bold(description)
		} else {
			message += getSummaryFieldSeparator(title) + " " + description
		}
	}
	if message == "" {
		return nil, noContentError
	}

	if videoData.DefaultScope.VideoDetail.ItemInfo.Item.Author.Name != "" {
		message += " • " + videoData.DefaultScope.VideoDetail.ItemInfo.Item.Author.Name
	}

	if videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Views > 0 {
		plural := ""
		if videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Views != 1 {
			plural = "s"
		}
		message += fmt.Sprintf(" • %s view%s", text.ShortenNumber(videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Views), plural)
	}

	if videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Likes > 0 {
		plural := ""
		if videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Likes != 1 {
			plural = "s"
		}
		message += fmt.Sprintf(" • %s like%s", text.ShortenNumber(videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Likes), plural)
	}

	if videoData.DefaultScope.VideoDetail.ItemInfo.Item.CreatedAt != "" {
		epoch, err := strconv.ParseInt(videoData.DefaultScope.VideoDetail.ItemInfo.Item.CreatedAt, 10, 64)
		if err == nil {
			t := elapse.PastTimeDescription(time.Unix(epoch, 0))
			message += " • " + t
		}
	}

	return &summaryResult{messages: []string{message}}, nil
}

type tikTokVideoData struct {
	DefaultScope struct {
		AppContext struct {
			OdinID string `json:"odinId"`
		} `json:"webapp.app-context"`
		VideoDetail struct {
			ItemInfo struct {
				Item struct {
					CreatedAt   string `json:"createTime"`
					Description string `json:"desc"`
					Author      struct {
						Name string `json:"nickname"`
					} `json:"author"`
					Stats struct {
						Likes    int `json:"diggCount"`
						Comments int `json:"commentCount"`
						Shares   int `json:"shareCount"`
						Views    int `json:"playCount"`
					} `json:"stats"`
				} `json:"itemStruct"`
			} `json:"itemInfo"`
			ShareMeta struct {
				Title             string `json:"title"`
				Description       string `json:"desc"`
				LegacyDescription string `json:"description"`
			} `json:"shareMeta"`
		} `json:"webapp.video-detail"`
	} `json:"__DEFAULT_SCOPE__"`
}

type tikTokItemData struct {
	Item struct {
		Description string `json:"desc"`
		Title       string `json:"title"`
	} `json:"itemCustomTDK"`
}
