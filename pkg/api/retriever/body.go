package retriever

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type BodyRetriever interface {
	RetrieveBody(e *irc.Event, params RetrievalParams) (*Body, error)
}

type Body struct {
	Data     []byte
	Response *http.Response
}

func NewBodyRetriever() BodyRetriever {
	return &bodyRetriever{
		//
	}
}

type bodyRetriever struct {
	//
}

func (r *bodyRetriever) RetrieveBody(e *irc.Event, params RetrievalParams) (*Body, error) {
	logger := log.Logger()

	req, err := http.NewRequest(params.Method, params.URL, params.Body)
	if err != nil {
		logger.Debugf(e, "request creation error, %s", err)
		return nil, err
	}

	headers := params.Headers
	if len(headers) == 0 && params.Impersonate {
		headers = RandomHeaderSet()
	}

	if len(headers) > 0 {
		for k, v := range headers {
			if len(v) == 0 {
				continue
			}
			req.Header.Add(k, v)
		}

		keys := make([]string, 0)
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		msg := ""
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

		// check if response is encoded and decode
		if resp.Header.Get("Content-Encoding") == "gzip" {
			logger.Debugf(e, "response is gzipped, decompressing")
			gzippedBody, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Debugf(e, "error reading gzipped body: %s", err)
				rc <- retrieved{nil, err}
				success = false
				return
			}

			// Create a new reader to decompress the gzipped body
			resp.Body.Close() // Close the original body before replacing it
			gzippedReader, err := gzip.NewReader(bytes.NewReader(gzippedBody))
			if err != nil {
				logger.Debugf(e, "error creating gzip reader: %s", err)
				rc <- retrieved{nil, err}
				success = false
				return
			}

			defer gzippedReader.Close()

			// Read the decompressed body
			decompressedBody, err := io.ReadAll(gzippedReader)
			if err != nil {
				logger.Debugf(e, "error reading decompressed body: %s", err)
				rc <- retrieved{nil, err}
				success = false
				return
			}

			// Replace the response body with the decompressed body
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(decompressedBody))
			resp.Header.Del("Content-Encoding")
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(gzippedBody)))
			success = true
		} else {
			rc <- retrieved{resp, nil}
			success = true
		}
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

	body, err := io.ReadAll(ret.response.Body)
	return &Body{
		Data:     body,
		Response: ret.response,
	}, err
}
