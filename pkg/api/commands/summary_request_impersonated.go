package commands

import "assistant/pkg/api/irc"

func (c *summaryCommand) impersonatedRequest(e *irc.Event, url string) (*summary, error) {
	return c.request(e, url, true)
}
