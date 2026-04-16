package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"salesradar/internal/review"
)

func TestCSVHeaderRow(t *testing.T) {
	var buf bytes.Buffer
	if err := writeReviewLeadsCSV(&buf, nil); err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(strings.NewReader(buf.String()))
	row, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"company", "industry", "size", "icp_match", "duplicate_status", "lead_status", "confidence", "action", "summary", "reasons"}
	if len(row) != len(want) {
		t.Fatalf("header len: got %d want %d", len(row), len(want))
	}
	for i := range want {
		if row[i] != want[i] {
			t.Fatalf("header[%d]: got %q want %q", i, row[i], want[i])
		}
	}
}

func TestCSVEmptyFields(t *testing.T) {
	leads := []review.ReviewLead{
		{
			Company:         nil,
			Industry:        nil,
			Size:            "",
			ICPMatch:        "no",
			DuplicateStatus: "new",
			LeadStatus:      "discarded",
			Confidence:      "low",
			Summary:         "Not a fit",
			Reasons:         nil,
		},
	}
	var buf bytes.Buffer
	if err := writeReviewLeadsCSV(&buf, leads); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want header+1 row, got %d", len(rows))
	}
	data := rows[1]
	if data[0] != "" || data[1] != "" || data[2] != "" {
		t.Fatalf("expected empty company/industry/size, got %v", data[:3])
	}
	if data[9] != "" {
		t.Fatalf("empty reasons should be empty string, got %q", data[9])
	}
}

func TestCSVReasonsJoined(t *testing.T) {
	leads := []review.ReviewLead{
		{
			ICPMatch:        "yes",
			DuplicateStatus: "new",
			LeadStatus:      "new",
			Confidence:      "high",
			Summary:         "Strong fit",
			Reasons:         []string{"reason a", "reason b"},
		},
	}
	var buf bytes.Buffer
	if err := writeReviewLeadsCSV(&buf, leads); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	got := rows[1][9]
	want := "reason a | reason b"
	if got != want {
		t.Fatalf("reasons: got %q want %q", got, want)
	}
}

func TestCSVMultipleRows(t *testing.T) {
	leads := []review.ReviewLead{
		{Company: strPtr("A Co"), ICPMatch: "yes", DuplicateStatus: "new", LeadStatus: "new", Confidence: "high", Summary: "S1"},
		{Company: strPtr("B Co"), ICPMatch: "partial", DuplicateStatus: "new", LeadStatus: "needs_review", Confidence: "medium", Summary: "S2"},
	}
	var buf bytes.Buffer
	if err := writeReviewLeadsCSV(&buf, leads); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows (header+2), got %d", len(rows))
	}
	if rows[1][0] != "A Co" || rows[2][0] != "B Co" {
		t.Fatalf("rows: %+v", rows)
	}
}

func strPtr(s string) *string { return &s }
