package functions

import (
	"assistant/pkg/api/irc"
	"assistant/pkg/log"
	"errors"
)

var rejectedTitleError = errors.New("rejected title")
var summaryTooShortError = errors.New("summary too short")
var noContentError = errors.New("no summary content")

var rsf []func(e *irc.Event, url string) (*summary, error)

func (f *summaryFunction) requestChain() []func(e *irc.Event, url string) (*summary, error) {
	if rsf == nil {
		rsf = []func(e *irc.Event, url string) (*summary, error){
			f.directRequest,
			f.impersonatedRequest,
			f.nuggetizeRequest,
			f.bingRequest,
			f.duckduckgoRequest,
		}
	}

	return rsf
}

func (f *summaryFunction) requestSummary(e *irc.Event, url string) (*summary, error) {
	logger := log.Logger()

	for _, fn := range f.requestChain() {
		s, err := fn(e, url)
		if err != nil {
			continue
		}
		if s == nil {
			continue
		}

		return s, nil
	}

	logger.Debugf(e, "no request summary found")
	return nil, nil
}
