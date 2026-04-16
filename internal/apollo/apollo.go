// Package apollo provides optional organization enrichment by domain (Apollo.io API).
// Set SALESRADAR_APOLLO_API_KEY to enable. No network calls when unset.
package apollo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"salesradar/internal/companycheck"
)

// APIKeyFromEnv returns SALESRADAR_APOLLO_API_KEY.
func APIKeyFromEnv() string {
	return strings.TrimSpace(os.Getenv("SALESRADAR_APOLLO_API_KEY"))
}

// Org is a minimal org payload used for enrichment.
type Org struct {
	Name                  string
	LinkedInURL           string
	EstimatedNumEmployees int
}

type searchResponse struct {
	Organizations []struct {
		Name                  string `json:"name"`
		PrimaryDomain         string `json:"primary_domain"`
		LinkedinURL           string `json:"linkedin_url"`
		EstimatedNumEmployees int    `json:"estimated_num_employees"`
	} `json:"organizations"`
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
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.apollo.io/v1/organizations/search", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apollo: HTTP %d", resp.StatusCode)
	}
	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	if len(sr.Organizations) == 0 {
		return nil, nil
	}
	o := sr.Organizations[0]
	return &Org{
		Name:                  strings.TrimSpace(o.Name),
		LinkedInURL:           strings.TrimSpace(o.LinkedinURL),
		EstimatedNumEmployees: o.EstimatedNumEmployees,
	}, nil
}
