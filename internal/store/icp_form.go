package store

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"salesradar/internal/domain"
)

const icpFormKVKey = "icp_form_settings"

// ICPFormSettings is persisted ICP configuration from the Settings UI.
type ICPFormSettings struct {
	// Version 2+ uses structured ICP sections; older payloads are migrated on read.
	Version int `json:"_v,omitempty"`

	TargetIndustries []string `json:"target_industries,omitempty"`
	// RegionFocus single value: "" | "any" | "idn" | "sea"
	RegionFocus string `json:"region_focus,omitempty"`
	SignalKeys  []string `json:"signal_keys,omitempty"`

	ExcludedIndustries []string `json:"excluded_industries,omitempty"`
	ExcludedSegments   []string `json:"excluded_segments,omitempty"`
	// ApplySub50 nil = default true (exclude very small companies when size is clear); false disables that rule.
	ApplySub50 *bool `json:"apply_sub50_rule,omitempty"`

	WeightIndustry string `json:"weight_industry,omitempty"`
	WeightSignal   string `json:"weight_signal,omitempty"`
	WeightSize     string `json:"weight_size,omitempty"`

	// MinEmployees / MaxEmployees: form values e.g. "100", "500"; empty = not set; max "0" or nolimit = no cap.
	MinEmployees string `json:"min_employees,omitempty"`
	MaxEmployees string `json:"max_employees,omitempty"`

	// Legacy v1 fields (migrated on read)
	TargetIndustry   string `json:"target_industry,omitempty"`
	CompanySize      string `json:"company_size,omitempty"`
	CountryRegion    string `json:"country_region,omitempty"`
	RequiredSignal   string `json:"required_signal,omitempty"`
	ExcludedIndustry string `json:"excluded_industry,omitempty"`
}

// GetICPFormSettings loads saved ICP form fields with normalization.
func GetICPFormSettings(db *sql.DB) (ICPFormSettings, error) {
	var raw sql.NullString
	err := db.QueryRow(`SELECT value FROM app_kv WHERE key = ?`, icpFormKVKey).Scan(&raw)
	if err == sql.ErrNoRows {
		s := ICPFormSettings{}
		NormalizeICPForm(&s)
		return s, nil
	}
	if err != nil {
		return ICPFormSettings{}, err
	}
	if !raw.Valid || raw.String == "" {
		s := ICPFormSettings{}
		NormalizeICPForm(&s)
		return s, nil
	}
	var s ICPFormSettings
	if err := json.Unmarshal([]byte(raw.String), &s); err != nil {
		return ICPFormSettings{}, err
	}
	migrateLegacyICPForm(&s)
	migrateICPFormVersion(&s)
	NormalizeICPForm(&s)
	return s, nil
}

func migrateLegacyICPForm(s *ICPFormSettings) {
	if len(s.TargetIndustries) == 0 && strings.TrimSpace(s.TargetIndustry) != "" {
		s.TargetIndustries = []string{legacyIndustryKey(s.TargetIndustry)}
	}
	if len(s.ExcludedIndustries) == 0 && strings.TrimSpace(s.ExcludedIndustry) != "" {
		s.ExcludedIndustries = []string{legacyIndustryKey(s.ExcludedIndustry)}
	}
}

func migrateICPFormVersion(s *ICPFormSettings) {
	if s.Version >= 2 {
		return
	}
	if len(s.ExcludedSegments) == 0 {
		s.ExcludedSegments = []string{"freelance_agency", "micro_enterprise"}
	}
	s.Version = 2
}

func legacyIndustryKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	switch {
	case strings.Contains(s, "bank"):
		return "banking"
	case strings.Contains(s, "retail"):
		return "retail"
	case strings.Contains(s, "hospit") || strings.Contains(s, "hotel"):
		return "hospitality"
	default:
		return s
	}
}

// defaultICPTargetIndustryIDs matches CatalogIndustries order (Sales Settings).
var defaultICPTargetIndustryIDs = []string{
	"banking", "retail", "hospitality", "manufacturing", "healthcare",
	"education", "technology", "logistics", "fmcg", "others",
}

// NormalizeICPForm applies defaults (empty target list = all catalog industries).
func NormalizeICPForm(s *ICPFormSettings) {
	if len(s.TargetIndustries) == 0 {
		s.TargetIndustries = append([]string(nil), defaultICPTargetIndustryIDs...)
	}
	if s.WeightIndustry == "" {
		s.WeightIndustry = "medium"
	}
	if s.WeightSignal == "" {
		s.WeightSignal = "medium"
	}
	if s.WeightSize == "" {
		s.WeightSize = "medium"
	}
	// First load: zero value false — treat as on only if JSON had the key; handled by unmarshaling.
	// For legacy JSON without apply_sub50_rule, default true.
}

// SetICPFormSettings persists ICP form fields.
func SetICPFormSettings(db *sql.DB, s ICPFormSettings) error {
	NormalizeICPForm(&s)
	s.Version = 2
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO app_kv (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		icpFormKVKey, string(b))
	return err
}

// ToICPRuntime maps stored form values to pipeline runtime settings.
func (s ICPFormSettings) ToICPRuntime() *domain.ICPRuntimeSettings {
	cp := s
	NormalizeICPForm(&cp)

	sub50 := true
	if cp.ApplySub50 != nil {
		sub50 = *cp.ApplySub50
	}
	out := &domain.ICPRuntimeSettings{
		TargetIndustryIDs:   dedupeTrimLower(cp.TargetIndustries),
		ExcludedIndustryIDs: dedupeTrimLower(cp.ExcludedIndustries),
		TargetBuckets:       parseBuckets(cp.TargetIndustries),
		ExcludedBuckets:     parseBuckets(cp.ExcludedIndustries),
		RegionFocus:         strings.TrimSpace(cp.RegionFocus),
		SignalKeys:          dedupeTrim(cp.SignalKeys),
		ExcludedSegmentKeys: dedupeTrim(cp.ExcludedSegments),
		ApplySub50Rule:      sub50,
		MinEmployees:        parseEmployeesBound(cp.MinEmployees),
		MaxEmployees:        parseEmployeesBound(cp.MaxEmployees),
		WeightIndustry:      normalizeWeight(cp.WeightIndustry),
		WeightSignal:        normalizeWeight(cp.WeightSignal),
		WeightSize:          normalizeWeight(cp.WeightSize),
	}
	if out.RegionFocus == "any" {
		out.RegionFocus = ""
	}
	// Default sub-50 on when unset in JSON historical rows: ApplySub50Rule false with empty new form — first save sets explicit.
	// If all weights missing, Normalize already set medium.
	return out
}

func parseBuckets(ids []string) []domain.ICPIndustryBucket {
	var out []domain.ICPIndustryBucket
	seen := map[domain.ICPIndustryBucket]struct{}{}
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		var b domain.ICPIndustryBucket
		switch id {
		case "banking":
			b = domain.BucketBanking
		case "retail":
			b = domain.BucketRetail
		case "hospitality":
			b = domain.BucketHospitality
		default:
			continue
		}
		if _, ok := seen[b]; ok {
			continue
		}
		seen[b] = struct{}{}
		out = append(out, b)
	}
	return out
}

func normalizeWeight(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "medium"
	}
}

func dedupeTrim(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func dedupeTrimLower(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func parseEmployeesBound(raw string) int {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || raw == "nolimit" || raw == "no_limit" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
