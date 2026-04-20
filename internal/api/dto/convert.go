package dto

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"salesradar/internal/domain"
	"salesradar/internal/store"
)

// LeadFromStore maps a persisted lead to the API DTO.
func LeadFromStore(l store.Lead) Lead {
	return Lead{
		ID:                l.ID,
		Company:           l.Company,
		Industry:          l.Industry,
		Size:              l.Size,
		ICPMatch:          l.ICPMatch,
		DuplicateStatus:   l.DuplicateStatus,
		LeadStatus:        l.LeadStatus,
		Confidence:        l.Confidence,
		Summary:           l.Summary,
		Reasons:           append([]string(nil), l.Reasons...),
		Source:            l.Source,
		CreatedAt:         l.CreatedAt.UTC().Format(time.RFC3339),
		WebsiteDomain:     l.WebsiteDomain,
		LinkedInURL:       l.LinkedInURL,
		CountryRegion:     l.CountryRegion,
		ReasonForFit:      l.ReasonForFit,
		WhyNow:            l.WhyNow,
		WhyNowStrength:    l.WhyNowStrength,
		SalesAngle:        l.SalesAngle,
		PriorityScore:     l.PriorityScore,
		DataCompleteness:  l.DataCompleteness,
		SalesStatus:       l.SalesStatus,
		EmployeeSize:      l.EmployeeSize,
		AcceptExplanation: l.AcceptExplanation,
		MissingOptional:   append([]string(nil), l.MissingOptional...),
		SourceRef:         l.SourceRef,
		SalesReady:        l.SalesReady,
		Action:            l.Action,
		OfficialDomain:    l.OfficialDomain,
		SourceTrace:       append([]string(nil), l.SourceTrace...),
		UsedGoogle:        l.UsedGoogle,
		UsedApollo:        l.UsedApollo,
		UsedLinkedIn:      l.UsedLinkedIn,
		WebsiteEnrichmentSelectedURLs: l.WebsiteEnrichmentSelectedURLs,
		WebsiteEnrichmentSummary:      l.WebsiteEnrichmentSummary,
		WebsiteEnrichmentSignals:      l.WebsiteEnrichmentSignals,
		WebsiteEnrichmentStatus:       l.WebsiteEnrichmentStatus,
		WebsiteEnrichedAt:             l.WebsiteEnrichedAt,
	}
}

// DiscoveryFromDomain maps domain toggles to API DTO.
func DiscoveryFromDomain(t domain.DiscoverySourceToggles) DiscoverySourcesToggles {
	return DiscoverySourcesToggles{
		Google:       t.Google,
		Seed:         t.Seed,
		WebsiteCrawl: t.WebsiteCrawl,
		JobSignal:    t.JobSignal,
		Apollo:       t.Apollo,
		LinkedIn:     t.LinkedIn,
	}
}

// DiscoveryToDomain maps API DTO to domain toggles.
func DiscoveryToDomain(d DiscoverySourcesToggles) domain.DiscoverySourceToggles {
	return domain.DiscoverySourceToggles{
		Google:       d.Google,
		Seed:         d.Seed,
		WebsiteCrawl: d.WebsiteCrawl,
		JobSignal:    d.JobSignal,
		Apollo:       d.Apollo,
		LinkedIn:     d.LinkedIn,
	}
}

// ICPFromStore maps persisted ICP form to API DTO.
func ICPFromStore(s store.ICPFormSettings) ICPForm {
	return ICPForm{
		Version:            s.Version,
		TargetIndustries:   append([]string(nil), s.TargetIndustries...),
		RegionFocus:        s.RegionFocus,
		SignalKeys:         append([]string(nil), s.SignalKeys...),
		ExcludedIndustries: append([]string(nil), s.ExcludedIndustries...),
		ExcludedSegments:   append([]string(nil), s.ExcludedSegments...),
		ApplySub50:         cloneBoolPtr(s.ApplySub50),
		WeightIndustry:     s.WeightIndustry,
		WeightSignal:       s.WeightSignal,
		WeightSize:         s.WeightSize,
		MinEmployees:       s.MinEmployees,
		MaxEmployees:       s.MaxEmployees,
		TargetIndustry:     s.TargetIndustry,
		CompanySize:        s.CompanySize,
		CountryRegion:      s.CountryRegion,
		RequiredSignal:     s.RequiredSignal,
		ExcludedIndustry:   s.ExcludedIndustry,
	}
}

// ICPToStore maps API PUT body to store settings (full replace).
func ICPToStore(i ICPForm) store.ICPFormSettings {
	return store.ICPFormSettings{
		Version:            i.Version,
		TargetIndustries:   append([]string(nil), i.TargetIndustries...),
		RegionFocus:        strings.TrimSpace(i.RegionFocus),
		SignalKeys:         append([]string(nil), i.SignalKeys...),
		ExcludedIndustries: append([]string(nil), i.ExcludedIndustries...),
		ExcludedSegments:   append([]string(nil), i.ExcludedSegments...),
		ApplySub50:         cloneBoolPtr(i.ApplySub50),
		WeightIndustry:     strings.TrimSpace(i.WeightIndustry),
		WeightSignal:       strings.TrimSpace(i.WeightSignal),
		WeightSize:         strings.TrimSpace(i.WeightSize),
		MinEmployees:       strings.TrimSpace(i.MinEmployees),
		MaxEmployees:       strings.TrimSpace(i.MaxEmployees),
		TargetIndustry:     i.TargetIndustry,
		CompanySize:        i.CompanySize,
		CountryRegion:      i.CountryRegion,
		RequiredSignal:     i.RequiredSignal,
		ExcludedIndustry:   i.ExcludedIndustry,
	}
}

func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// OptionalPipelineSummaryFromQuery parses legacy redirect-style query params into a summary struct.
func OptionalPipelineSummaryFromQuery(q url.Values) *PipelineSummaryNumbers {
	get := func(k string) string {
		return strings.TrimSpace(q.Get(k))
	}
	if get("candidates") == "" && get("stored") == "" {
		return nil
	}
	return &PipelineSummaryNumbers{
		CandidatesFound:   atoi(get("candidates")),
		Enriched:          atoi(get("enriched")),
		ContactReady:      atoi(get("contact_ready")),
		ResearchFirst:     atoi(get("research_first")),
		Rejected:          atoi(get("rejected")),
		DuplicatesRemoved: atoi(get("dupes")),
		SemanticMerged:    atoi(get("merged")),
		RowsStored:        atoi(get("stored")),
	}
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
