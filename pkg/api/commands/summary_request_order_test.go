package commands

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSummaryRequestChainKeepsSlugSearchLast(t *testing.T) {
	chain := (&SummaryCommand{}).requestChain()
	if len(chain) == 0 {
		t.Fatal("request chain is empty")
	}

	lastName := runtime.FuncForPC(reflect.ValueOf(chain[len(chain)-1]).Pointer()).Name()
	if !strings.Contains(lastName, "slugSearchRequest") {
		t.Fatalf("last request chain step = %s, want slugSearchRequest", lastName)
	}

	exactSearches := []string{
		"braveSearchRequest",
		"startPageRequest",
		"duckduckgoRequest",
		"bingRequest",
	}
	for _, expected := range exactSearches {
		found := false
		for _, request := range chain[:len(chain)-1] {
			name := runtime.FuncForPC(reflect.ValueOf(request).Pointer()).Name()
			if strings.Contains(name, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s does not precede slug fallback", expected)
		}
	}
}
