package giphy

import (
	"assistant/pkg/config"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const animatedTextURL = "https://api.giphy.com/v1/text/animate?api_key=%s&m=%s"

type AnimatedText struct {
	Data []struct {
		ID     string
		Type   string
		URL    string
		Title  string
		Style  string `json:"animated_text_style"`
		Images map[string]struct {
			URL    string
			Height string
			Width  string
			Size   string
		}
	}
	Pagination struct {
		TotalCount int `json:"total_count"`
		Count      int
		Offset     int
	}
	Meta struct {
		Status     int
		Msg        string
		ResponseID string `json:"response_id"`
	}
}

func CreateAnimatedText(cfg *config.Config, message string) (AnimatedText, error) {
	resp, err := http.Get(fmt.Sprintf(animatedTextURL, cfg.Giphy.APIKey, url.QueryEscape(message)))
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
