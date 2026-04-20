// Package extraction is a conservative data-preparation layer.
// It parses tags, extracts weak signals, and merges outputs without classification.
package extraction

import (
	"context"
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/enrich"
)

// IndustrySignal is a weak heuristic signal from unstructured text.
// Confidence is currently always "low" for heuristic matches.
type IndustrySignal struct {
	Value      *string
	Confidence string
	Ambiguous  bool
}

type taggedFields struct {
	companies []string
	industry  *string
	size      *string
	location  *string

	hasAnyTag bool
}

type heuristicSignals struct {
	industry IndustrySignal
	size     *string
	location *string
}

// Extract maps RawCandidate.UnstructuredContext into ExtractedLead.
// It performs deterministic parsing only, with no external calls.
func Extract(ctx context.Context, c domain.RawCandidate) (*domain.ExtractedLead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tags, body := extractTaggedFields(c.UnstructuredContext)
	signals := extractHeuristicSignals(body)

	out := mergeSignals(c, tags, signals)
	return out, nil
}

func mergeSignals(c domain.RawCandidate, tags taggedFields, signals heuristicSignals) *domain.ExtractedLead {
	companyName, multiNameNote := normalizeCompanyNames(tags.companies)
	companyFromFallback := false
	headlineFallback := false
	if companyName == nil {
		if fb := extractCompanyNameFallback(c.UnstructuredContext); fb != nil {
			companyName = fb
			companyFromFallback = true
		} else if h := extractHeadlineCompany(c.UnstructuredContext); h != nil {
			companyName = h
			companyFromFallback = true
			headlineFallback = true
		}
	}

	industry := tags.industry
	size := tags.size
	location := tags.location

	var notes []string
	if multiNameNote != "" {
		notes = append(notes, multiNameNote)
	}
	if companyFromFallback {
		if headlineFallback {
			notes = append(notes, "Display name from first line of description (no legal-entity pattern).")
		} else {
			notes = append(notes, "Weak evidence: company name from text pattern.")
		}
	}

	if tags.industry != nil && signals.industry.Value != nil && !equalFoldTrim(*tags.industry, *signals.industry.Value) {
		notes = append(notes, "Conflict between tagged and heuristic industry signals.")
	}
	if tags.size != nil && signals.size != nil && !equalFoldTrim(*tags.size, *signals.size) {
		notes = append(notes, "Conflict between tagged and heuristic size signals.")
	}
	if signals.industry.Ambiguous {
		notes = append(notes, "Ambiguous industry signal from unstructured text.")
	}

	// Fill only missing fields from weak heuristic signals.
	if industry == nil && !signals.industry.Ambiguous && signals.industry.Value != nil {
		industry = signals.industry.Value
	}
	if size == nil && signals.size != nil {
		size = signals.size
	}
	if location == nil && signals.location != nil {
		location = signals.location
	}

	if !tags.hasAnyTag {
		if signals.industry.Value != nil || signals.size != nil || signals.location != nil || signals.industry.Ambiguous {
			notes = append(notes, "Weak evidence: inferred from unstructured text only.")
		}
	}

	var extractionNotes *string
	if len(notes) > 0 {
		n := strings.Join(notes, " ")
		extractionNotes = &n
	}

	official := companycheck.SanitizeCompanyWebsiteDomain(c.OfficialDomain)
	if official == "" {
		official = enrich.WebsiteDomainFromRef(c.SourceRef)
	}
	official = companycheck.SanitizeCompanyWebsiteDomain(official)

	return &domain.ExtractedLead{
		DiscoveryID:          c.DiscoveryID,
		Source:               c.Source,
		SourceRef:            c.SourceRef,
		CompanyName:          companyName,
		Industry:             industry,
		StrongClassification: false,
		ICPIndustryBucket:    domain.BucketNone,
		CompanySizeEstimated: size,
		Location:             location,
		AISummaryShort:       nil,
		ExtractionNotes:      extractionNotes,
		UnstructuredContext:  c.UnstructuredContext,
		OfficialDomain:       official,
		ProspectTrace:        c.ProspectTrace,
		EnrichedLinkedInURL:  strings.TrimSpace(c.EnrichedLinkedInURL),
		WebsiteEnrichment:    c.WebsiteEnrichment,
	}
}
