// Package status maps ICP + dedup results to a final lead status (pure mapping).
package status

import (
	"context"

	"salesradar/internal/domain"
	"salesradar/internal/review"
)

// AssignStatus sets LeadStatus from icp_match and duplicate_status only.
func AssignStatus(ctx context.Context, d *domain.DedupedLead) (*domain.StagedOdooLead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}

	var st domain.LeadStatus
	switch d.ICPMatch {
	case domain.ICPNo:
		st = domain.StatusDiscarded
	case domain.ICPPartial:
		st = domain.StatusNeedsReview
	case domain.ICPYes:
		switch d.DuplicateStatus {
		case domain.DupExact:
			st = domain.StatusDiscarded
		case domain.DupSuspectedDuplicate:
			st = domain.StatusNeedsReview
		default:
			st = domain.StatusNew
		}
	default:
		st = domain.StatusDiscarded
	}

	out := &domain.StagedOdooLead{
		DedupedLead: *d,
		Status:      st,
	}
	out.Explanation = review.BuildExplanation(*out)
	return out, nil
}
