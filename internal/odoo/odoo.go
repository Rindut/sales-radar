// Package odoo writes staged leads to Odoo with retry policy in implementation.
package odoo

import (
	"context"

	"salesradar/internal/domain"
)

// Client is the external boundary to Odoo (HTTP/XML-RPC in a real adapter).
type Client interface {
	CreateLead(ctx context.Context, lead domain.StagedOdooLead) (*string, error)
}

// Push attempts to persist a staged lead up to the caller's retry policy.
func Push(ctx context.Context, lead *domain.StagedOdooLead, c Client) (*domain.OdooPushResult, error) {
	return nil, nil
}
