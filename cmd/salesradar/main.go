// Command salesradar runs the Phase 1 pipeline: discovery → Odoo.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"salesradar/internal/appenv"
	"salesradar/internal/crm"
	"salesradar/internal/pipeline"
	"salesradar/internal/review"
)

// jsonReviewLead mirrors ReviewLead JSON shape; pointer fields use "" when nil.
type jsonReviewLead struct {
	Company         string   `json:"company"`
	Industry        string   `json:"industry"`
	Size            string   `json:"size"`
	ICPMatch        string   `json:"icp_match"`
	DuplicateStatus string   `json:"duplicate_status"`
	LeadStatus      string   `json:"lead_status"`
	Confidence      string   `json:"confidence"`
	Summary         string   `json:"summary"`
	Reasons         []string `json:"reasons"`
	Action          string   `json:"action"`
}

func toSafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func reviewLeadToJSON(r review.ReviewLead) jsonReviewLead {
	return jsonReviewLead{
		Company:         toSafeString(r.Company),
		Industry:        toSafeString(r.Industry),
		Size:            r.Size,
		ICPMatch:        r.ICPMatch,
		DuplicateStatus: r.DuplicateStatus,
		LeadStatus:      r.LeadStatus,
		Confidence:      r.Confidence,
		Summary:         r.Summary,
		Reasons:         r.Reasons,
		Action:          r.Action,
	}
}

func printReviewLeadsJSON(reviews []review.ReviewLead, pretty bool) error {
	out := make([]jsonReviewLead, 0, len(reviews))
	for _, rl := range reviews {
		out = append(out, reviewLeadToJSON(rl))
	}

	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(out, "", "  ")
	} else {
		b, err = json.Marshal(out)
	}
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(b, '\n'))
	return err
}

func printCRMLeadsJSON(reviews []review.ReviewLead, pretty bool) error {
	out := make([]crm.CRMLead, 0, len(reviews))
	for _, rl := range reviews {
		out = append(out, crm.MapToCRMLead(rl))
	}
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(out, "", "  ")
	} else {
		b, err = json.Marshal(out)
	}
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(b, '\n'))
	return err
}

func main() {
	appenv.Load()
	log.Printf("Apollo key exists: %t", os.Getenv("APOLLO_API_KEY") != "")

	format := flag.String("format", "json", "output format: json, csv, or crm (CRM-shaped JSON for integration)")
	pretty := flag.Bool("pretty", true, "pretty-print JSON to stdout (for -format=json or crm; false = compact)")
	flag.Parse()

	if *format != "json" && *format != "csv" && *format != "crm" {
		log.Fatal("invalid -format: use json, csv, or crm")
	}

	ctx := context.Background()
	staged, err := pipeline.Run(ctx, pipeline.DefaultRunParams())
	if err != nil {
		log.Fatal(err)
	}
	reviews := review.BuildReviewLeads(staged)

	switch *format {
	case "json":
		if err := printReviewLeadsJSON(reviews, *pretty); err != nil {
			log.Fatal(err)
		}
	case "csv":
		if err := printReviewLeadsCSV(reviews); err != nil {
			log.Fatal(err)
		}
	case "crm":
		if err := printCRMLeadsJSON(reviews, *pretty); err != nil {
			log.Fatal(err)
		}
	}
}
