package apollo

import (
	"testing"

	"salesradar/internal/domain"
)

func TestDiscoveryQueriesForICP_UsesTargetIndustries(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{
		TargetIndustryIDs: []string{"banking", "retail", "banking"},
	}
	got := discoveryQueriesForICP(cfg)
	if len(got) != 2 {
		t.Fatalf("len(discoveryQueriesForICP) = %d, want 2", len(got))
	}
	if got[0] != "Banking" || got[1] != "Retail" {
		t.Fatalf("unexpected queries: %#v", got)
	}
}

func TestEmployeeRangesForICP(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{MinEmployees: 100, MaxEmployees: 500}
	got := employeeRangesForICP(cfg)
	if len(got) != 1 || got[0] != "100,500" {
		t.Fatalf("employeeRangesForICP = %#v, want [\"100,500\"]", got)
	}
}

func TestLocationsForICP(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{RegionFocus: "idn"}
	got := locationsForICP(cfg)
	if len(got) != 1 || got[0] != "Indonesia" {
		t.Fatalf("locationsForICP = %#v, want [\"Indonesia\"]", got)
	}
}

func TestBuildCompanySearchPayload(t *testing.T) {
	body := buildCompanySearchPayload("test-key", CompanySearchFilters{
		Industries:   []string{"Banking"},
		MinEmployees: 200,
		MaxEmployees: 10000,
		Location:     "sea",
		Keyword:      "compliance",
		Limit:        10,
	}, searchPlan{
		Name:           "core",
		Query:          "bank",
		EmployeeRanges: []string{"500,1000", "1000,5000", "5000,10000"},
		PerPage:        25,
	})

	if body["api_key"] != "test-key" {
		t.Fatalf("api_key = %#v, want test-key", body["api_key"])
	}
	if body["per_page"] != 25 {
		t.Fatalf("per_page = %#v, want 25", body["per_page"])
	}
	if got := body["q_organization_keyword"].(string); got != "bank" {
		t.Fatalf("q_organization_keyword = %#v, want \"bank\"", body["q_organization_keyword"])
	}
	if got := body["organization_num_employees_ranges"].([]string); len(got) != 3 || got[0] != "500,1000" {
		t.Fatalf("organization_num_employees_ranges = %#v, want core ranges", body["organization_num_employees_ranges"])
	}
	if got := body["organization_locations"].([]string); len(got) != 6 || got[0] != "Indonesia" {
		t.Fatalf("organization_locations = %#v, want SEA locations", body["organization_locations"])
	}
}

func TestBuildRelaxedCompanySearchPayload(t *testing.T) {
	body := buildRelaxedCompanySearchPayload("test-key", CompanySearchFilters{
		Industries: []string{"Banking"},
		Location:   "sea",
		Keyword:    "compliance",
	}, searchPlan{
		Name:  "core",
		Query: "bank",
	})

	if body["api_key"] != "test-key" {
		t.Fatalf("api_key = %#v, want test-key", body["api_key"])
	}
	if body["per_page"] != 10 {
		t.Fatalf("per_page = %#v, want 10", body["per_page"])
	}
	if got := body["q_organization_keyword"].(string); got != "bank" {
		t.Fatalf("q_organization_keyword = %#v, want \"bank\"", body["q_organization_keyword"])
	}
	if _, ok := body["organization_locations"]; ok {
		t.Fatalf("organization_locations present in relaxed payload, want omitted")
	}
	if _, ok := body["organization_num_employees_ranges"]; ok {
		t.Fatalf("organization_num_employees_ranges present in relaxed payload, want omitted")
	}
}

func TestBuildApolloPayload_DefaultBroadRecall(t *testing.T) {
	body := BuildApolloPayload()
	if got := body["q_organization_keyword"].(string); got != defaultApolloKeyword {
		t.Fatalf("q_organization_keyword = %q, want %q", got, defaultApolloKeyword)
	}
	if got := body["per_page"].(int); got != 20 {
		t.Fatalf("per_page = %d, want 20", got)
	}
	if got := body["organization_locations"].([]string); len(got) != 1 || got[0] != "Indonesia" {
		t.Fatalf("organization_locations = %#v, want [\"Indonesia\"]", got)
	}
}

func TestBuildApolloPayloadByICP_Banking(t *testing.T) {
	body := BuildApolloPayloadByICP("banking")
	if got := body["q_organization_keyword"].(string); got != "bank OR lending OR financial services" {
		t.Fatalf("q_organization_keyword = %q, want banking broad query", got)
	}
}

func TestBuildSearchPlans_UsesStructuredQueries(t *testing.T) {
	got := buildSearchPlans(CompanySearchFilters{
		Industries: []string{"Banking"},
		Keyword:    "compliance",
	})
	if len(got) != 6 {
		t.Fatalf("len(buildSearchPlans) = %d, want 6", len(got))
	}
	if got[0].Name != "core" || got[0].Query != "bank" {
		t.Fatalf("first plan = %#v, want core bank", got[0])
	}
	if got[2].Name != "signal" || got[2].Query != "bank compliance" {
		t.Fatalf("signal plan = %#v, want bank compliance", got[2])
	}
}

func TestFiltersFromICP(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{
		TargetIndustryIDs: []string{"banking"},
		MinEmployees:      100,
		MaxEmployees:      500,
		RegionFocus:       "sea",
	}
	got := FiltersFromICP(cfg, 12)
	if got.Limit != 12 {
		t.Fatalf("Limit = %d, want 12", got.Limit)
	}
	if len(got.Industries) != 1 || got.Industries[0] != "Banking" {
		t.Fatalf("Industries = %#v, want [\"Banking\"]", got.Industries)
	}
	if got.MinEmployees != 100 || got.MaxEmployees != 500 {
		t.Fatalf("employee range = %d,%d, want 100,500", got.MinEmployees, got.MaxEmployees)
	}
	if got.Location != "sea" {
		t.Fatalf("Location = %q, want sea", got.Location)
	}
}

func TestOrgFromResponse_NormalizesDomain(t *testing.T) {
	org := orgFromResponse(organizationRow{
		ID:                    "123",
		Name:                  "PT Alpha",
		PrimaryDomain:         "www.alpha.co.id",
		Industry:              "Retail",
		LocalizedLocation:     "Indonesia",
		LinkedinURL:           "https://www.linkedin.com/company/alpha",
		EstimatedNumEmployees: 250,
	})
	if org.PrimaryDomain != "alpha.co.id" {
		t.Fatalf("PrimaryDomain = %q, want alpha.co.id", org.PrimaryDomain)
	}
	if org.Name != "PT Alpha" {
		t.Fatalf("Name = %q, want PT Alpha", org.Name)
	}
}

func TestMergeOrg_KeepsHighestPriorityAndSignals(t *testing.T) {
	got := mergeOrg(
		Org{PrimaryDomain: "alpha.co.id", PriorityScore: 40, Signals: []string{"compliance-heavy"}},
		Org{PrimaryDomain: "alpha.co.id", PriorityScore: 70, Signals: []string{"training / workforce scale"}},
	)
	if got.PriorityScore != 70 {
		t.Fatalf("PriorityScore = %d, want 70", got.PriorityScore)
	}
	if len(got.Signals) != 2 {
		t.Fatalf("Signals = %#v, want merged signal set", got.Signals)
	}
}
