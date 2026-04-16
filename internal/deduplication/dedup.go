// Package deduplication classifies duplicate status before status assignment.
package deduplication

import (
	"context"
	"net/url"
	"strings"
	"unicode"

	"salesradar/internal/domain"
)

// Store is the external boundary for CRM / index lookups (implement outside).
type Store interface {
	ExactNameDomainExists(ctx context.Context, normalizedName, domain string) (bool, error)
	StrongNameMatchExists(ctx context.Context, normalizedName string) (bool, error)
}

// Classify sets DuplicateStatus on the lead (new, exact duplicate, suspected).
func Classify(ctx context.Context, lead *domain.ICPLead, store Store) (*domain.DedupedLead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, nil
	}

	out := &domain.DedupedLead{
		ICPLead:          *lead,
		DuplicateStatus: domain.DupNew,
	}
	if store == nil || lead.CompanyName == nil {
		return out, nil
	}

	normalizedName := normalizeName(*lead.CompanyName)
	if normalizedName == "" {
		return out, nil
	}

	domainPart := domainFromRef(lead.SourceRef)
	if domainPart != "" {
		exact, err := store.ExactNameDomainExists(ctx, normalizedName, domainPart)
		if err != nil {
			return nil, err
		}
		if exact {
			out.DuplicateStatus = domain.DupExact
			return out, nil
		}
	}

	strongMatch, err := store.StrongNameMatchExists(ctx, normalizedName)
	if err != nil {
		return nil, err
	}
	if strongMatch {
		out.DuplicateStatus = domain.DupSuspectedDuplicate
	}
	return out, nil
}

func normalizeName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' || r == ',' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func domainFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.Contains(ref, "://") {
		if u, err := url.Parse(ref); err == nil {
			return strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
		}
	}
	return ""
}
