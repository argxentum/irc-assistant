package drudge

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/retriever"
	"assistant/pkg/log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const drudgeReportURL = "https://drudgereport.com/"
const processMax = 50

func GetHeadlineURLs(e *irc.Event, n int) ([]string, error) {
	logger := log.Logger()

	r := retriever.NewDocumentRetriever(retriever.NewBodyRetriever())
	doc, err := r.RetrieveDocument(e, retriever.DefaultParams(drudgeReportURL))
	if err != nil {
		logger.Debugf(e, "failed to retrieve drudge headlines: %v", err)
		return nil, err
	}

	if doc == nil {
		return nil, nil
	}

	urls := make([]string, 0)
	processed := 0

	doc.Root.Find("a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if processed >= processMax {
			logger.Debugf(e, "processed maximum number of urls (%d)", processed)
			return false
		}

		processed++

		href := s.AttrOr("href", "")
		if href == "" {
			return true
		}

		logger.Debugf(e, "found drudge headline URL: %s", href)

		src, err := repository.FindSource(href)
		if err != nil {
			logger.Debugf(e, "error trying to retrieve source details for drudge headline URL: %v", err)
			return true
		}

		if src == nil {
			logger.Debugf(e, "source details for drudge headline URL not found, skipping: %s", href)
			return true
		}

		if !strings.Contains(strings.ToLower(src.Credibility), "high") {
			logger.Debugf(e, "source credibility %s too low, skipping: %s", src.Credibility, href)
			return true
		}

		urls = append(urls, href)

		if len(urls) == n {
			logger.Debugf(e, "found all %d drudge headline URL(s)", n)
			return false
		}

		return true
	})

	return urls, nil
}
