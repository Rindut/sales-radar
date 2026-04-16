package icp

import (
	"strings"

	"salesradar/internal/domain"
)

// InferLeadIndustryID maps a lead to a catalog industry ID (banking, retail, hospitality, or extended sectors).
// Returns "" when no confident match (leave targeting / exclusions to other rules).
func InferLeadIndustryID(e *domain.ExtractedLead, bucket domain.ICPIndustryBucket) string {
	switch bucket {
	case domain.BucketBanking:
		return "banking"
	case domain.BucketRetail:
		return "retail"
	case domain.BucketHospitality:
		return "hospitality"
	}
	if e != nil && e.Industry != nil {
		if id := matchIndustryCatalogString(*e.Industry); id != "" {
			return id
		}
	}
	if e != nil {
		if id := matchIndustryCatalogString(combinedContext(e)); id != "" {
			return id
		}
	}
	return ""
}

func matchIndustryCatalogString(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	if s == "" {
		return ""
	}
	// More specific patterns first (overlap with hospitality vs healthcare).
	if strings.Contains(s, "fmcg") || strings.Contains(s, "consumer packaged") || containsCPG(s) {
		return "fmcg"
	}
	if strings.Contains(s, "logistic") || strings.Contains(s, "freight") || strings.Contains(s, "supply chain") {
		return "logistics"
	}
	if strings.Contains(s, "software") || strings.Contains(s, "saas") || strings.Contains(s, "technology") || strings.Contains(s, " it ") {
		return "technology"
	}
	if strings.Contains(s, "education") || strings.Contains(s, "university") || strings.Contains(s, "e-learning") || strings.Contains(s, "school") {
		return "education"
	}
	if strings.Contains(s, "manufactur") || strings.Contains(s, "factory") {
		return "manufacturing"
	}
	if strings.Contains(s, "health") || strings.Contains(s, "pharma") || strings.Contains(s, "medical") || strings.Contains(s, "biotech") {
		return "healthcare"
	}
	if strings.Contains(s, "bank") || strings.Contains(s, "banking") {
		return "banking"
	}
	if strings.Contains(s, "retail") || strings.Contains(s, "grocery") || strings.Contains(s, "supermarket") {
		return "retail"
	}
	if strings.Contains(s, "hospitality") || strings.Contains(s, "hotel") || strings.Contains(s, "resort") {
		return "hospitality"
	}
	if strings.Contains(s, "other") && (strings.Contains(s, "industry") || strings.Contains(s, "sector")) {
		return "others"
	}
	return ""
}

func containsCPG(s string) bool {
	return strings.Contains(s, "cpg") || (strings.Contains(s, "consumer") && strings.Contains(s, "goods"))
}
