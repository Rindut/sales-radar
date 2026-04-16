// Package companycheck applies deterministic rules for identifiable company names and domains.
package companycheck

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var genericSubstrings = []string{
	"retail company",
	"hospitality operator",
	"department store retailer",
	"national grocery",
	"regional banking group",
	"grocery retail chain",
	"quick-service restaurant",
	"boutique hotel collection",
	"retail banking subsidiary",
	"private bank;",
	"private bank,",
	"private bank ",
	"franchise network",
	"hotel chain in",
	"department store",
	"restaurant franchise",
	"banking group,",
	"banking group ",
	"retail chain;",
	"national grocery retail",
	"abstract category",
	"sector player",
	"industry leader",
	"market participant",
}

// categoryVocabulary: words that are fine inside a real name (e.g. "Kawan Lama Group")
// but if EVERY substantive token is only from this set, the string is an abstract category, not a brand.
var categoryVocabulary = map[string]struct{}{
	"national": {}, "regional": {}, "private": {}, "public": {}, "retail": {}, "wholesale": {},
	"hospitality": {}, "banking": {}, "bank": {}, "banks": {}, "department": {}, "store": {}, "stores": {},
	"grocery": {}, "supermarket": {}, "supermarkets": {}, "quick": {}, "service": {}, "services": {},
	"restaurant": {}, "restaurants": {}, "franchise": {}, "network": {}, "networks": {},
	"operator": {}, "operators": {}, "chain": {}, "chains": {}, "group": {}, "groups": {},
	"subsidiary": {}, "subsidiaries": {}, "collection": {}, "collections": {}, "boutique": {},
	"hotel": {}, "hotels": {}, "outlet": {}, "outlets": {}, "company": {}, "companies": {},
	"business": {}, "businesses": {}, "sector": {}, "integrated": {}, "multi": {}, "branch": {},
	"branches": {}, "corporate": {}, "enterprise": {}, "enterprises": {}, "mid": {}, "large": {},
	"small": {}, "scale": {}, "food": {}, "beverage": {}, "f&b": {}, "retailer": {}, "retailers": {},
	"fashion": {}, "mixed": {}, "soft": {}, "brand": {}, "brands": {}, "tier": {}, "tier-1": {},
	"cities": {}, "operations": {}, "operation": {}, "operational": {}, "excellence": {},
	"program": {}, "programs": {}, "workforce": {}, "staff": {}, "guest": {}, "facing": {},
	"housekeeping": {}, "bumn": {}, "adjacent": {}, "digital": {}, "initiative": {}, "initiatives": {},
	"footprint": {}, "certification": {}, "standardization": {}, "level": {},
}

var reTokenize = regexp.MustCompile(`[^a-zA-Z0-9&]+`)

// IsGenericCompanyName is true for sector labels, not identifiable brands.
func IsGenericCompanyName(name string) bool {
	t := strings.TrimSpace(strings.ToLower(name))
	if t == "" {
		return true
	}
	for _, g := range genericSubstrings {
		if strings.Contains(t, g) {
			return true
		}
	}
	if matched, _ := regexp.MatchString(`^(national|regional|private|quick-service|department store|retail|hospitality)\s+(bank|operator|retailer|chain|group|network|subsidiary)\s*$`, t); matched {
		return true
	}
	return false
}

// isAbstractCategoryOnly is true when all substantive words are generic sector vocabulary (no brand token).
func isAbstractCategoryOnly(name string) bool {
	raw := reTokenize.Split(strings.TrimSpace(name), -1)
	var words []string
	for _, w := range raw {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return true
	}
	for _, w := range words {
		if len(w) <= 2 {
			continue
		}
		if _, onlySector := categoryVocabulary[w]; !onlySector {
			return false
		}
	}
	return true
}

// looksLikeRealOrganizationName: structural signals for a real org / brand (not a prose description).
func looksLikeRealOrganizationName(name string) bool {
	t := strings.TrimSpace(name)
	if len(t) < 3 {
		return false
	}
	lower := strings.ToLower(t)
	// Indonesian legal form
	if regexp.MustCompile(`(?i)\bPT\s+[A-Za-z]`).MatchString(t) {
		return true
	}
	if regexp.MustCompile(`[0-9]`).MatchString(t) {
		return true
	}
	// All-lowercase long phrase → likely a description, not a extracted brand
	if t == lower && len(t) > 12 && strings.Contains(t, " ") {
		return false
	}
	hasUpper := false
	hasLower := false
	for _, r := range t {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	if hasUpper && hasLower {
		return true
	}
	// Short all-caps ticker-style (e.g. ABC, BCA)
	if m, _ := regexp.MatchString(`^[A-Z]{2,8}$`, strings.TrimSpace(t)); m {
		return true
	}
	// Single proper token (unusual but allow short acronyms)
	if len(t) >= 4 && hasUpper && !strings.Contains(t, " ") {
		return true
	}
	return false
}

// IsIdentifiableCompany is true only for names that look like real organizations or brands.
// If this returns false, the lead should be discarded.
func IsIdentifiableCompany(name string) bool {
	t := strings.TrimSpace(name)
	if t == "" {
		return false
	}
	if IsGenericCompanyName(t) {
		return false
	}
	if isAbstractCategoryOnly(t) {
		return false
	}
	if !looksLikeRealOrganizationName(t) {
		return false
	}
	return true
}

// LooksLikeRealEntityName is kept for backward compatibility; use IsIdentifiableCompany for gates.
func LooksLikeRealEntityName(name string) bool {
	return IsIdentifiableCompany(name)
}

// BlockedNonCompanyDomains lists hosts that must never be treated as the lead's company-owned website.
// Policy: reject search, professional network, and data-provider origins; only real employer domains pass.
var BlockedNonCompanyDomains = []string{
	// Required policy
	"google.com",
	"linkedin.com",
	"apollo.io",
	// Other non-employer / aggregator hosts
	"google.co.id",
	"bing.com",
	"duckduckgo.com",
	"facebook.com",
	"instagram.com",
	"twitter.com",
	"x.com",
	"youtube.com",
	"tiktok.com",
}

// NormalizeHost lowercases and strips a leading www. label.
func NormalizeHost(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))
	h = strings.TrimPrefix(h, "www.")
	return h
}

// IsBlockedNonCompanyDomain is true when the host is not a company-owned corporate site (use for website fields).
func IsBlockedNonCompanyDomain(host string) bool {
	h := NormalizeHost(host)
	if h == "" {
		return true
	}
	for _, b := range BlockedNonCompanyDomains {
		if h == b || strings.HasSuffix(h, "."+b) {
			return true
		}
	}
	return false
}

// IsAggregatorDomain is an alias for IsBlockedNonCompanyDomain (legacy name).
func IsAggregatorDomain(host string) bool {
	return IsBlockedNonCompanyDomain(host)
}

// SanitizeCompanyWebsiteDomain returns the host or empty if it is not an allowable company domain.
func SanitizeCompanyWebsiteDomain(host string) string {
	h := NormalizeHost(host)
	if h == "" || IsBlockedNonCompanyDomain(h) {
		return ""
	}
	return h
}

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

var semanticStop = map[string]struct{}{
	"the": {}, "and": {}, "of": {}, "for": {}, "a": {}, "an": {}, "in": {}, "at": {}, "or": {},
}

// MergeDedupKey builds an order-independent signature so "Boutique Hotel Collection" and
// "Collection Boutique Hotel" collapse to one bucket; punctuation/spacing differences are ignored.
func MergeDedupKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = reNonAlnum.ReplaceAllString(s, " ")
	fields := strings.Fields(s)
	var tok []string
	for _, w := range fields {
		if _, skip := semanticStop[w]; skip {
			continue
		}
		if len(w) < 2 {
			continue
		}
		tok = append(tok, w)
	}
	if len(tok) == 0 {
		return ""
	}
	sort.Strings(tok)
	return strings.Join(tok, " ")
}

// SemanticKey is an alias for MergeDedupKey (legacy name).
func SemanticKey(name string) string {
	return MergeDedupKey(name)
}
