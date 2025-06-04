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
		Root: doc,
		Body: body,
	}, err
}

func (r *docRetriever) RetrieveDocumentSelection(e *irc.Event, params RetrievalParams, selector string) (*goquery.Selection, error) {
	logger := log.Logger()

	var selection docResult
	res := make(chan docResult)

	attempts := 0

	for {
		if attempts+1 > params.Retries {
			logger.Debugf(e, "stopping after %d attempts", attempts)
			selection = docResult{nil, errors.New("reached max attempts")}
			break
		}

		attempts++

		if attempts > 1 {
			delay := (retryDelayOffset * time.Millisecond) + (time.Duration(rand.IntN(params.RetryDelay)) * time.Millisecond)
			logger.Debugf(e, "waiting %s before attempt %d", delay, attempts)
			time.Sleep(delay)
		}

		logger.Debugf(e, "%s %s, attempt %d", params.Method, params.URL, attempts)

		go func() {
			doc, err := r.RetrieveDocument(e, params)
			if err != nil {
				if errors.Is(err, DisallowedContentTypeError) {
					logger.Debugf(e, "exiting due to disallowed content type")
					res <- docResult{nil, err}
				} else {
					logger.Debugf(e, "retrieval error: %s", err)
				}
				return
			}

			node := doc.Root.Find(selector)
			if node.Nodes == nil {
				logger.Debugf(e, "no nodes found for selector %s", selector)
				return
			}

			res <- docResult{node.First(), nil}
		}()

		go func() {
			time.Sleep(params.Timeout * time.Millisecond)

			if selection.doc != nil {
				return
			}

			logger.Debugf(e, "retrieval attempt %d timed out", attempts)
			res <- docResult{nil, RequestTimedOutError}
		}()

		selection = <-res

		if errors.Is(selection.err, RequestTimedOutError) {
			continue
		}

		break
	}

	return selection.doc, selection.err
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
