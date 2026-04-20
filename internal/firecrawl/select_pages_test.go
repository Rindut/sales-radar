package firecrawl

import (
	"strings"
	"testing"
)

func TestPickURLs_excludesLowValue(t *testing.T) {
	root := "https://example.com/"
	links := []string{
		"https://example.com/privacy-policy",
		"https://example.com/about",
		"https://example.com/careers",
	}
	out := pickURLs(root, links, 10)
	for _, u := range out {
		if strings.Contains(strings.ToLower(u), "privacy") {
			t.Fatalf("privacy URL should be excluded: %s", u)
		}
	}
}

func TestPickURLs_includesHome(t *testing.T) {
	root := "https://acme.co/"
	out := pickURLs(root, nil, 3)
	if len(out) == 0 || !strings.Contains(out[0], "acme.co") {
		t.Fatalf("expected homepage-like URL, got %v", out)
	}
}
