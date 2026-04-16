// Package pipeline runs the Sales Radar Phase 1 pipeline (discovery → Odoo noop).
// Shared by CLI and web UI without changing step logic.
package pipeline

import (
	"context"

	"salesradar/internal/deduplication"
	"salesradar/internal/discovery"
	"salesradar/internal/domain"
	"salesradar/internal/extraction"
	"salesradar/internal/icp"
	"salesradar/internal/odoo"
	"salesradar/internal/status"
)

type noopDedupStore struct{}

func (noopDedupStore) ExactNameDomainExists(context.Context, string, string) (bool, error) {
	return false, nil
}
func (noopDedupStore) StrongNameMatchExists(context.Context, string) (bool, error) {
	return false, nil
}

type noopOdooClient struct{}

func (noopOdooClient) CreateLead(context.Context, domain.StagedOdooLead) (*string, error) {
	return nil, nil
}

// Run executes the full pipeline and returns staged leads (same behavior as CLI).
func Run(ctx context.Context, params domain.RunParams) ([]domain.StagedOdooLead, error) {
	raw, err := discovery.Discover(ctx, params)
	if err != nil {
		return nil, err
	}

	store := noopDedupStore{}
	client := noopOdooClient{}

	var stagedLeads []domain.StagedOdooLead
	for _, c := range raw {
		ext, err := extraction.Extract(ctx, c)
		if err != nil {
			return nil, err
		}
		icpLead, err := icp.Evaluate(ctx, ext, nil)
		if err != nil {
			return nil, err
		}
		deduped, err := deduplication.Classify(ctx, icpLead, store)
		if err != nil {
			return nil, err
		}
		staged, err := status.AssignStatus(ctx, deduped)
		if err != nil {
			return nil, err
		}
		stagedLeads = append(stagedLeads, *staged)
		_, err = odoo.Push(ctx, staged, client)
		if err != nil {
			return nil, err
		}
	}

	return stagedLeads, nil
}

// DefaultRunParams matches the CLI batch and source allowlist.
func DefaultRunParams() domain.RunParams {
	return domain.RunParams{
		MaxLeadsThisRun: domain.MaxLeadsPerRunDefault,
		SourceAllowlist: []domain.Source{
			domain.SourceLinkedIn,
			domain.SourceApollo,
			domain.SourceCompanyWebsite,
			domain.SourceGoogle,
			domain.SourceJobPortal,
		},
	}
}
