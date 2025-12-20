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

var tikTokVideoURLRegex = regexp.MustCompile(`^https://(?:www\.)?tiktok.com/(.*?)/video/(\d+)`)
var jsonDataRegex = regexp.MustCompile(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">(.*?)</script>`)

func (c *SummaryCommand) parseTikTok(e *irc.Event, url string) (*summary, *models.Source, error) {
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

	resp, err := http.DefaultClient.Get(url)
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

	detailURL := fmt.Sprintf(tikTokItemDetailURL, itemID, videoData.DefaultScope.AppContext.OdinID)
	detailResp, err := http.DefaultClient.Get(detailURL)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error fetching item detail for %s, %v", detailURL, err)
	}
	if detailResp == nil {
		return nil, nil, fmt.Errorf("tiktok nil response fetching item detail for %s", detailURL)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("tiktok invalid status code %d fetching item detail for %s", resp.StatusCode, detailURL)
	}

	defer detailResp.Body.Close()

	b, err = io.ReadAll(detailResp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error reading item detail body content for %s, %v", detailURL, err)
	}
	itemJson := string(b)

	var itemData tikTokItemData
	err = json.Unmarshal([]byte(itemJson), &itemData)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error unmarshaling item data for %s, %v", detailURL, err)
	}

	src, err := repository.FindSource(author)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok error finding source for author %s: %v", author, err)
	}

	message := ""

	title := itemData.Item.Title
	if title == "" {
		title = videoData.DefaultScope.VideoDetail.ShareMeta.Title
	}

	description := itemData.Item.Description
	if description == "" {
		description = videoData.DefaultScope.VideoDetail.ShareMeta.Description
	}
	if title == "" {
		title = description
	}

	if len(title) > maximumTitleLength {
		title = title[:maximumTitleLength] + "..."
	}

	if len(description) > standardMaximumDescriptionLength {
		description = description[:standardMaximumDescriptionLength] + "..."
	}

	if c.isRejectedTitle(title) {
		return nil, nil, rejectedTitleError
	}

	if len(title)+len(description) < minimumTitleLength {
		return nil, nil, summaryTooShortError
	}

	if title != "" {
		message += style.Bold(title)
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

	return &summary{messages: []string{message}}, src, nil
}

type tikTokVideoData struct {
	DefaultScope struct {
		AppContext struct {
			OdinID string `json:"odinId"`
		} `json:"webapp.app-context"`
		VideoDetail struct {
			ItemInfo struct {
				Item struct {
					CreatedAt string `json:"createTime"`
					Author    struct {
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
				Title       string `json:"title"`
				Description string `json:"description"`
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
