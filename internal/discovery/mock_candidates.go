package discovery

import (
	"fmt"

	"salesradar/internal/domain"
)

// mockFixtures are fictional, ICP-relevant company situations—not generic “large brand” lists.
// Lines prefixed with @company provide identifiable names for extraction + realism gates.
var mockFixtures = []string{
	"@company: Kawan Lama Group\n@industry: retail\nMulti-outlet retail; training and onboarding programs; estimated 10,000+ employees; Indonesia.",
	"@company: Sejahtera Bank Indonesia\n@industry: banking\nMulti-branch network; compliance and L&D; employee base over 1k; Indonesia.",
	"Regional banking group, BUMN-adjacent operations; multi-branch; L&D hiring; compliance and onboarding focus; estimated 1,100–1,400 employees; Indonesia.",
	"National grocery retail chain; multi-outlet; frontline-heavy; seasonal hiring spikes; structured onboarding needed; ~280 employees corporate + stores.",
	"Hospitality operator; hotel chain in tier-1 cities; large housekeeping/F&B staff; service standardization and training ops; size est. 120–180 at operator level.",
	"Department store retailer; fashion + supermarket mix; branch expansion announced; HR training coordinator roles open; scale mid-enterprise retail.",
	"Private bank; digital onboarding initiative; branch network; regulatory training requirements; employee base over 1k.",
	"Quick-service restaurant franchise network; outlet count growing; crew turnover; training systems for new store openings.",
	"Boutique hotel collection; soft brand; operational excellence program; workforce concentrated in guest-facing roles.",
	"Retail banking subsidiary; retail footprint; hiring learning specialists; emphasis on frontline certification.",
}

func mockCandidateAt(index int, src domain.Source) domain.RawCandidate {
	fixture := mockFixtures[index%len(mockFixtures)]
	host := fmt.Sprintf("mock-%d.example.com", index)
	return domain.RawCandidate{
		DiscoveryID:         fmt.Sprintf("mock-disc-%d", index),
		Source:              src,
		SourceRef:           fmt.Sprintf("https://%s/", host),
		UnstructuredContext: fixture,
		OfficialDomain:      host,
		ProspectTrace: domain.ProspectTrace{
			SourceTrace: []string{"mock_discovery"},
		},
	}
}
