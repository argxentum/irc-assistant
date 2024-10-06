package reddit

import (
	"assistant/pkg/api/retriever"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LoginResult struct {
	Modhash string
	JWT     string
	URL     *url.URL
	Cookies []*http.Cookie
}

type Listing struct {
	Data struct {
		Children []struct {
			Data Post
		}
	}
}

type Post struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Created     float64 `json:"created_utc"`
	Subreddit   string  `json:"subreddit"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
}

func IsJWTExpired(tok string) bool {
	if len(tok) == 0 {
		return true
	}

	token, _ := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		return token, nil
	})

	if token == nil {
		return true
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil || exp == nil {
		return true
	}

	return time.Now().Unix() > exp.Unix()
}

func Login(username, password string) (*LoginResult, error) {
	data := url.Values{}
	data.Set("user", username)
	data.Set("passwd", password)
	data.Set("api_type", "json")

	req, err := http.NewRequest(http.MethodPost, "https://ssl.reddit.com/api/login", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range retriever.RandomHeaderSet() {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var body struct {
		JSON struct {
			Errors []string
			Data   struct {
				Modhash string
				Cookie  string
			}
		}
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	result := &LoginResult{}

	result.Modhash = body.JSON.Data.Modhash
	result.JWT = body.JSON.Data.Cookie
	u, err := url.Parse("https://reddit.com")
	if err != nil {
		return nil, err
	}

	result.URL = u
	result.Cookies = resp.Cookies()
	return result, nil
}
