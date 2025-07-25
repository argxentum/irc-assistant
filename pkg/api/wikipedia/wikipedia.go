package wikipedia

import (
	"assistant/pkg/log"
	"fmt"
	gowiki "github.com/Arilucea/go-wiki"
	"github.com/Arilucea/go-wiki/page"
	"net/url"
	"regexp"
)

func Search(query string) (*page.WikipediaPage, error) {
	results, suggestion, err := gowiki.Search(query, 3, false)
	if err != nil {
		return nil, err
	}

	if len(suggestion) > 0 {
		return GetPage(suggestion)
	}

	for _, r := range results {
		p, err := GetPage(r)
		if err != nil {
			continue
		}

		if len(p.URL) == 0 {
			continue
		}

		return p, nil
	}

	return nil, nil
}

func GetPage(query string) (*page.WikipediaPage, error) {
	// try to get the page with suggested titles first; if that fails, try again without suggested titles
	p, err := gowiki.GetPage(query, -1, true, true)
	if err != nil {
		log.Logger().Debugf(nil, "error getting page for query %s with suggested titles: %v", query, err)
		p, err = gowiki.GetPage(query, -1, false, true)
		if err != nil {
			log.Logger().Debugf(nil, "error getting page for query %s without suggested titles: %v", query, err)
			return nil, err
		}
	}

	_, err = p.GetSummary()
	if err != nil {
		return nil, err
	}

	_, err = p.GetContent()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

var wikipediaURLRegex = regexp.MustCompile(`^https?://(?:[a-z0-9-]+\.)*wikipedia\.org/wiki/([^/?]+)(?:\?.*?)?$`)

func GetPageForURL(u string) (*page.WikipediaPage, error) {
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

	return GetPage(title)
}
