package icp

import (
	"context"
	"testing"

	"salesradar/internal/domain"
)

func ptr(s string) *string { return &s }

func TestEvaluate_BankingYes(t *testing.T) {
	in := &domain.ExtractedLead{
		Industry:             ptr("banking"),
		CompanySizeEstimated: ptr("1200"),
		OfficialDomain:       "bca.co.id",
		UnstructuredContext:  "multi-branch compliance training onboarding L&D hiring frontline workforce",
	}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPYes {
		t.Fatalf("match = %s, want yes", out.ICPMatch)
	}
	if out.ICPIndustryBucket != domain.BucketBanking {
		t.Fatalf("bucket = %s", out.ICPIndustryBucket)
	}
	if len(out.ICPReason) == 0 {
		t.Fatal("expected reasons")
	}
	if out.ICPScore < 70 || out.ICPScore > 100 {
		t.Fatalf("ICPScore = %d, want high band 70–100", out.ICPScore)
	}
	if out.ScoreAction != domain.ScoreActionContact {
		t.Fatalf("ScoreAction = %s, want Contact", out.ScoreAction)
	}
}

func TestEvaluate_RetailPartial_SizeMissing(t *testing.T) {
	in := &domain.ExtractedLead{Industry: ptr("retail")}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPPartial {
		t.Fatalf("match = %s, want partial", out.ICPMatch)
	}
	if out.ICPIndustryBucket != domain.BucketRetail {
		t.Fatalf("bucket = %s", out.ICPIndustryBucket)
	}
	if out.ScoreAction != domain.ScoreActionResearch {
		t.Fatalf("ScoreAction = %s, want Research", out.ScoreAction)
	}
}

func TestEvaluate_HospitalityNo_TooSmall(t *testing.T) {
	in := &domain.ExtractedLead{
		Industry:             ptr("hospitality"),
		CompanySizeEstimated: ptr("80"),
	}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPNo {
		t.Fatalf("match = %s, want no", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionReject {
		t.Fatalf("ScoreAction = %s, want Reject", out.ScoreAction)
	}
}

func TestEvaluate_ManufacturingNotInTargetList_No(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{
		TargetIndustryIDs: []string{"banking", "retail", "hospitality"},
		ApplySub50Rule:    true,
		WeightIndustry:    "medium", WeightSignal: "medium", WeightSize: "medium",
	}
	in := &domain.ExtractedLead{Industry: ptr("manufacturing")}
	out, err := Evaluate(context.Background(), in, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPNo {
		t.Fatalf("match = %s, want no", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionReject {
		t.Fatalf("ScoreAction = %s, want Reject", out.ScoreAction)
	}
}

func TestEvaluate_AmbiguousIndustry_NotYes(t *testing.T) {
	note := "Ambiguous industry signal from unstructured text."
	in := &domain.ExtractedLead{ExtractionNotes: &note}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch == domain.ICPYes {
		t.Fatalf("ambiguous industry must not be yes")
	}
}

func TestEvaluate_MissingEverything_No(t *testing.T) {
	in := &domain.ExtractedLead{}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPNo {
		t.Fatalf("match = %s, want no", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionReject {
		t.Fatalf("ScoreAction = %s, want Reject", out.ScoreAction)
	}
}

func TestEvaluate_RetailMeetsSizeWeakLXP_Partial(t *testing.T) {
	in := &domain.ExtractedLead{
		Industry:             ptr("retail"),
		CompanySizeEstimated: ptr("over 500 employees"),
		UnstructuredContext:  "fashion brand with stores; corporate description only; static market positioning",
	}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPPartial {
		t.Fatalf("match = %s, want partial (size OK, weak LXP text)", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionResearch {
		t.Fatalf("ScoreAction = %s, want Research", out.ScoreAction)
	}
}

func TestEvaluate_Below50Employees_No(t *testing.T) {
	in := &domain.ExtractedLead{
		Industry:             ptr("retail"),
		CompanySizeEstimated: ptr("40"),
		UnstructuredContext:  "training onboarding compliance workforce",
	}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPNo {
		t.Fatalf("match = %s, want no", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionReject {
		t.Fatalf("ScoreAction = %s, want Reject", out.ScoreAction)
	}
}

func TestEvaluate_StrongClassificationMissingIndustry_Partial(t *testing.T) {
	in := &domain.ExtractedLead{
		StrongClassification: true,
		CompanySizeEstimated: ptr("1200"),
	}
	out, err := Evaluate(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPPartial {
		t.Fatalf("match = %s, want partial", out.ICPMatch)
	}
	if out.ScoreAction != domain.ScoreActionResearch {
		t.Fatalf("ScoreAction = %s, want Research", out.ScoreAction)
	}
}

func TestEvaluate_TargetRetailOnly_BankingDisqualified(t *testing.T) {
	cfg := &domain.ICPRuntimeSettings{
		TargetIndustryIDs: []string{"retail"},
		TargetBuckets:     []domain.ICPIndustryBucket{domain.BucketRetail},
		ApplySub50Rule:    true,
		WeightIndustry:    "medium", WeightSignal: "medium", WeightSize: "medium",
	}
	in := &domain.ExtractedLead{
		Industry:             ptr("banking"),
		CompanySizeEstimated: ptr("5000"),
		UnstructuredContext:  "compliance training L&D hiring",
	}
	out, err := Evaluate(context.Background(), in, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.ICPMatch != domain.ICPNo {
		t.Fatalf("match = %s, want no when banking excluded from target set", out.ICPMatch)
	}
}
