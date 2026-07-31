package commands

import (
	"assistant/pkg/config"
	"strings"
	"testing"
)

func TestCreateTikTokSummaryAllowsShortTitleAndUsesPageDescription(t *testing.T) {
	command := &SummaryCommand{commandStub: &commandStub{cfg: &config.Config{}}}
	var videoData tikTokVideoData
	videoData.DefaultScope.VideoDetail.ShareMeta.Title = "CNN on TikTok"
	videoData.DefaultScope.VideoDetail.ItemInfo.Item.Description = "The useful description already present in TikTok's page data."
	videoData.DefaultScope.VideoDetail.ItemInfo.Item.Author.Name = "CNN"
	videoData.DefaultScope.VideoDetail.ItemInfo.Item.Stats.Views = 429500

	result, err := command.createTikTokSummary(videoData, tikTokItemData{})
	if err != nil {
		t.Fatalf("create TikTok summary: %v", err)
	}
	if result == nil || len(result.messages) != 1 {
		t.Fatalf("TikTok result = %#v", result)
	}
	message := result.messages[0]
	for _, expected := range []string{"CNN on TikTok", "useful description", "CNN", "429.5K views"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("TikTok message %q does not contain %q", message, expected)
		}
	}
}
