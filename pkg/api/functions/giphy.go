package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const giphyAPIAnimatedTextURL = "https://api.giphy.com/v1/text/animate?api_key=%s&m=%s"

type giphyAnimatedTextResponse struct {
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

func (f *FunctionStub) giphyAnimatedTextRequest(e *irc.Event, message string) (giphyAnimatedTextResponse, error) {
	logger := log.Logger()

	resp, err := http.Get(fmt.Sprintf(giphyAPIAnimatedTextURL, f.cfg.Giphy.APIKey, url.QueryEscape(message)))
	if err != nil {
		logger.Errorf(e, "error getting giphy text, %s", err)
		return giphyAnimatedTextResponse{}, err
	}

	if resp == nil {
		logger.Errorf(e, "no response from giphy")
		return giphyAnimatedTextResponse{}, errors.New("no response from giphy")
	}

	if resp.StatusCode != http.StatusOK {
		logger.Errorf(e, "error getting giphy text, %s", resp.Status)
		return giphyAnimatedTextResponse{}, fmt.Errorf("error getting giphy text, status %s", resp.Status)
	}

	defer resp.Body.Close()

	var giphy giphyAnimatedTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&giphy); err != nil {
		logger.Errorf(e, "error decoding giphy text, %s", err)
		return giphyAnimatedTextResponse{}, err
	}

	return giphy, nil
}
