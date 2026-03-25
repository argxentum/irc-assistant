package retriever

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"math/rand/v2"
	"time"
)

type DocumentRetriever interface {
	RetrieveDocument(e *irc.Event, params RetrievalParams) (*Document, error)
	RetrieveDocumentSelection(e *irc.Event, params RetrievalParams, selector string) (*goquery.Selection, error)
	Parse(e *irc.Event, doc *goquery.Document, selectors ...string) []*goquery.Selection
}

type Document struct {
	URL  string
	Root *goquery.Document
	Body *Body
}

func NewDocumentRetriever(bodyRetriever BodyRetriever) DocumentRetriever {
	return &docRetriever{
		bodyRetriever: bodyRetriever,
	}
}

type docRetriever struct {
	bodyRetriever BodyRetriever
}

type docResult struct {
	doc *goquery.Selection
	err error
}

func (r *docRetriever) RetrieveDocument(e *irc.Event, params RetrievalParams) (*Document, error) {
	body, err := r.bodyRetriever.RetrieveBody(e, params)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, errors.New("empty body")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body.Data))
	return &Document{
		URL:  params.URL,
		Root: doc,
		Body: body,
	}, err
}

func (r *docRetriever) RetrieveDocumentSelection(e *irc.Event, params RetrievalParams, selector string) (*goquery.Selection, error) {
	logger := log.Logger()

	for attempt := 1; attempt <= params.Retries; attempt++ {
		if attempt > 1 {
			delay := (retryDelayOffset * time.Millisecond) + (time.Duration(rand.IntN(params.RetryDelay)) * time.Millisecond)
			logger.Debugf(e, "waiting %s before attempt %d", delay, attempt)
			time.Sleep(delay)
		}

		logger.Debugf(e, "%s %s, attempt %d", params.Method, params.URL, attempt)

		doc, err := r.RetrieveDocument(e, params)
		if err != nil {
			if errors.Is(err, DisallowedContentTypeError) {
				logger.Debugf(e, "exiting due to disallowed content type")
				return nil, err
			}
			if errors.Is(err, RequestTimedOutError) {
				logger.Debugf(e, "retrieval attempt %d timed out", attempt)
				continue
			}
			logger.Debugf(e, "retrieval error: %s", err)
			continue
		}

		node := doc.Root.Find(selector)
		if node.Nodes == nil {
			logger.Debugf(e, "no nodes found for selector %s", selector)
			continue
		}

		return node.First(), nil
	}

	logger.Debugf(e, "stopping after %d attempts", params.Retries)
	return nil, errors.New("reached max attempts")
}

func (r *docRetriever) Parse(e *irc.Event, doc *goquery.Document, selectors ...string) []*goquery.Selection {
	logger := log.Logger()
	logger.Debugf(e, "parsing document")

	results := make([]*goquery.Selection, 0)

	for _, selector := range selectors {
		nodes := doc.Find(selector)
		if nodes == nil {
			logger.Debugf(e, "no nodes found for selector %s", selector)
			continue
		}

		node := nodes.First()
		if node == nil {
			logger.Debugf(e, "no node found for selector %s", selector)
			continue
		}

		results = append(results, node)
	}

	return results
}
