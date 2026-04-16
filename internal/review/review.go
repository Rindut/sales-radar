// Package review aggregates factual explanations for human review (no business logic).
package review

import (
	"regexp"
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/enrich"
)

const maxWordsPerLine = 10
const maxSummaryWords = 14

const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Sales-facing pipeline outcome (stored leads only use qualified | partial_data | needs_manual_review).
const (
	SalesQualified         = "qualified"
	SalesPartialData       = "partial_data"
	SalesNeedsManualReview = "needs_manual_review"
	SalesDuplicate         = "duplicate" // not stored; used in CLI/review summaries
)

// Action is the recommended next step for sales (list/export column).
const (
	ActionContact       = "Contact"
	ActionResearchFirst = "Research first"
	ActionIgnore        = "Ignore"
)

// ReviewLead is a compact payload for review UIs and exports (sales-friendly fields).
type ReviewLead struct {
	Company         *string
	Industry        *string
	Size            string
	ICPMatch        string
	DuplicateStatus string
	LeadStatus      string
	Confidence      string
	Summary         string
	Reasons         []string

	SourceRef string

	OfficialDomain    string // validated company website host (primary for outreach)
	WebsiteDomain     string // same as OfficialDomain for legacy/export paths
	SourceTrace       []string
	UsedGoogle        bool
	UsedApollo        bool
	UsedLinkedIn      bool
	LinkedInURL       string
	EmployeeSize      string
	CountryRegion     string
	ReasonForFit      string
	WhyNow            string
	WhyNowStrength    string
	SalesAngle        string
	PriorityScore     int // PRD ICP score 0–100 (same as ICPScore from engine)
	ICPScore          int // explicit engine score for exports/detail
	DataCompleteness  int
	SalesStatus       string
	SalesReady        bool   // true when name + company domain + ICP yes (execution-ready bar)
	Action            string // Contact | Research first | Ignore
	AcceptExplanation string
	MissingOptional   []string
}

const (
	WhyNowHigh   = "high"
	WhyNowMedium = "medium"
	WhyNowLow    = "low"
)

func icpLevelFromMatch(m domain.ICPMatch) string {
	switch m {
	case domain.ICPYes:
		return "high"
	case domain.ICPPartial:
		return "medium"
	default:
		return "low"
	}
}

func inferWhyNow(industry, contextBlob string) (string, string) {
	ind := strings.ToLower(strings.TrimSpace(industry))
	ctx := strings.ToLower(strings.TrimSpace(contextBlob))
	switch {
	case strings.Contains(ind, "bank") || strings.Contains(ind, "finance") || strings.Contains(ctx, "compliance"):
		return "Compliance-heavy environment requires trackable training", WhyNowHigh
	case hasAny(ctx, "expanding", "expansion", "opening new branch", "new branch", "new outlet", "rollout", "scale-up", "hiring at scale"):
		return "Company appears to be expanding operations", WhyNowHigh
	case hasAny(ctx, "hiring", "careers", "vacancies", "recruiting", "talent acquisition", "training manager", "learning and development", "hrbp", "hr business partner"):
		return "Active hiring suggests onboarding and training needs", WhyNowHigh
	case hasAny(ctx, "multi-branch", "multi branch", "multi-location", "multi location", "distributed workforce", "nationwide", "across regions", "across locations", "outlet network"):
		return "Distributed operations likely require standardized training", WhyNowMedium
	default:
		return "No strong urgency signal detected", WhyNowLow
	}
}

func buildSalesAngle(company, industry, employeeSize, whyNow, reason string) string {
	seg := "mid-scale operations"
	if employeeSize != "" && employeeSize != "unknown" && employeeSize != "—" {
		seg = "operations at " + employeeSize + " workforce scale"
	}
	ind := strings.TrimSpace(industry)
	if ind == "" {
		ind = "target sector"
	}
	name := strings.TrimSpace(company)
	if name == "" {
		name = "This account"
	}
	angleLead := name + " operates in " + titleFirst(ind) + " with " + seg + "."
	switch {
	case strings.Contains(strings.ToLower(ind), "retail") || strings.Contains(strings.ToLower(ind), "fmcg") || strings.Contains(strings.ToLower(ind), "grocery"):
		angleLead = name + " runs store/outlet operations where frontline execution consistency is critical now."
	case strings.Contains(strings.ToLower(ind), "bank"):
		angleLead = name + " operates in a compliance-heavy banking environment with recurring certification and audit-readiness needs."
	case strings.Contains(strings.ToLower(ind), "hospitality") || strings.Contains(strings.ToLower(ind), "hotel"):
		angleLead = name + " relies on service-quality consistency and SOP adherence across properties/teams."
	}
	second := "Why now: " + whyNow + "."
	if strings.TrimSpace(reason) != "" {
		second = "Pitch: " + trimToMaxWords(strings.TrimSpace(reason), 14) + "."
	}
	return angleLead + " " + second
}

func titleFirst(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

var reOverEmployees = regexp.MustCompile(`(?i)^over\s+(\d+)\s+employees$`)

// NormalizeSizeDisplay converts internal size text to a short display form where applicable.
func NormalizeSizeDisplay(s *string) string {
	if s == nil {
		return ""
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return ""
	}
	lower := strings.ToLower(t)
	if strings.Contains(lower, "over 1k") || strings.Contains(lower, "over 1000") {
		return "1000+"
	}
	if m := reOverEmployees.FindStringSubmatch(t); len(m) == 2 {
		return m[1] + "+"
	}
	return t
}

// MapConfidence follows sales-facing rules from staged lead state.
func MapConfidence(l domain.StagedOdooLead) string {
	if l.ICPMatch == domain.ICPNo || l.Status == domain.StatusDiscarded {
		return ConfidenceLow
	}
	if l.ICPMatch == domain.ICPPartial || l.DuplicateStatus == domain.DupSuspectedDuplicate {
		return ConfidenceMedium
	}
	if l.ICPMatch == domain.ICPYes && l.DuplicateStatus == domain.DupNew {
		return ConfidenceHigh
	}
	return ConfidenceMedium
}

// MapSalesStatus maps pipeline state to a clear sales label (for rows that passed the storage gate).
func MapSalesStatus(l domain.StagedOdooLead) string {
	if l.DuplicateStatus == domain.DupExact {
		return SalesDuplicate
	}
	if l.DuplicateStatus == domain.DupSuspectedDuplicate {
		return SalesNeedsManualReview
	}
	if l.ICPMatch == domain.ICPPartial {
		return SalesPartialData
	}
	if l.ICPMatch == domain.ICPYes && l.DuplicateStatus == domain.DupNew {
		return SalesQualified
	}
	return SalesPartialData
}

// BuildSummary returns a short business line, deterministic from pipeline state and sales status.
func BuildSummary(l domain.StagedOdooLead, salesStatus string) string {
	var s string
	switch salesStatus {
	case SalesQualified:
		s = "ICP-qualified — target sector, size threshold, website, and LXP/training relevance"
	case SalesPartialData:
		s = "Partial ICP fit — confirm size and territory before outreach"
	case SalesNeedsManualReview:
		s = "Suspected duplicate or overlap — resolve in CRM before outreach"
	case SalesDuplicate:
		s = "Exact duplicate detected — do not create a new opportunity"
	default:
		switch {
		case l.Status == domain.StatusDiscarded && l.ICPMatch == domain.ICPNo:
			s = "Not a fit for current ICP targets"
		case l.Status == domain.StatusDiscarded && l.DuplicateStatus == domain.DupExact:
			s = "Duplicate detected, not recommended"
		case l.ICPMatch == domain.ICPYes && l.DuplicateStatus == domain.DupNew && l.Status == domain.StatusNew:
			s = "Strong ICP fit, ready for outreach"
		default:
			s = "Review this lead before outreach"
		}
	}
	return capWords(s, maxSummaryWords)
}

func capWords(s string, max int) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) <= max {
		return strings.TrimSpace(s)
	}
	return strings.Join(fields[:max], " ")
}

// BuildExplanation aggregates icp_reason, dedup, and status into short lines.
func BuildExplanation(l domain.StagedOdooLead) []string {
	var out []string

	for _, r := range l.ICPReason {
		if len(out) >= domain.MaxICPReasons {
			break
		}
		line := trimToMaxWords(r, maxWordsPerLine)
		if line != "" {
			out = append(out, line)
		}
	}

	switch l.DuplicateStatus {
	case domain.DupExact:
		out = append(out, "exact duplicate detected")
	case domain.DupSuspectedDuplicate:
		out = append(out, "suspected duplicate based on name similarity")
	}

	switch l.Status {
	case domain.StatusNeedsReview:
		out = append(out, "requires manual review")
	case domain.StatusDiscarded:
		switch {
		case l.ICPMatch == domain.ICPNo:
			out = append(out, "discarded due to ICP mismatch")
		case l.ICPMatch == domain.ICPYes && l.DuplicateStatus == domain.DupExact:
			out = append(out, "discarded: exact duplicate blocks push")
		}
	}

	return out
}

// BuildReviewLeads maps each staged lead to a ReviewLead (same order).
func BuildReviewLeads(staged []domain.StagedOdooLead) []ReviewLead {
	out := make([]ReviewLead, 0, len(staged))
	for _, s := range staged {
		out = append(out, BuildReviewLead(s))
	}
	return out
}

func icpContextText(l domain.StagedOdooLead) string {
	var b strings.Builder
	for _, r := range l.ICPReason {
		b.WriteString(r)
		b.WriteByte(' ')
	}
	for _, e := range l.Explanation {
		b.WriteString(e)
		b.WriteByte(' ')
	}
	return b.String()
}

func locationDisplay(loc *string) string {
	if loc == nil {
		return ""
	}
	t := strings.TrimSpace(*loc)
	if t == "" {
		return ""
	}
	if len(t) == 1 {
		return strings.ToUpper(t)
	}
	return strings.ToUpper(t[:1]) + strings.ToLower(t[1:])
}

// BuildReviewLead maps a staged lead into a review payload for sales and CLI.
func BuildReviewLead(l domain.StagedOdooLead) ReviewLead {
	reasons := l.Explanation
	if len(reasons) == 0 {
		reasons = BuildExplanation(l)
	}

	industryStr := ""
	var industryPtr *string
	if l.Industry != nil && strings.TrimSpace(*l.Industry) != "" {
		industryStr = strings.TrimSpace(*l.Industry)
		industryPtr = l.Industry
	} else if l.ICPIndustryBucket != domain.BucketNone {
		industryStr = string(l.ICPIndustryBucket)
		v := industryStr
		industryPtr = &v
	}

	emp := NormalizeSizeDisplay(l.CompanySizeEstimated)
	if emp == "" || emp == "—" {
		emp = "unknown"
	}
	official := companycheck.SanitizeCompanyWebsiteDomain(l.OfficialDomain)
	if official == "" {
		official = enrich.WebsiteDomainFromRef(l.SourceRef)
	}
	official = companycheck.SanitizeCompanyWebsiteDomain(official)
	web := official
	li := enrich.LinkedInCompanyURL(l.SourceRef)
	if li == "" {
		li = enrich.LinkedInCompanyURL(l.EnrichedLinkedInURL)
	}
	if li == "" && strings.TrimSpace(l.EnrichedLinkedInURL) != "" {
		li = strings.TrimSpace(l.EnrichedLinkedInURL)
	}
	country := locationDisplay(l.Location)
	if country == "" {
		country = enrich.CountryRegionFromText(icpContextText(l))
	}

	ctxBlob := strings.TrimSpace(l.UnstructuredContext + " " + icpContextText(l))
	whyNow, whyStrength := inferWhyNow(industryStr, strings.TrimSpace(ctxBlob+" "+strings.TrimSpace(l.SourceRef)))
	reasonFit := enrich.ReasonForFit(l.ICPMatch, industryStr, append([]string(nil), l.ICPReason...), ctxBlob)
	reasonFit = strings.TrimSpace(reasonFit)
	if reasonFit == "" {
		reasonFit = enrich.DefaultReasonForFit(l.ICPMatch, industryStr)
	}

	compStr := ""
	if l.CompanyName != nil {
		compStr = *l.CompanyName
	}

	dc := enrich.DataCompleteness(compStr, emp, official, country, industryStr, li)

	ss := MapSalesStatus(l)
	tr := l.ProspectTrace
	rl := ReviewLead{
		Company:          l.CompanyName,
		Industry:         industryPtr,
		Size:             emp,
		ICPMatch:         icpLevelFromMatch(l.ICPMatch),
		DuplicateStatus:  string(l.DuplicateStatus),
		LeadStatus:       string(l.Status),
		Confidence:       MapConfidence(l),
		Reasons:          reasons,
		SourceRef:        l.SourceRef,
		OfficialDomain:   official,
		WebsiteDomain:    web,
		SourceTrace:      append([]string(nil), tr.SourceTrace...),
		UsedGoogle:       tr.UsedGoogle,
		UsedApollo:       tr.UsedApollo,
		UsedLinkedIn:     tr.UsedLinkedIn,
		LinkedInURL:      li,
		EmployeeSize:     emp,
		CountryRegion:    country,
		ReasonForFit:     reasonFit,
		WhyNow:           whyNow,
		WhyNowStrength:   whyStrength,
		SalesAngle:       buildSalesAngle(compStr, industryStr, emp, whyNow, reasonFit),
		DataCompleteness: dc,
		PriorityScore:    l.ICPScore,
		ICPScore:         l.ICPScore,
		SalesStatus:      ss,
	}
	rl.Summary = BuildSummary(l, ss)
	rl.MissingOptional = computeOptionalGaps(rl)
	rl.SalesReady = ComputeSalesReady(rl, l)
	rl.Action = ComputeAction(rl, l)
	return rl
}

func hasAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// HasEnrichmentSignals is true when any enrichment signal exists (partial counts for MVP stats).
func HasEnrichmentSignals(rl ReviewLead) bool {
	if strings.TrimSpace(rl.OfficialDomain) != "" || strings.TrimSpace(rl.WebsiteDomain) != "" {
		return true
	}
	if rl.EmployeeSize != "" && rl.EmployeeSize != "—" && rl.EmployeeSize != "unknown" {
		return true
	}
	if strings.TrimSpace(rl.ReasonForFit) != "" {
		return true
	}
	return false
}

func isICPRelatedIndustry(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return false
	}
	for _, kw := range []string{
		"bank", "banking",
		"retail", "fmcg", "grocery", "supermarket", "department store", "convenience",
		"hospitality", "hotel",
	} {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

// ComputeSalesReady is true when the lead is actionable for outbound:
// identifiable company + valid official domain + ICP-related industry + reason_for_fit present.
func ComputeSalesReady(rl ReviewLead, l domain.StagedOdooLead) bool {
	if l.ICPMatch == domain.ICPNo {
		return false
	}
	if rl.Company == nil {
		return false
	}
	n := strings.TrimSpace(*rl.Company)
	if n == "" || !companycheck.IsIdentifiableCompany(n) {
		return false
	}
	d := strings.TrimSpace(strings.ToLower(rl.OfficialDomain))
	if d == "" {
		d = strings.TrimSpace(strings.ToLower(rl.WebsiteDomain))
	}
	if d == "" || companycheck.IsBlockedNonCompanyDomain(d) {
		return false
	}
	industry := ""
	if rl.Industry != nil {
		industry = strings.TrimSpace(*rl.Industry)
	}
	if industry == "" && l.ICPIndustryBucket != domain.BucketNone {
		industry = string(l.ICPIndustryBucket)
	}
	if !isICPRelatedIndustry(industry) {
		return false
	}
	if strings.TrimSpace(rl.ReasonForFit) == "" {
		return false
	}
	return true
}

// ComputeAction sets Contact vs Research first vs Ignore from pipeline state and sales-ready bar.
func ComputeAction(rl ReviewLead, l domain.StagedOdooLead) string {
	ss := rl.SalesStatus
	if ss == "" {
		ss = MapSalesStatus(l)
	}
	if ss == SalesDuplicate {
		return ActionIgnore
	}
	if ss == SalesNeedsManualReview || l.DuplicateStatus == domain.DupSuspectedDuplicate {
		return ActionIgnore
	}

	// Research first only when identity/domain/industry/reason data is still unclear.
	if rl.Company == nil || !companycheck.IsIdentifiableCompany(strings.TrimSpace(*rl.Company)) {
		return mergeScoreAction(ActionResearchFirst, l.ScoreAction)
	}
	d := strings.TrimSpace(strings.ToLower(rl.OfficialDomain))
	if d == "" {
		d = strings.TrimSpace(strings.ToLower(rl.WebsiteDomain))
	}
	if d == "" || companycheck.IsBlockedNonCompanyDomain(d) {
		return mergeScoreAction(ActionResearchFirst, l.ScoreAction)
	}
	industry := ""
	if rl.Industry != nil {
		industry = strings.TrimSpace(*rl.Industry)
	}
	if industry == "" && l.ICPIndustryBucket != domain.BucketNone {
		industry = string(l.ICPIndustryBucket)
	}
	if !isICPRelatedIndustry(industry) {
		return mergeScoreAction(ActionResearchFirst, l.ScoreAction)
	}
	if strings.TrimSpace(rl.ReasonForFit) == "" {
		return mergeScoreAction(ActionResearchFirst, l.ScoreAction)
	}
	return mergeScoreAction(ActionContact, l.ScoreAction)
}

func mergeScoreAction(legacy string, sa domain.ScoreAction) string {
	if sa == "" {
		return legacy
	}
	switch sa {
	case domain.ScoreActionReject:
		return ActionIgnore
	case domain.ScoreActionResearch:
		return ActionResearchFirst
	case domain.ScoreActionContact:
		if legacy == ActionResearchFirst {
			return ActionResearchFirst
		}
		return ActionContact
	default:
		return legacy
	}
}

// ApplySalesStatusAndCopy sets sales status, summary, explanation, and optional gaps for persisted rows.
func ApplySalesStatusAndCopy(l domain.StagedOdooLead, rl ReviewLead) ReviewLead {
	ss := MapSalesStatus(l)
	rl.SalesStatus = ss
	rl.Summary = BuildSummary(l, ss)
	rl.MissingOptional = computeOptionalGaps(rl)
	rl.SalesReady = ComputeSalesReady(rl, l)
	rl.PriorityScore = l.ICPScore
	rl.ICPScore = l.ICPScore
	rl.Action = ComputeAction(rl, l)
	rl.AcceptExplanation = buildAcceptExplanation(l, rl)
	return rl
}

func computeOptionalGaps(rl ReviewLead) []string {
	var m []string
	if rl.CountryRegion == "" {
		m = append(m, "country/region")
	}
	if rl.LinkedInURL == "" && strings.Contains(strings.ToLower(rl.SourceRef), "linkedin.com") {
		m = append(m, "LinkedIn company URL (parse)")
	}
	if rl.OfficialDomain == "" && rl.WebsiteDomain == "" {
		m = append(m, "website/domain")
	}
	if rl.EmployeeSize == "unknown" || rl.EmployeeSize == "" {
		m = append(m, "employee_size (confirmed)")
	}
	return m
}

func buildAcceptExplanation(l domain.StagedOdooLead, rl ReviewLead) string {
	switch rl.SalesStatus {
	case SalesQualified:
		return "Stored as qualified: company and industry present, ICP match strong (sector + size threshold + LXP signal), duplicate status new. Check sales_ready for execution bar (specific name + company-owned domain)."
	case SalesPartialData:
		if len(rl.MissingOptional) > 0 {
			return "Downgraded to partial_data: ICP match is partial. Some enrichment fields are still missing (" + strings.Join(rl.MissingOptional, ", ") + ")."
		}
		return "Downgraded to partial_data: ICP match is partial — validate industry and size before heavy outreach."
	case SalesNeedsManualReview:
		return "Flagged needs_manual_review: suspected duplicate or overlapping account — confirm in CRM before outreach."
	default:
		return "Lead passed minimum storage rules; see reasons and ICP fields for detail."
	}
}

func trimToMaxWords(s string, maxWords int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	if len(fields) <= maxWords {
		return s
	}
	return strings.Join(fields[:maxWords], " ") + "…"
}
