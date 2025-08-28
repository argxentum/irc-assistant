package wikipedia

import (
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type SearchResult struct {
	Query struct {
		Search []struct {
			PageID int `json:"pageid"`
			Title  string
		}
	}
}

type Page struct {
	Title  string
	Titles struct {
		Canonical  string
		Normalized string
	}
	Description string
	ContentURLs struct {
		Desktop struct {
			Page string
		}
	} `json:"content_urls"`
	Extract string
}

func get(url, userAgent string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %d", res.StatusCode)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

const wikipediaSearchAPIURL = "https://en.wikipedia.org/w/api.php?action=query&format=json&list=search&srlimit=1&srsearch=%s"

func Search(query, userAgent string) (*Page, error) {
	logger := log.Logger()

	query = url.QueryEscape(query)
	u := fmt.Sprintf(wikipediaSearchAPIURL, query)

	logger.Debugf(nil, "searching wikipedia for query: %s (%s)", query, u)

	d, err := get(u, userAgent)
	if err != nil {
		return nil, err
	}

	var results SearchResult
	err = json.Unmarshal(d, &results)
	if err != nil {
		return nil, err
	}

	if len(results.Query.Search) == 0 {
		return nil, nil
	}

	return GetPage(results.Query.Search[0].Title, userAgent)
}

const wikipediaPageAPIURL = "https://en.wikipedia.org/api/rest_v1/page/summary/%s"

func GetPage(query, userAgent string) (*Page, error) {
	logger := log.Logger()

	query = strings.Replace(query, " ", "_", -1)
	query = url.QueryEscape(query)
	u := fmt.Sprintf(wikipediaPageAPIURL, query)

	logger.Debugf(nil, "getting wikipedia page for query: %s (%s)", query, u)

	d, err := get(u, userAgent)
	if err != nil {
		return nil, err
	}

	var page Page
	err = json.Unmarshal(d, &page)
	if err != nil {
		return nil, err
	}

	return &page, nil
}

var wikipediaURLRegex = regexp.MustCompile(`^https?://(?:[a-z0-9-]+\.)*wikipedia\.org/wiki/([^/?]+)(?:\?.*?)?$`)

func GetPageForURL(u, userAgent string) (*Page, error) {
	if !wikipediaURLRegex.MatchString(u) {
		return nil, fmt.Errorf("invalid wikipedia URL")
	}

	m := wikipediaURLRegex.FindStringSubmatch(u)
	if len(m) != 2 {
		return nil, fmt.Errorf("invalid wikipedia URL")
	}

	title, err := url.QueryUnescape(m[1])
	if err != nil {
		return nil, err
	}

	return GetPage(title, userAgent)
}
