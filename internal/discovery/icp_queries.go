package discovery

import "fmt"

// icpSearchQueries returns deterministic Google discovery queries using product-approved templates.
func icpSearchQueries() []string {
	industries := []string{"retail", "fmcg", "grocery", "department store", "banking", "hospitality", "hotel"}
	var out []string
	for _, industry := range industries {
		// Template 1: list of companies.
		out = append(out, fmt.Sprintf("list of %s companies indonesia", industry))
		// Template 2: LinkedIn company pages.
		out = append(out, fmt.Sprintf("%s companies indonesia site:linkedin.com/company", industry))
		// Template 3: official website bias.
		out = append(out, fmt.Sprintf("%s companies indonesia official website", industry))
	}
	return out
}
