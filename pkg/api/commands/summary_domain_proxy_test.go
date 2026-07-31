package commands

import (
	"assistant/pkg/config"
	"testing"
)

func TestRedditTriesLocalSummaryBeforeConfiguredProxy(t *testing.T) {
	command := &SummaryCommand{commandStub: &commandStub{cfg: &config.Config{
		Summary: config.SummaryConfig{ProxiedDomains: []string{"reddit.com", "example.com"}},
	}}}

	if command.shouldProxyDomainBeforeLocalSummary("https://www.reddit.com/r/example/comments/123/post") {
		t.Fatal("Reddit should try its local oEmbed handler before the configured proxy")
	}
	if !command.shouldProxyDomainBeforeLocalSummary("https://www.example.com/post") {
		t.Fatal("other configured proxy domains should remain proxy-first")
	}
}
