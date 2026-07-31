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

func TestCreateOEmbedSummarySupportsRedditMetadata(t *testing.T) {
	result, err := createOEmbedSummary(oEmbedData{
		Title:        "How to get API access",
		AuthorName:   "example-user",
		ProviderName: "reddit",
	})
	if err != nil {
		t.Fatalf("create Reddit oEmbed summary: %v", err)
	}
	if result == nil || len(result.messages) != 1 {
		t.Fatalf("Reddit oEmbed result = %#v", result)
	}
	message := result.messages[0]
	for _, expected := range []string{"How to get API access", "example-user"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("Reddit oEmbed message %q does not contain %q", message, expected)
		}
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
			Body:       io.NopCloser(strings.NewReader(`{"title":"A useful video title","author_name":"Example Author","author_url":"https://www.youtube.com/@example","author_unique_id":"example","provider_name":"YouTube"}`)),
		}, nil
	})}

	command := &SummaryCommand{}
	result, metadata, err := command.oEmbedSummaryWithClient(nil, client, "https://oembed.example?url=%s", "https://example.com/video/1")
	if err != nil {
		t.Fatalf("retrieve oEmbed summary: %v", err)
	}
	if metadata == nil || metadata.AuthorName != "Example Author" || metadata.AuthorURL != "https://www.youtube.com/@example" || metadata.AuthorUniqueID != "example" || metadata.ProviderName != "YouTube" {
		t.Fatalf("oEmbed metadata = %#v", metadata)
	}
	if result == nil || len(result.messages) != 1 || !strings.Contains(result.messages[0], "A useful video title") {
		t.Fatalf("oEmbed result = %#v", result)
	}
}

func TestOEmbedSourceIdentitiesPreferProfileAndHandle(t *testing.T) {
	tests := []struct {
		name string
		data oEmbedData
		want []string
	}{
		{
			name: "youtube",
			data: oEmbedData{
				AuthorName: "Secular Talk",
				AuthorURL:  "https://www.youtube.com/@SecularTalk",
			},
			want: []string{"youtube.com/@SecularTalk", "SecularTalk", "Secular Talk"},
		},
		{
			name: "tiktok deduplicates handle and display name",
			data: oEmbedData{
				AuthorName:     "CNN",
				AuthorURL:      "https://www.tiktok.com/@cnn?refer=embed",
				AuthorUniqueID: "cnn",
			},
			want: []string{"tiktok.com/@cnn", "cnn"},
		},
		{
			name: "instagram",
			data: oEmbedData{
				AuthorName: "nasaearth",
				AuthorURL:  "https://www.instagram.com/nasaearth",
			},
			want: []string{"instagram.com/nasaearth", "nasaearth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oEmbedSourceIdentities(tt.data)
			if len(got) != len(tt.want) {
				t.Fatalf("identities = %#v, want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("identities = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}
