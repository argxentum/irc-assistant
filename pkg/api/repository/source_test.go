package repository

import (
	"assistant/pkg/models"
	"testing"
)

func TestFirstSourceMatchingIdentityPreservesCandidatePriority(t *testing.T) {
	displayNameSource := &models.Source{ID: "display", URLs: []string{"example news"}}
	profileSource := &models.Source{ID: "profile", URLs: []string{"YouTube.com/@ExampleNews"}}

	source, identity, err := firstSourceMatchingIdentity(
		[]string{"youtube.com/@examplenews", "examplenews", "example news"},
		[]*models.Source{displayNameSource, profileSource},
	)
	if err != nil {
		t.Fatalf("match source identity: %v", err)
	}
	if source != profileSource {
		t.Fatalf("source = %#v, want profile source", source)
	}
	if identity != "youtube.com/@examplenews" {
		t.Fatalf("identity = %q", identity)
	}
}
