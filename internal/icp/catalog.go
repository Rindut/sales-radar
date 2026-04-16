package icp

// IndustryOption is a target or excluded industry in Settings.
type IndustryOption struct {
	ID     string // catalog slug (e.g. banking, manufacturing)
	Label  string
	Helper string // optional; UI may omit
}

// SignalOption is an intent / discovery signal group for soft scoring.
type SignalOption struct {
	ID       string
	Label    string
	Helper   string
	Keywords []string
}

// ExclusionOption is a non-target segment used in soft scoring.
type ExclusionOption struct {
	ID       string
	Label    string
	Helper   string
	Keywords []string
}

// RegionOption is optional geographic focus for a small score adjustment.
type RegionOption struct {
	ID     string
	Label  string
	Helper string
}

// CatalogIndustries returns industries Sales can target.
func CatalogIndustries() []IndustryOption {
	return []IndustryOption{
		{ID: "banking", Label: "Banking"},
		{ID: "retail", Label: "Retail"},
		{ID: "hospitality", Label: "Hospitality"},
		{ID: "manufacturing", Label: "Manufacturing"},
		{ID: "healthcare", Label: "Healthcare"},
		{ID: "education", Label: "Education"},
		{ID: "technology", Label: "Technology"},
		{ID: "logistics", Label: "Logistics"},
		{ID: "fmcg", Label: "FMCG"},
		{ID: "others", Label: "Others"},
	}
}

// CatalogSignals returns intent groups matched from discovery context.
func CatalogSignals() []SignalOption {
	return []SignalOption{
		{
			ID: "hiring_workforce", Label: "Hiring & workforce pressure",
			Helper: "Hiring, turnover, frontline HR.",
			Keywords: []string{
				"hiring", "turnover", "workforce", "frontline", "hr ",
			},
		},
		{
			ID: "compliance_training", Label: "Compliance-heavy training",
			Helper: "Regulatory and certification needs.",
			Keywords: []string{
				"compliance", "certification",
			},
		},
		{
			ID: "learning_ops", Label: "Learning & onboarding",
			Helper: "L&D and training tools.",
			Keywords: []string{
				"training", "onboarding", "learning", "l&d", "l and d",
			},
		},
		{
			ID: "expansion_growth", Label: "Expansion & growth",
			Helper: "Multi-branch or rollout growth.",
			Keywords: []string{
				"expansion", "growth_signal", "multi-branch", "multi-outlet",
			},
		},
		{
			ID: "multi_site_ops", Label: "Multi-site operational scale",
			Helper: "Branch or outlet networks.",
			Keywords: []string{
				"branch", "outlet",
			},
		},
		{
			ID: "operational_transformation", Label: "Operational & digital transformation",
			Helper: "Standardization and operational change.",
			Keywords: []string{
				"structured", "operational", "standardization", "service standard", "digital", "transformation",
			},
		},
		{
			ID: "job_posting_signal", Label: "Job-posting signal",
			Helper: "Role demand from job discovery.",
			Keywords: []string{
				"job_signal", "job_signal_detected",
			},
		},
	}
}

// CatalogExclusions returns non-target segment hints.
func CatalogExclusions() []ExclusionOption {
	return []ExclusionOption{
		{
			ID: "freelance_agency", Label: "Freelance & agency-only profiles",
			Helper: "Solo or agency-style businesses.",
			Keywords: []string{
				"freelance", "freelancer", "solo consultant", "agency-only", "creative agency",
			},
		},
		{
			ID: "micro_enterprise", Label: "Micro-enterprise profiles",
			Helper: "Very small local businesses.",
			Keywords: []string{
				"umkm", "micro business", "micro-enterprise", "sme micro", "warung",
			},
		},
	}
}

// CatalogRegions returns region focus for soft scoring.
func CatalogRegions() []RegionOption {
	return []RegionOption{
		{ID: "idn", Label: "Indonesia"},
		{ID: "sea", Label: "ASEAN"},
		{ID: "", Label: "Global"},
	}
}

// CatalogWeights is the priority weight labels for scoring (High / Medium / Low).
func CatalogWeights() []string {
	return []string{"high", "medium", "low"}
}

func signalKeywordsByIDs(ids []string) [][]string {
	if len(ids) == 0 {
		var all [][]string
		for _, s := range CatalogSignals() {
			all = append(all, s.Keywords)
		}
		return all
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out [][]string
	for _, s := range CatalogSignals() {
		if _, ok := want[s.ID]; ok {
			out = append(out, s.Keywords)
		}
	}
	if len(out) == 0 {
		return signalKeywordsByIDs(nil)
	}
	return out
}

func exclusionKeywordsByIDs(ids []string) [][]string {
	if len(ids) == 0 {
		return nil
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out [][]string
	for _, s := range CatalogExclusions() {
		if _, ok := want[s.ID]; ok {
			out = append(out, s.Keywords)
		}
	}
	return out
}
