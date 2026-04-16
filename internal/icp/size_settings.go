package icp

import (
	"strings"
)

// settingsEmployeeBounds returns a rough (low, high) from headcount text; high may equal low for a point estimate.
func settingsEmployeeBounds(size *string) (lo, hi int, ok bool) {
	if size == nil {
		return 0, 0, false
	}
	text := normalizeNumericText(strings.ToLower(strings.TrimSpace(*size)))
	if text == "" {
		return 0, 0, false
	}
	if a, b, ok2 := parseRange(text); ok2 {
		return a, b, true
	}
	if n, ok2 := parseOverValue(text); ok2 {
		return n, n, true
	}
	if n, ok2 := parseSingleNumber(text); ok2 {
		return n, n, true
	}
	return 0, 0, false
}

func disqualifiersForConfiguredSize(size *string, minEmployees, maxEmployees int) []string {
	if size == nil || (minEmployees <= 0 && maxEmployees <= 0) {
		return nil
	}
	raw := strings.ToLower(strings.TrimSpace(*size))
	text := normalizeNumericText(raw)
	if text == "" {
		return nil
	}
	lo, hi, ok := settingsEmployeeBounds(size)
	if !ok {
		return nil
	}
	overish := strings.Contains(raw, "over") || strings.Contains(raw, ">") || strings.Contains(raw, "~")

	if minEmployees > 0 {
		if hi > 0 && hi < minEmployees {
			return []string{"below minimum company size"}
		}
		if hi == 0 && lo > 0 && !overish && lo < minEmployees {
			return []string{"below minimum company size"}
		}
	}

	if maxEmployees > 0 {
		if lo > maxEmployees {
			return []string{"above maximum company size"}
		}
		if hi > 0 && lo == hi && lo > maxEmployees {
			return []string{"above maximum company size"}
		}
		if overish {
			if n, ok2 := parseOverValue(text); ok2 && n > maxEmployees {
				return []string{"above maximum company size"}
			}
		}
	}

	return nil
}
