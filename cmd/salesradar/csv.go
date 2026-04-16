package main

import (
	"encoding/csv"
	"io"
	"os"
	"strings"

	"salesradar/internal/review"
)

// printReviewLeadsCSV writes ReviewLead rows as CSV to stdout (header + spreadsheet-safe quoting).
func printReviewLeadsCSV(leads []review.ReviewLead) error {
	return writeReviewLeadsCSV(os.Stdout, leads)
}

func writeReviewLeadsCSV(w io.Writer, leads []review.ReviewLead) error {
	cw := csv.NewWriter(w)
	header := []string{
		"company",
		"industry",
		"size",
		"icp_match",
		"duplicate_status",
		"lead_status",
		"confidence",
		"action",
		"summary",
		"reasons",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range leads {
		row := []string{
			toSafeString(r.Company),
			toSafeString(r.Industry),
			r.Size,
			r.ICPMatch,
			r.DuplicateStatus,
			r.LeadStatus,
			r.Confidence,
			r.Action,
			r.Summary,
			strings.Join(r.Reasons, " | "),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
