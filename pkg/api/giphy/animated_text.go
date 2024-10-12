package giphy

import (
	"assistant/pkg/config"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const giphyAPIAnimatedTextURL = "https://api.giphy.com/v1/text/animate?api_key=%s&m=%s"

type AnimatedText struct {
	Data []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		URL    string `json:"url"`
		Title  string `json:"title"`
		Style  string `json:"animated_text_style"`
		Images map[string]struct {
			URL    string `json:"url"`
			Height string `json:"height"`
			Width  string `json:"width"`
			Size   string `json:"size"`
			Webp   string `json:"webp"`
		}
	}
	Pagination struct {
		TotalCount int `json:"total_count"`
		Count      int `json:"count"`
		Offset     int `json:"offset"`
	}
	Meta struct {
		Status     int    `json:"status"`
		Msg        string `json:"msg"`
		ResponseID string `json:"response_id"`
	}
}

func GetAnimatedText(cfg *config.Config, message string) (AnimatedText, error) {
	resp, err := http.Get(fmt.Sprintf(giphyAPIAnimatedTextURL, cfg.Giphy.APIKey, url.QueryEscape(message)))
	if err != nil {
		return AnimatedText{}, err
	}

	if resp == nil {
		return AnimatedText{}, errors.New("no response from giphy")
	}

	if resp.StatusCode != http.StatusOK {
		return AnimatedText{}, fmt.Errorf("error getting giphy text, status %s", resp.Status)
	}

	defer resp.Body.Close()

	var giphy AnimatedText
	if err := json.NewDecoder(resp.Body).Decode(&giphy); err != nil {
		return AnimatedText{}, err
	}

	return giphy, nil
}
