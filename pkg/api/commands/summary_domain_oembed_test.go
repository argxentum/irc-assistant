package commands

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCreateOEmbedSummaryAllowsShortTitle(t *testing.T) {
	result, err := createOEmbedSummary(oEmbedData{Title: "CNN on TikTok", AuthorName: "CNN"})
	if err != nil {
		t.Fatalf("create oEmbed summary: %v", err)
	}
	if result == nil || len(result.messages) != 1 {
		t.Fatalf("oEmbed result = %#v", result)
	}
	if message := result.messages[0]; !strings.Contains(message, "CNN on TikTok") || !strings.Contains(message, "CNN") {
		t.Fatalf("oEmbed message = %q", message)
	}
}

func TestOEmbedSummaryRequestsAndDecodesMetadata(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.URL.Query().Get("url"); got != "https://example.com/video/1" {
			t.Errorf("oEmbed URL parameter = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"title":"A useful video title","author_name":"Example Author"}`)),
		}, nil
	})}

	command := &SummaryCommand{}
	result, author, err := command.oEmbedSummaryWithClient(nil, client, "https://oembed.example?url=%s", "https://example.com/video/1")
	if err != nil {
		t.Fatalf("retrieve oEmbed summary: %v", err)
	}
	if author != "Example Author" {
		t.Fatalf("oEmbed author = %q", author)
	}
	if result == nil || len(result.messages) != 1 || !strings.Contains(result.messages[0], "A useful video title") {
		t.Fatalf("oEmbed result = %#v", result)
	}
}
