package extraction

import (
	"context"
	"strings"
	"testing"

	"salesradar/internal/domain"
)

func TestExtract_normalCase(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID: "d1",
		Source:      domain.SourceApollo,
		SourceRef:   "apollo:org:1",
		UnstructuredContext: strings.Join([]string{
			"@company: PT Bank Sejahtera",
			"@industry: BANKING",
			"@size: 1100-1400",
			"@location: Indonesia",
		}, "\n"),
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.CompanyName == nil || *out.CompanyName != "PT Bank Sejahtera" {
		t.Fatalf("company_name = %v", out.CompanyName)
	}
	if out.Industry == nil || *out.Industry != "banking" {
		t.Fatalf("industry = %v", out.Industry)
	}
	if out.CompanySizeEstimated == nil || *out.CompanySizeEstimated != "1100-1400" {
		t.Fatalf("size = %v", out.CompanySizeEstimated)
	}
	if out.Location == nil || *out.Location != "Indonesia" {
		t.Fatalf("location = %v", out.Location)
	}
	if out.Source != raw.Source || out.SourceRef != raw.SourceRef {
		t.Fatal("source/source_ref must pass through")
	}
	if out.ExtractionNotes != nil {
		t.Fatalf("unexpected notes: %v", *out.ExtractionNotes)
	}
}

func TestExtract_missingFields(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID:         "d2",
		Source:              domain.SourceLinkedIn,
		SourceRef:           "https://linkedin.example/company/1",
		UnstructuredContext: "@company: Solo Entity",
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.CompanyName == nil || *out.CompanyName != "Solo Entity" {
		t.Fatalf("company_name = %v", out.CompanyName)
	}
	if out.Industry != nil || out.CompanySizeEstimated != nil || out.Location != nil {
		t.Fatalf("expected nil fields, got industry=%v size=%v location=%v", out.Industry, out.CompanySizeEstimated, out.Location)
	}
}

func TestExtract_conflictingSignals(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID: "d3",
		Source:      domain.SourceGoogle,
		SourceRef:   "google:q=bank",
		UnstructuredContext: strings.Join([]string{
			"@industry: banking",
			"this profile mentions retail heavily with outlet expansion",
		}, "\n"),
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Industry == nil || *out.Industry != "banking" {
		t.Fatalf("tagged industry must win, got %v", out.Industry)
	}
	if out.ExtractionNotes == nil || !strings.Contains(strings.ToLower(*out.ExtractionNotes), "conflict between tagged and heuristic industry") {
		t.Fatalf("expected industry conflict note, got %v", out.ExtractionNotes)
	}
}

func TestExtract_multipleCompanyNamesNormalizeAndNote(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID: "d4",
		Source:      domain.SourceJobPortal,
		SourceRef:   "job:123",
		UnstructuredContext: strings.Join([]string{
			"@company: PT Alpha Makmur",
			"@company: PT Beta Logistics",
		}, "\n"),
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.CompanyName == nil || *out.CompanyName != "PT Alpha Makmur" {
		t.Fatalf("normalized company_name = %v", out.CompanyName)
	}
	if out.ExtractionNotes == nil || !strings.Contains(strings.ToLower(*out.ExtractionNotes), "multiple company names") {
		t.Fatalf("expected multiple-company note, got %v", out.ExtractionNotes)
	}
}

func TestExtract_ambiguityCase(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID:         "d5",
		Source:              domain.SourceGoogle,
		SourceRef:           "google:q=ambiguous",
		UnstructuredContext: "retail banking platform",
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Industry != nil {
		t.Fatalf("industry should be nil on ambiguity, got %v", out.Industry)
	}
	if out.ExtractionNotes == nil || !strings.Contains(strings.ToLower(*out.ExtractionNotes), "ambiguous industry") {
		t.Fatalf("expected ambiguous industry note, got %v", out.ExtractionNotes)
	}
}

func TestExtract_noSignalCase(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID:         "d6",
		Source:              domain.SourceGoogle,
		SourceRef:           "google:q=none",
		UnstructuredContext: "we help companies grow",
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.CompanyName != nil || out.Industry != nil || out.CompanySizeEstimated != nil || out.Location != nil {
		t.Fatalf("expected all nil parsed fields, got %+v", out)
	}
	if out.ExtractionNotes != nil {
		t.Fatalf("expected no notes, got %v", *out.ExtractionNotes)
	}
}

func TestExtract_companyFallback(t *testing.T) {
	raw := domain.RawCandidate{
		DiscoveryID: "d7",
		Source:      domain.SourceApollo,
		SourceRef:   "apollo:org:7",
		UnstructuredContext: "Regional banking context; PT Mandiri Sejahtera operates nationwide; estimated 1,100–1,400 employees; Indonesia.",
	}
	out, err := Extract(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.CompanyName == nil || *out.CompanyName != "PT Mandiri Sejahtera" {
		t.Fatalf("expected PT fallback company, got %v", out.CompanyName)
	}
	if out.ExtractionNotes == nil || !strings.Contains(strings.ToLower(*out.ExtractionNotes), "company name from text pattern") {
		t.Fatalf("expected fallback note, got %v", out.ExtractionNotes)
	}
}

func TestExtract_contextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Extract(ctx, domain.RawCandidate{UnstructuredContext: "@company: X"})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
