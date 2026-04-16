package domain

// DiscoverySourceToggles controls which discovery integrations participate in a run.
// Nil on RunParams means all sources enabled (default).
type DiscoverySourceToggles struct {
	Google       bool
	Seed         bool
	WebsiteCrawl bool
	JobSignal    bool
	Apollo       bool
	LinkedIn     bool
}

// DefaultDiscoverySourceToggles returns all-on defaults (product default).
func DefaultDiscoverySourceToggles() DiscoverySourceToggles {
	return DiscoverySourceToggles{
		Google:       true,
		Seed:         true,
		WebsiteCrawl: true,
		JobSignal:    true,
		Apollo:       true,
		LinkedIn:     true,
	}
}

// SourceTogglesOrDefault returns t or defaults when t is nil.
func SourceTogglesOrDefault(t *DiscoverySourceToggles) DiscoverySourceToggles {
	if t == nil {
		return DefaultDiscoverySourceToggles()
	}
	return *t
}
