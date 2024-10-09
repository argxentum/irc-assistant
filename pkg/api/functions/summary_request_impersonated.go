package functions

import "assistant/pkg/api/irc"

func (f *summaryFunction) impersonatedRequest(e *irc.Event, url string) (*summary, error) {
	return f.request(e, url, true)
}
