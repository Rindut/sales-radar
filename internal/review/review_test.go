package review

import (
	"strings"
	"testing"

	"salesradar/internal/domain"
)

func ptr(s string) *string { return &s }

func staged(base domain.StagedOdooLead) domain.StagedOdooLead {
	base.Explanation = BuildExplanation(base)
	return base
}

func TestBuildExplanation_CleanYes(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ExtractedLead: domain.ExtractedLead{
					CompanyName: ptr("PT Bank"),
				},
				ICPMatch: domain.ICPYes,
				ICPReason: []string{
					"banking industry signal detected",
					"size appears above 1000 employees",
				},
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusNew,
	})
	if len(l.Explanation) < 1 || len(l.Explanation) > 2 {
		t.Fatalf("want 1–2 icp-only lines, got %d: %v", len(l.Explanation), l.Explanation)
	}
	if strings.Contains(strings.Join(l.Explanation, " "), "duplicate") {
		t.Fatalf("unexpected duplicate note: %v", l.Explanation)
	}
}

func TestBuildExplanation_PartialAmbiguity(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ICPMatch: domain.ICPPartial,
				ICPReason: []string{
					"industry plausible but not explicit",
					"size unclear or missing",
				},
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusNeedsReview,
	})
	text := strings.ToLower(strings.Join(l.Explanation, " "))
	if !strings.Contains(text, "requires manual review") {
		t.Fatalf("expected review hint, got %v", l.Explanation)
	}
	if !strings.Contains(text, "unclear") && !strings.Contains(text, "explicit") {
		t.Fatalf("expected partial/ambiguity signal, got %v", l.Explanation)
	}
}

func TestBuildExplanation_ExactDuplicate(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ICPMatch:  domain.ICPYes,
				ICPReason: []string{"banking industry signal detected"},
			},
			DuplicateStatus: domain.DupExact,
		},
		Status: domain.StatusDiscarded,
	})
	text := strings.Join(l.Explanation, " ")
	if !strings.Contains(text, "exact duplicate detected") {
		t.Fatalf("expected exact dup line: %v", l.Explanation)
	}
	if !strings.Contains(text, "exact duplicate blocks push") {
		t.Fatalf("expected discard line: %v", l.Explanation)
	}
}

func TestBuildExplanation_DuplicateSuspected(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ICPMatch:  domain.ICPYes,
				ICPReason: []string{"banking industry signal detected"},
			},
			DuplicateStatus: domain.DupSuspectedDuplicate,
		},
		Status: domain.StatusNeedsReview,
	})
	text := strings.Join(l.Explanation, " ")
	if !strings.Contains(text, "suspected duplicate based on name similarity") {
		t.Fatalf("expected suspected dup line: %v", l.Explanation)
	}
}

func TestBuildExplanation_NoICP(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ICPMatch:  domain.ICPNo,
				ICPReason: []string{"non-target industry"},
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusDiscarded,
	})
	text := strings.ToLower(strings.Join(l.Explanation, " "))
	if !strings.Contains(text, "discarded due to icp mismatch") {
		t.Fatalf("expected discard reason, got %v", l.Explanation)
	}
	if !strings.Contains(text, "non-target") {
		t.Fatalf("expected disqualification from ICP, got %v", l.Explanation)
	}
}

func TestBuildReviewLead_mapsFields(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ExtractedLead: domain.ExtractedLead{
					CompanyName:          ptr("Co"),
					Industry:             ptr("banking"),
					CompanySizeEstimated: ptr("5000"),
				},
				ICPMatch:  domain.ICPYes,
				ICPReason: []string{"banking signal"},
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusNew,
	})
	rl := BuildReviewLead(l)
	if rl.Company == nil || *rl.Company != "Co" {
		t.Fatal("company mapping")
	}
	if rl.Size != "5000" {
		t.Fatalf("size display = %q", rl.Size)
	}
	if rl.ICPMatch != "high" || rl.LeadStatus != "new" {
		t.Fatal("string mapping")
	}
	if rl.Confidence != ConfidenceHigh {
		t.Fatalf("confidence = %s", rl.Confidence)
	}
	if rl.Summary == "" {
		t.Fatal("summary empty")
	}
	if len(rl.Reasons) == 0 {
		t.Fatal("reasons empty")
	}
}

func TestConfidenceMapping(t *testing.T) {
	tests := []struct {
		name string
		l    domain.StagedOdooLead
		want string
	}{
		{
			"high yes new",
			staged(domain.StagedOdooLead{
				DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPYes}, DuplicateStatus: domain.DupNew},
				Status:      domain.StatusNew,
			}),
			ConfidenceHigh,
		},
		{
			"medium partial",
			staged(domain.StagedOdooLead{
				DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPPartial}, DuplicateStatus: domain.DupNew},
				Status:      domain.StatusNeedsReview,
			}),
			ConfidenceMedium,
		},
		{
			"medium suspected",
			staged(domain.StagedOdooLead{
				DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPYes}, DuplicateStatus: domain.DupSuspectedDuplicate},
				Status:      domain.StatusNeedsReview,
			}),
			ConfidenceMedium,
		},
		{
			"low no",
			staged(domain.StagedOdooLead{
				DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPNo}, DuplicateStatus: domain.DupNew},
				Status:      domain.StatusDiscarded,
			}),
			ConfidenceLow,
		},
		{
			"low discarded exact",
			staged(domain.StagedOdooLead{
				DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPYes}, DuplicateStatus: domain.DupExact},
				Status:      domain.StatusDiscarded,
			}),
			ConfidenceLow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := BuildReviewLead(tt.l)
			if rl.Confidence != tt.want {
				t.Fatalf("got %s want %s", rl.Confidence, tt.want)
			}
		})
	}
}

func TestSummaryGeneration(t *testing.T) {
	yesNew := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPYes}, DuplicateStatus: domain.DupNew},
		Status:      domain.StatusNew,
	})
	if s := BuildReviewLead(yesNew).Summary; !strings.Contains(s, "ICP-qualified") {
		t.Fatalf("yes+new summary: %q", s)
	}

	partial := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPPartial}, DuplicateStatus: domain.DupNew},
		Status:      domain.StatusNeedsReview,
	})
	if s := BuildReviewLead(partial).Summary; !strings.Contains(s, "Partial ICP") {
		t.Fatalf("partial summary: %q", s)
	}

	dup := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{ICPLead: domain.ICPLead{ICPMatch: domain.ICPYes}, DuplicateStatus: domain.DupExact},
		Status:      domain.StatusDiscarded,
	})
	if s := BuildReviewLead(dup).Summary; !strings.Contains(strings.ToLower(s), "duplicate") {
		t.Fatalf("duplicate summary: %q", s)
	}

	if w := len(strings.Fields(BuildReviewLead(yesNew).Summary)); w > 12 {
		t.Fatalf("summary too long: %d words", w)
	}
}

func TestNormalizeSizeDisplay_overEmployees(t *testing.T) {
	raw := "over 1000 employees"
	if got := NormalizeSizeDisplay(&raw); got != "1000+" {
		t.Fatalf("got %q want 1000+", got)
	}
}

func TestComputeAction_partialICPRelevantIndustry_isContact(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ExtractedLead: domain.ExtractedLead{
					CompanyName:         ptr("Bank Central Asia"),
					Industry:            ptr("banking"),
					OfficialDomain:      "bca.co.id",
					UnstructuredContext: "banking training onboarding",
				},
				ICPMatch: domain.ICPPartial,
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusNeedsReview,
	})
	rl := BuildReviewLead(l)
	rl.ReasonForFit = "Banking sector with onboarding and compliance training needs"
	rl = ApplySalesStatusAndCopy(l, rl)
	if rl.Action != ActionContact {
		t.Fatalf("action = %q, want %q", rl.Action, ActionContact)
	}
}

func TestComputeAction_unknownIndustry_isResearchFirst(t *testing.T) {
	l := staged(domain.StagedOdooLead{
		DedupedLead: domain.DedupedLead{
			ICPLead: domain.ICPLead{
				ExtractedLead: domain.ExtractedLead{
					CompanyName:    ptr("Bank Mandiri"),
					OfficialDomain: "bankmandiri.co.id",
				},
				ICPMatch: domain.ICPPartial,
			},
			DuplicateStatus: domain.DupNew,
		},
		Status: domain.StatusNeedsReview,
	})
	rl := BuildReviewLead(l)
	rl.ReasonForFit = "Needs more industry confirmation"
	rl = ApplySalesStatusAndCopy(l, rl)
	if rl.Action != ActionResearchFirst {
		t.Fatalf("action = %q, want %q", rl.Action, ActionResearchFirst)
	}
}
