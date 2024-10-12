package giphy

import (
	"assistant/pkg/config"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const gifSearchURL = "https://api.giphy.com/v1/gifs/search?api_key=%s&q=%s&limit=1"

type GifSearch struct {
	Data []struct {
		Type   string
		ID     string
		URL    string
		Images []struct {
			Original struct {
				URL    string
				Width  string
				Height string
				Size   string
			}
		}
	}
}

func SearchGIFs(cfg *config.Config, query string) (GifSearch, error) {
	resp, err := http.Get(fmt.Sprintf(gifSearchURL, cfg.Giphy.APIKey, url.QueryEscape(query)))
	if err != nil {
		return GifSearch{}, err
	}

	if resp == nil {
		return GifSearch{}, errors.New("no response from giphy")
	}

	if resp.StatusCode != http.StatusOK {
		return GifSearch{}, fmt.Errorf("error getting giphy search, status %s", resp.Status)
	}

	defer resp.Body.Close()

	var giphy GifSearch
	if err := json.NewDecoder(resp.Body).Decode(&giphy); err != nil {
		return GifSearch{}, err
	}

	return giphy, nil
}
