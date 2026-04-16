package domain

// ICPRuntimeSettings is the normalized ICP configuration used during pipeline evaluation
// (loaded from Settings → store.ICPFormSettings).
type ICPRuntimeSettings struct {
	// TargetIndustryIDs / ExcludedIndustryIDs are catalog slugs (e.g. banking, manufacturing).
	TargetIndustryIDs   []string
	ExcludedIndustryIDs []string
	// TargetBuckets / ExcludedBuckets: core three sectors only; used for legacy size thresholds.
	TargetBuckets       []ICPIndustryBucket
	ExcludedBuckets     []ICPIndustryBucket
	RegionFocus         string
	SignalKeys          []string
	ExcludedSegmentKeys []string
	ApplySub50Rule      bool
	MinEmployees        int
	MaxEmployees        int
	WeightIndustry      string
	WeightSignal        string
	WeightSize          string
}

// DefaultICPRuntimeSettings returns baseline behavior when no Settings are passed (e.g. CLI).
// Empty TargetIndustryIDs means no industry allow-list (all industries allowed).
func DefaultICPRuntimeSettings() *ICPRuntimeSettings {
	return &ICPRuntimeSettings{
		TargetIndustryIDs:   nil,
		ExcludedIndustryIDs: nil,
		TargetBuckets:       []ICPIndustryBucket{BucketBanking, BucketRetail, BucketHospitality},
		ExcludedBuckets:     nil,
		RegionFocus:         "",
		SignalKeys:          nil,
		ExcludedSegmentKeys: nil,
		ApplySub50Rule:      true,
		MinEmployees:        0,
		MaxEmployees:        0,
		WeightIndustry:      "medium",
		WeightSignal:        "medium",
		WeightSize:          "medium",
	}
}
