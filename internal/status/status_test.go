package status

import (
	"context"
	"testing"

	"salesradar/internal/domain"
)

func deduped(icp domain.ICPMatch, dup domain.DuplicateStatus) *domain.DedupedLead {
	return &domain.DedupedLead{
		ICPLead: domain.ICPLead{
			ICPMatch: icp,
		},
		DuplicateStatus: dup,
	}
}

func TestAssignStatus_YesNew(t *testing.T) {
	got, err := AssignStatus(context.Background(), deduped(domain.ICPYes, domain.DupNew))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusNew {
		t.Fatalf("status=%s want=%s", got.Status, domain.StatusNew)
	}
}

func TestAssignStatus_YesDuplicateDiscarded(t *testing.T) {
	got, err := AssignStatus(context.Background(), deduped(domain.ICPYes, domain.DupExact))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusDiscarded {
		t.Fatalf("status=%s want=%s", got.Status, domain.StatusDiscarded)
	}
}

func TestAssignStatus_YesSuspectedNeedsReview(t *testing.T) {
	got, err := AssignStatus(context.Background(), deduped(domain.ICPYes, domain.DupSuspectedDuplicate))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusNeedsReview {
		t.Fatalf("status=%s want=%s", got.Status, domain.StatusNeedsReview)
	}
}

func TestAssignStatus_PartialAnyNeedsReview(t *testing.T) {
	for _, dup := range []domain.DuplicateStatus{
		domain.DupNew,
		domain.DupExact,
		domain.DupSuspectedDuplicate,
	} {
		got, err := AssignStatus(context.Background(), deduped(domain.ICPPartial, dup))
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.StatusNeedsReview {
			t.Fatalf("partial + %s: status=%s want=%s", dup, got.Status, domain.StatusNeedsReview)
		}
	}
}

func TestAssignStatus_NoAnyDiscarded(t *testing.T) {
	for _, dup := range []domain.DuplicateStatus{
		domain.DupNew,
		domain.DupExact,
		domain.DupSuspectedDuplicate,
	} {
		got, err := AssignStatus(context.Background(), deduped(domain.ICPNo, dup))
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.StatusDiscarded {
			t.Fatalf("no + %s: status=%s want=%s", dup, got.Status, domain.StatusDiscarded)
		}
	}
}
