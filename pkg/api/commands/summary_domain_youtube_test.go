package commands

import (
	"assistant/pkg/config"
	"errors"
	"strings"
	"testing"
)

func TestDecodeYouTubeInitialDataAcceptsAssignmentVariants(t *testing.T) {
	variants := []string{
		`<script>var ytInitialData = {"engagementPanels":[{}]};window.after = true;</script>`,
		`<script>ytInitialData={"engagementPanels":[{}]};</script>`,
		`<script>window["ytInitialData"] = {"engagementPanels":[{}]};</script>`,
		`<script>window['ytInitialData']={"engagementPanels":[{}]};</script>`,
		`<script>window.ytInitialData = {"engagementPanels":[{}]};</script>`,
	}

	for _, body := range variants {
		var data ytData
		if err := decodeYouTubeInitialData([]byte(body), &data); err != nil {
			t.Fatalf("decode assignment %q: %v", body, err)
		}
		if len(data.EngagementPanels) != 1 {
			t.Fatalf("engagement panels = %d, want 1", len(data.EngagementPanels))
		}
	}
}

func TestDecodeYouTubeInitialDataReportsMissingAssignment(t *testing.T) {
	var data ytData
	err := decodeYouTubeInitialData([]byte(`<html><title>No data</title></html>`), &data)
	if !errors.Is(err, errYouTubeInitialDataNotFound) {
		t.Fatalf("error = %v, want %v", err, errYouTubeInitialDataNotFound)
	}
}

func TestParseYouTubeMetadataFallback(t *testing.T) {
	command := &SummaryCommand{commandStub: &commandStub{cfg: &config.Config{}}}
	body := []byte(`<html><head>
		<meta property="og:title" content="Example video title">
		<meta property="og:description" content="Example video description">
	</head></html>`)

	result, err := command.parseYouTubeMetadata(body)
	if err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	if result == nil || len(result.messages) != 1 {
		t.Fatalf("metadata result = %#v", result)
	}
	if !strings.Contains(result.messages[0], "Example video title") || !strings.Contains(result.messages[0], "Example video description") {
		t.Fatalf("metadata message = %q", result.messages[0])
	}
}
