package crm

import (
	"strings"
	"testing"

	"salesradar/internal/review"
)

func ptr(s string) *string { return &s }

func TestMapToCRMLead(t *testing.T) {
	r := review.ReviewLead{
		Company:         ptr("ACME Corp"),
		Industry:        ptr("banking"),
		Size:            "1000+",
		LeadStatus:      "new",
		Confidence:      "high",
		Summary:         "Strong ICP fit, ready for outreach",
		Reasons:         []string{"reason one", "reason two"},
	}
	got := MapToCRMLead(r)
	if got.Name != "ACME Corp" || got.Industry != "banking" || got.CompanySize != "1000+" {
		t.Fatalf("fields: %+v", got)
	}
	if got.LeadStatus != "new" || got.Confidence != "high" {
		t.Fatal("status/confidence")
	}
	if got.Notes == "" {
		t.Fatal("notes empty")
	}
	if !strings.Contains(got.Notes, "Strong ICP fit") || !strings.Contains(got.Notes, "reason one") {
		t.Fatalf("notes: %q", got.Notes)
	}
}

func TestMapToCRMLead_nilCompany(t *testing.T) {
	r := review.ReviewLead{
		LeadStatus: "discarded",
		Confidence: "low",
		Summary:    "Not a fit",
	}
	got := MapToCRMLead(r)
	if got.Name != "" {
		t.Fatalf("name = %q", got.Name)
	}
}
