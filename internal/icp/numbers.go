package icp

import (
	"regexp"
	"strconv"
	"strings"
)

func normalizeNumericText(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), ",", "")
}

var (
	reOver  = regexp.MustCompile(`(?i)(?:over|>|~)\s*([0-9]{2,5}|1k)`)
	reRange = regexp.MustCompile(`(?i)([0-9]{2,5})\s*[–-]\s*([0-9]{2,5})`)
	reNum   = regexp.MustCompile(`([0-9]{2,5})`)
)

func parseOverValue(s string) (int, bool) {
	s = normalizeNumericText(s)
	m := reOver.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	if strings.EqualFold(m[1], "1k") {
		return 1000, true
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseRange(s string) (int, int, bool) {
	s = normalizeNumericText(s)
	m := reRange.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, 0, false
	}
	lo, err1 := strconv.Atoi(m[1])
	hi, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

func parseSingleNumber(s string) (int, bool) {
	s = normalizeNumericText(s)
	m := reNum.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// sizeClearlyBelowFifty implements ICP doc §7: companies with &lt;50 employees must be rejected.
func sizeClearlyBelowFifty(size *string) bool {
	if size == nil {
		return false
	}
	t := normalizeNumericText(strings.ToLower(strings.TrimSpace(*size)))
	if t == "" {
		return false
	}
	if _, hi, ok := parseRange(t); ok {
		return hi < 50
	}
	if n, ok := parseOverValue(t); ok {
		return n < 50
	}
	if n, ok := parseSingleNumber(t); ok {
		if n < 50 && !strings.Contains(t, "over") && !strings.Contains(t, ">") && !strings.Contains(t, "~") {
			return true
		}
	}
	return false
}
