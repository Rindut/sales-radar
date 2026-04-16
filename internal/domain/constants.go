package domain

// Run and ICP limits shared by the pipeline (locked product rules).
const (
	// Default candidate pool size per run (v2): collect more candidates, keep strict gates.
	MaxLeadsPerRunDefault = 40
	MaxLeadsPerRunCap     = 100
	MaxICPReasons         = 3
	MaxOdooPushRetries    = 2 // retries after the first attempt; see ops spec
)
