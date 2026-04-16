package enrichment

import (
	"context"
	"strings"
	"testing"

	"salesradar/internal/domain"
)

func TestApply_FillsSummaryAndLocation(t *testing.T) {
	txt := "@company: TestCo\n@industry: retail\nWe operate nationwide in Indonesia with store rollout.\nMore prose here."
	e := &domain.ExtractedLead{UnstructuredContext: txt}
	Apply(context.Background(), e)
	if e.AISummaryShort == nil || !strings.Contains(*e.AISummaryShort, "Indonesia") {
		t.Fatalf("AISummaryShort = %v", e.AISummaryShort)
	}
	if e.Location == nil || *e.Location != "Indonesia" {
		t.Fatalf("Location = %v", e.Location)
	}
}

func TestApply_DoesNotReplaceExisting(t *testing.T) {
	summary := "already set"
	loc := "Singapore"
	e := &domain.ExtractedLead{
		UnstructuredContext: "noise in Indonesia",
		AISummaryShort:      &summary,
		Location:            &loc,
	}
	Apply(context.Background(), e)
	if e.AISummaryShort == nil || *e.AISummaryShort != "already set" {
		t.Fatal("should keep AISummaryShort")
	}
	if e.Location == nil || *e.Location != "Singapore" {
		t.Fatal("should keep Location")
	}
}
