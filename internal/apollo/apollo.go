// Package apollo provides optional organization enrichment and discovery (Apollo.io API).
// Set APOLLO_API_KEY to enable. No network calls when unset.
package apollo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/icp"
)

const (
	organizationsSearchEndpoint = "https://api.apollo.io/v1/organizations/search"
	defaultApolloKeyword        = "bank OR finance OR fintech OR retail OR logistics OR manufacturing OR corporate"
)

// APIKeyFromEnv returns APOLLO_API_KEY.
func APIKeyFromEnv() string {
	key := strings.TrimSpace(os.Getenv("APOLLO_API_KEY"))
	slog.Info("apollo: API key loaded",
		"loaded", key != "",
		"length", len(key),
	)
	return key
}

// Org is a minimal org payload used for enrichment.
type Org struct {
	ID                    string
	Name                  string
	PrimaryDomain         string
	Industry              string
	Location              string
	ShortDescription      string
	LinkedInURL           string
	EstimatedNumEmployees int
	Signals               []string
	PriorityScore         int
}

type organizationRow struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	PrimaryDomain         string `json:"primary_domain"`
	Industry              string `json:"industry"`
	RawAddress            string `json:"raw_address"`
	LocalizedLocation     string `json:"localized_location"`
	ShortDescription      string `json:"short_description"`
	LinkedinURL           string `json:"linkedin_url"`
	EstimatedNumEmployees int    `json:"estimated_num_employees"`
}

type searchResponse struct {
	Organizations []organizationRow `json:"organizations"`
}

type DiscoveryParams struct {
	Limit int
	ICP   *domain.ICPRuntimeSettings
}

// CompanySearchFilters is the simple extension point for Apollo organization discovery.
type CompanySearchFilters struct {
	Industries   []string
	MinEmployees int
	MaxEmployees int
	Location     string
	Keyword      string
	Limit        int
}

type searchPlan struct {
	Name           string
	Query          string
	EmployeeRanges []string
	PerPage        int
}

// BuildApolloPayload returns a broad high-recall Apollo organization payload.
func BuildApolloPayload() map[string]interface{} {
	return map[string]interface{}{
		"page":                              1,
		"per_page":                          20,
		"q_organization_keyword":            defaultApolloKeyword,
		"organization_locations":            []string{"Indonesia"},
		"organization_num_employees_ranges": []string{"50,1000", "100,5000"},
	}
}

// BuildApolloPayloadByICP returns a broad ICP-shaped payload while still prioritizing recall.
func BuildApolloPayloadByICP(industry string) map[string]interface{} {
	keyword := defaultApolloKeyword
	switch strings.ToLower(strings.TrimSpace(industry)) {
	case "banking":
		keyword = "bank OR lending OR financial services"
	case "retail":
		keyword = "retail chain OR supermarket OR convenience store"
	case "hospitality":
		keyword = "hotel OR hospitality OR resort"
	}
	return map[string]interface{}{
		"page":                              1,
		"per_page":                          20,
		"q_organization_keyword":            keyword,
		"organization_locations":            []string{"Indonesia"},
		"organization_num_employees_ranges": []string{"50,1000", "100,5000"},
	}
}

// EnrichByDomain returns org data when the API key is set and a match exists.
func EnrichByDomain(ctx context.Context, apiKey, domain string) (*Org, error) {
	apiKey = strings.TrimSpace(apiKey)
	domain = companycheck.NormalizeHost(strings.TrimSpace(domain))
	if apiKey == "" || domain == "" {
		return nil, nil
	}

	body := map[string]any{
		"api_key":                apiKey,
		"q_organization_domains": domain,
		"page":                   1,
		"per_page":               1,
	}
	sr, err := searchOrganizations(ctx, body)
	if err != nil {
		return nil, err
	}
	if len(sr.Organizations) == 0 {
		return nil, nil
	}
	return orgFromResponse(sr.Organizations[0]), nil
}

// DiscoverOrganizations searches Apollo organizations using conservative ICP-aligned filters.
func DiscoverOrganizations(ctx context.Context, apiKey string, p DiscoveryParams) ([]Org, error) {
	return SearchCompanies(ctx, apiKey, FiltersFromICP(p.ICP, p.Limit))
}

// SearchCompanies performs a simple Apollo organization search for raw company discovery.
func SearchCompanies(ctx context.Context, apiKey string, filters CompanySearchFilters) ([]Org, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, nil
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 10
	}
	plans := buildSearchPlans(filters)
	if len(plans) == 0 {
		plans = []searchPlan{{Name: "default", Query: "", EmployeeRanges: expandedEmployeeRanges(), PerPage: 25}}
	}

	merged := map[string]Org{}
	rawPerQuery := make([]string, 0, len(plans))
	slog.Info("apollo: company search plans",
		"queries_executed", len(plans),
		"location", strings.TrimSpace(filters.Location),
		"industries", strings.Join(filters.Industries, ","),
		"keyword", strings.TrimSpace(filters.Keyword),
	)
	for _, plan := range plans {
		body := buildCompanySearchPayload(apiKey, filters, plan)
		slog.Info("apollo: company search attempt",
			"attempt", "filtered",
			"industries", strings.Join(filters.Industries, ","),
			"location", strings.TrimSpace(filters.Location),
			"employee_ranges", strings.Join(plan.EmployeeRanges, ","),
			"keyword", strings.TrimSpace(filters.Keyword),
			"query_name", plan.Name,
			"query", plan.Query,
			"per_page", plan.PerPage,
		)
		sr, err := searchOrganizationsWithFallback(ctx, body)
		if err != nil {
			return nil, err
		}
		if len(sr.Organizations) == 0 {
			relaxedBody := buildRelaxedCompanySearchPayload(apiKey, filters, plan)
			slog.Info("apollo: company search attempt",
				"attempt", "fallback",
				"industries", strings.Join(filters.Industries, ","),
				"location", "",
				"employee_ranges", "",
				"keyword", strings.TrimSpace(filters.Keyword),
				"query_name", plan.Name,
				"query", relaxedBody["q_keywords"],
				"per_page", relaxedBody["per_page"],
			)
			relaxedSR, relaxedErr := searchOrganizationsWithFallback(ctx, relaxedBody)
			if relaxedErr != nil {
				return nil, relaxedErr
			}
			sr = relaxedSR
		}
		rawPerQuery = append(rawPerQuery, fmt.Sprintf("%s:%d", plan.Name, len(sr.Organizations)))
		for _, row := range sr.Organizations {
			org := orgFromResponse(row)
			if org == nil {
				continue
			}
			org.Signals = extractSignals(org)
			org.PriorityScore = priorityScore(org, filters)
			key := discoveryDedupKey(org)
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(org.Name))
			}
			if key == "" {
				continue
			}
			if existing, ok := merged[key]; ok {
				merged[key] = mergeOrg(existing, *org)
				continue
			}
			merged[key] = *org
		}
		slog.Info("apollo: company search complete",
			"query_name", plan.Name,
			"query", plan.Query,
			"results", len(sr.Organizations),
			"merged_total", len(merged),
			"industries", strings.Join(filters.Industries, ","),
			"location", strings.TrimSpace(filters.Location),
			"employee_ranges", strings.Join(plan.EmployeeRanges, ","),
			"keyword", strings.TrimSpace(filters.Keyword),
		)
	}
	out := make([]Org, 0, minInt(limit, len(merged)))
	for _, org := range merged {
		out = append(out, org)
	}
	sortOrgsByPriority(out)
	if len(out) > limit {
		out = out[:limit]
	}
	slog.Info("apollo: company search merged",
		"queries_executed", len(plans),
		"raw_results_per_query", strings.Join(rawPerQuery, ","),
		"final_merged_count", len(out),
	)
	return out, nil
}

// FiltersFromICP converts current ICP settings into a simple Apollo company search.
func FiltersFromICP(cfg *domain.ICPRuntimeSettings, limit int) CompanySearchFilters {
	f := CompanySearchFilters{
		Industries: discoveryQueriesForICP(cfg),
		Limit:      limit,
	}
	if cfg == nil {
		return f
	}
	f.MinEmployees = cfg.MinEmployees
	f.MaxEmployees = cfg.MaxEmployees
	f.Location = strings.TrimSpace(cfg.RegionFocus)
	return f
}

// ExampleCompanySearchPayload returns a representative Apollo Search API payload.
func ExampleCompanySearchPayload() map[string]any {
	plans := buildSearchPlans(CompanySearchFilters{
		Industries:   []string{"Banking"},
		MinEmployees: 200,
		MaxEmployees: 10000,
		Location:     "idn",
		Keyword:      "compliance",
		Limit:        10,
	})
	return buildCompanySearchPayload("YOUR_APOLLO_API_KEY", CompanySearchFilters{
		Industries:   []string{"Banking"},
		MinEmployees: 200,
		MaxEmployees: 10000,
		Location:     "idn",
		Keyword:      "compliance",
		Limit:        10,
	}, plans[0])
}

func searchOrganizations(ctx context.Context, body map[string]any) (*searchResponse, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	slog.Info("apollo: request",
		"endpoint", organizationsSearchEndpoint,
		"payload", string(raw),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, organizationsSearchEndpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("apollo: read response body: %w", err)
	}
	bodyPreview := truncateForLog(string(bodyBytes), 800)
	slog.Info("apollo: response",
		"endpoint", organizationsSearchEndpoint,
		"status_code", resp.StatusCode,
		"body", bodyPreview,
	)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apollo: organizations search HTTP %d body=%s", resp.StatusCode, bodyPreview)
	}
	var sr searchResponse
	if err := json.Unmarshal(bodyBytes, &sr); err != nil {
		return nil, fmt.Errorf("apollo: decode response: %w", err)
	}
	slog.Info("apollo: decoded response",
		"endpoint", organizationsSearchEndpoint,
		"results", len(sr.Organizations),
	)
	return &sr, nil
}

func searchOrganizationsWithFallback(ctx context.Context, body map[string]any) (*searchResponse, error) {
	sr, err := searchOrganizations(ctx, body)
	if err == nil {
		return sr, nil
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		return nil, err
	}
	reduced := map[string]any{
		"api_key":  body["api_key"],
		"page":     body["page"],
		"per_page": body["per_page"],
	}
	if keywords, ok := body["q_organization_keyword"]; ok {
		reduced["q_organization_keyword"] = keywords
	}
	sr, err = searchOrganizations(ctx, reduced)
	if err == nil {
		return sr, nil
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		return nil, err
	}
	minimal := map[string]any{
		"api_key":  body["api_key"],
		"page":     body["page"],
		"per_page": body["per_page"],
	}
	return searchOrganizations(ctx, minimal)
}

func orgFromResponse(o organizationRow) *Org {
	loc := strings.TrimSpace(o.LocalizedLocation)
	if loc == "" {
		loc = strings.TrimSpace(o.RawAddress)
	}
	return &Org{
		ID:                    strings.TrimSpace(o.ID),
		Name:                  strings.TrimSpace(o.Name),
		PrimaryDomain:         companycheck.SanitizeCompanyWebsiteDomain(strings.TrimSpace(o.PrimaryDomain)),
		Industry:              strings.TrimSpace(o.Industry),
		Location:              loc,
		ShortDescription:      strings.TrimSpace(o.ShortDescription),
		LinkedInURL:           strings.TrimSpace(o.LinkedinURL),
		EstimatedNumEmployees: o.EstimatedNumEmployees,
	}
}

func discoveryDedupKey(org *Org) string {
	if org == nil {
		return ""
	}
	if org.PrimaryDomain != "" {
		return strings.ToLower(strings.TrimSpace(org.PrimaryDomain))
	}
	if org.ID != "" {
		return "apollo:" + strings.ToLower(strings.TrimSpace(org.ID))
	}
	return ""
}

func buildCompanySearchPayload(apiKey string, filters CompanySearchFilters, plan searchPlan) map[string]any {
	body := BuildApolloPayloadByICP(primaryIndustry(filters.Industries))
	body["api_key"] = strings.TrimSpace(apiKey)
	if plan.PerPage > 0 {
		body["per_page"] = plan.PerPage
	}
	if q := strings.TrimSpace(plan.Query); q != "" {
		body["q_organization_keyword"] = q
	}
	ranges := plan.EmployeeRanges
	if len(ranges) == 0 {
		ranges = employeeRanges(filters.MinEmployees, filters.MaxEmployees)
	}
	if len(ranges) > 0 {
		body["organization_num_employees_ranges"] = ranges
	}
	if regions := locationFilters(strings.TrimSpace(filters.Location)); len(regions) > 0 {
		body["organization_locations"] = regions
	}
	return body
}

func buildRelaxedCompanySearchPayload(apiKey string, filters CompanySearchFilters, plan searchPlan) map[string]any {
	query := strings.TrimSpace(plan.Query)
	if query == "" {
		query = firstNonEmpty(strings.TrimSpace(filters.Keyword), firstStructuredIndustryQuery(filters.Industries))
	}
	body := map[string]any{
		"api_key":                strings.TrimSpace(apiKey),
		"page":                   1,
		"per_page":               10,
		"q_organization_keyword": firstNonEmpty(query, defaultApolloKeyword),
	}
	return body
}

func searchQueries(filters CompanySearchFilters) []string {
	base := filters.Industries
	if len(base) == 0 {
		base = []string{""}
	}
	keyword := strings.TrimSpace(filters.Keyword)
	out := make([]string, 0, len(base))
	for _, industry := range base {
		industry = strings.TrimSpace(industry)
		switch {
		case industry != "" && keyword != "":
			out = append(out, industry+" "+keyword)
		case industry != "":
			out = append(out, industry)
		case keyword != "":
			out = append(out, keyword)
		default:
			out = append(out, "")
		}
	}
	return out
}

func buildSearchPlans(filters CompanySearchFilters) []searchPlan {
	coreRanges := []string{"500,1000", "1000,5000", "5000,10000"}
	expansionRanges := []string{"200,500", "500,1000"}
	signalRanges := expandedEmployeeRanges()
	queries := structuredIndustryQueries(filters.Industries)
	if len(queries) == 0 {
		queries = []string{strings.TrimSpace(filters.Keyword)}
	}
	seen := map[string]struct{}{}
	var plans []searchPlan
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		addSearchPlan(&plans, seen, searchPlan{Name: "core", Query: q, EmployeeRanges: coreRanges, PerPage: 25})
		addSearchPlan(&plans, seen, searchPlan{Name: "expansion", Query: q, EmployeeRanges: expansionRanges, PerPage: 25})
		signalQuery := strings.TrimSpace(strings.Join([]string{q, firstNonEmpty(strings.TrimSpace(filters.Keyword), "training compliance")}, " "))
		addSearchPlan(&plans, seen, searchPlan{Name: "signal", Query: signalQuery, EmployeeRanges: signalRanges, PerPage: 25})
	}
	return plans
}

func addSearchPlan(plans *[]searchPlan, seen map[string]struct{}, plan searchPlan) {
	key := plan.Name + "|" + strings.TrimSpace(plan.Query) + "|" + strings.Join(plan.EmployeeRanges, ",")
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*plans = append(*plans, plan)
}

func structuredIndustryQueries(industries []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, industry := range industries {
		for _, term := range industrySearchTerms(industry) {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			out = append(out, term)
		}
	}
	return out
}

func firstStructuredIndustryQuery(industries []string) string {
	queries := structuredIndustryQueries(industries)
	if len(queries) == 0 {
		return ""
	}
	return queries[0]
}

func primaryIndustry(industries []string) string {
	if len(industries) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(industries[0]))
}

func industrySearchTerms(industry string) []string {
	switch strings.ToLower(strings.TrimSpace(industry)) {
	case "banking":
		return []string{"bank", "financial services"}
	case "retail":
		return []string{"retail", "store network"}
	case "hospitality":
		return []string{"hospitality", "hotel group"}
	case "manufacturing":
		return []string{"manufacturing", "factory operations"}
	case "healthcare":
		return []string{"healthcare", "hospital network"}
	case "education":
		return []string{"education", "school network"}
	case "technology":
		return []string{"technology", "software company"}
	case "logistics":
		return []string{"logistics", "distribution network"}
	case "fmcg":
		return []string{"fmcg", "consumer goods"}
	default:
		if label := strings.TrimSpace(industry); label != "" {
			return []string{strings.ToLower(label)}
		}
		return nil
	}
}

func expandedEmployeeRanges() []string {
	return []string{"200,500", "500,1000", "1000,5000", "5000,10000"}
}

func mergeOrg(existing, incoming Org) Org {
	if incoming.PriorityScore > existing.PriorityScore {
		existing.PriorityScore = incoming.PriorityScore
	}
	existing.Signals = mergeSignals(existing.Signals, incoming.Signals)
	if existing.Name == "" {
		existing.Name = incoming.Name
	}
	if existing.Industry == "" {
		existing.Industry = incoming.Industry
	}
	if existing.Location == "" {
		existing.Location = incoming.Location
	}
	if existing.ShortDescription == "" {
		existing.ShortDescription = incoming.ShortDescription
	}
	if existing.LinkedInURL == "" {
		existing.LinkedInURL = incoming.LinkedInURL
	}
	if existing.EstimatedNumEmployees == 0 {
		existing.EstimatedNumEmployees = incoming.EstimatedNumEmployees
	}
	if existing.PrimaryDomain == "" {
		existing.PrimaryDomain = incoming.PrimaryDomain
	}
	if existing.ID == "" {
		existing.ID = incoming.ID
	}
	return existing
}

func mergeSignals(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	seen := map[string]struct{}{}
	for _, list := range [][]string{a, b} {
		for _, s := range list {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func extractSignals(org *Org) []string {
	if org == nil {
		return nil
	}
	text := strings.ToLower(strings.Join([]string{org.Name, org.Industry, org.ShortDescription}, " "))
	var out []string
	if containsAny(text, "compliance", "regulatory", "certification", "governance", "risk") {
		out = append(out, "compliance-heavy")
	}
	if containsAny(text, "branch", "branches", "multi-site", "multi branch", "multi-branch", "outlet", "network", "chain") {
		out = append(out, "multi-site / multi-branch")
	}
	if containsAny(text, "training", "learning", "academy", "workforce", "frontline", "onboarding", "enablement") {
		out = append(out, "training / workforce scale")
	}
	return out
}

func priorityScore(org *Org, filters CompanySearchFilters) int {
	if org == nil {
		return 0
	}
	score := 0
	if industryMatches(org, filters.Industries) {
		score += 40
	}
	if sizeMatches(org.EstimatedNumEmployees) {
		score += 30
	}
	if keywordOrSignalMatches(org, filters.Keyword) {
		score += 20
	}
	if locationMatches(org.Location, filters.Location) {
		score += 10
	}
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func industryMatches(org *Org, industries []string) bool {
	if org == nil {
		return false
	}
	hay := strings.ToLower(org.Industry)
	for _, industry := range industries {
		for _, term := range industrySearchTerms(industry) {
			if strings.Contains(hay, strings.ToLower(term)) {
				return true
			}
		}
	}
	return len(industries) == 0
}

func sizeMatches(n int) bool {
	for _, r := range expandedEmployeeRanges() {
		lo, hi := parseEmployeeRange(r)
		if n >= lo && n <= hi {
			return true
		}
	}
	return false
}

func keywordOrSignalMatches(org *Org, keyword string) bool {
	if org == nil {
		return false
	}
	if strings.TrimSpace(keyword) != "" {
		text := strings.ToLower(strings.Join([]string{org.Name, org.Industry, org.ShortDescription}, " "))
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return len(org.Signals) > 0
}

func locationMatches(location, region string) bool {
	loc := strings.ToLower(strings.TrimSpace(location))
	for _, allowed := range locationFilters(region) {
		if strings.Contains(loc, strings.ToLower(allowed)) {
			return true
		}
	}
	return strings.TrimSpace(region) == ""
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(keyword))) {
			return true
		}
	}
	return false
}

func parseEmployeeRange(v string) (int, int) {
	parts := strings.Split(strings.TrimSpace(v), ",")
	if len(parts) != 2 {
		return 0, 0
	}
	lo, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	hi, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return lo, hi
}

func sortOrgsByPriority(out []Org) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PriorityScore == out[j].PriorityScore {
			return strings.ToLower(out[i].PrimaryDomain) < strings.ToLower(out[j].PrimaryDomain)
		}
		return out[i].PriorityScore > out[j].PriorityScore
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateForLog(v string, max int) string {
	v = strings.TrimSpace(v)
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max] + "...(truncated)"
}

func discoveryQueriesForICP(cfg *domain.ICPRuntimeSettings) []string {
	if cfg == nil {
		return []string{"banking", "retail", "hospitality"}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(cfg.TargetIndustryIDs))
	for _, id := range cfg.TargetIndustryIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if label := industryLabel(id); label != "" {
			out = append(out, label)
		}
	}
	return out
}

func industryLabel(id string) string {
	for _, opt := range icp.CatalogIndustries() {
		if strings.EqualFold(strings.TrimSpace(opt.ID), strings.TrimSpace(id)) {
			return opt.Label
		}
	}
	return strings.TrimSpace(id)
}

func employeeRangesForICP(cfg *domain.ICPRuntimeSettings) []string {
	if cfg == nil {
		return nil
	}
	lo := cfg.MinEmployees
	if lo <= 0 && cfg.ApplySub50Rule {
		lo = 50
	}
	return employeeRanges(lo, cfg.MaxEmployees)
}

func employeeRanges(lo, hi int) []string {
	switch {
	case lo > 0 && hi > 0:
		return []string{fmt.Sprintf("%d,%d", lo, hi)}
	case lo > 0:
		return []string{fmt.Sprintf("%d,", lo)}
	case hi > 0:
		return []string{fmt.Sprintf(",%d", hi)}
	default:
		return nil
	}
}

func locationsForICP(cfg *domain.ICPRuntimeSettings) []string {
	if cfg == nil {
		return nil
	}
	return locationFilters(cfg.RegionFocus)
}

func locationFilters(region string) []string {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "idn":
		return []string{"Indonesia"}
	case "sea":
		return []string{"Indonesia", "Singapore", "Malaysia", "Thailand", "Philippines", "Vietnam"}
	default:
		return nil
	}
}
