package retriever

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type BodyRetriever interface {
	RetrieveBody(e *irc.Event, params RetrievalParams) ([]byte, error)
}

func NewBodyRetriever() BodyRetriever {
	return &bodyRetriever{
		//
	}
}

type bodyRetriever struct {
	//
}

func (r *bodyRetriever) RetrieveBody(e *irc.Event, params RetrievalParams) ([]byte, error) {
	logger := log.Logger()

	translated := translateURL(params.URL)
	if translated != params.URL {
		logger.Debugf(e, "translated %s to %s", params.URL, translated)
	}
	params.URL = translated

	req, err := http.NewRequest(params.Method, params.URL, params.Body)
	if err != nil {
		logger.Debugf(e, "request creation error, %s", err)
		return nil, err
	}

	if params.Impersonate {
		headers := RandomHeaderSet()
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		msg := ""
		keys := make([]string, 0)
		for k := range req.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			for _, v := range req.Header[k] {
				if len(msg) > 0 {
					msg += ", "
				}
				msg += fmt.Sprintf("%v: %v", k, v)
			}
		}

		logger.Debugf(e, "added impersonation request headers: [%v]", msg)
	}

	success := false

	var rc = make(chan retrieved)
	go func() {
		go func() {
			time.Sleep(params.Timeout * time.Millisecond)
			if !success {
				logger.Debugf(e, "timing out request")
			}
			rc <- retrieved{nil, RequestTimedOutError}
		}()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if resp != nil {
				logger.Debugf(e, "retrieval error (status %d), %s", resp.StatusCode, err)
			} else {
				logger.Debugf(e, "retrieval error, %s", err)
			}
			rc <- retrieved{nil, err}
		}
		if resp == nil {
			logger.Debugf(e, "retrieval error")
			rc <- retrieved{nil, NoResponseError}
		}
		rc <- retrieved{resp, nil}
		success = true
	}()

	ret := <-rc

	if ret.err != nil {
		logger.Debugf(e, "retrieval error: %s", ret.err)
		return nil, ret.err
	}

	if ret.response == nil {
		logger.Debugf(e, "no response")
		return nil, NoResponseError
	}

	defer ret.response.Body.Close()

	logger.Debugf(e, "[%d] (%s) %s", ret.response.StatusCode, ret.response.Header.Get("Content-Type"), req.URL.String())

	if ret.response.StatusCode == http.StatusOK && !IsContentTypeAllowed(ret.response.Header.Get("Content-Type")) {
		logger.Debugf(e, "disallowed content type %s", ret.response.Header.Get("Content-Type"))
		return nil, DisallowedContentTypeError
	}

	return io.ReadAll(ret.response.Body)
}
