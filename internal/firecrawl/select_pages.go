package firecrawl

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

var lowValuePathTokens = []string{
	"/privacy", "/terms", "/legal", "/cookie", "/cookies", "/login", "/signin", "/signup",
	"/register", "/cart", "/checkout", "/account", "/wp-admin", "/admin",
}

var highValuePathTokens = []string{
	"/about", "/company", "/who-we-are", "/careers", "/jobs", "/vacancies", "/hiring",
	"/academy", "/training", "/learning", "/education", "/news", "/press", "/media", "/blog",
}

func isLowValueURL(u *url.URL) bool {
	p := strings.ToLower(path.Clean(u.Path))
	if p == "" {
		return false
	}
	for _, tok := range lowValuePathTokens {
		if strings.Contains(p, tok) {
			return true
		}
	}
	// Generic support/help hubs (heuristic).
	if strings.Contains(p, "/support") && !strings.Contains(p, "/training") {
		return true
	}
	if p == "/help" || strings.HasPrefix(p, "/help/") {
		return true
	}
	return false
}

func urlScore(u *url.URL) int {
	p := strings.ToLower(path.Clean(u.Path))
	sc := 1
	if p == "" || p == "/" {
		sc += 50
	}
	for _, tok := range highValuePathTokens {
		if strings.Contains(p, tok) {
			sc += 25
		}
	}
	if strings.Count(p, "/") <= 2 {
		sc += 3
	}
	return sc
}

// sameSite returns true if other is same registrable host as root (ignore www).
func sameSite(root, other *url.URL) bool {
	if root == nil || other == nil {
		return false
	}
	a := strings.TrimPrefix(strings.ToLower(root.Hostname()), "www.")
	b := strings.TrimPrefix(strings.ToLower(other.Hostname()), "www.")
	return a == b && root.Scheme == other.Scheme
}

// pickURLs merges map links with seed paths, scores, dedupes, and returns up to max URLs.
func pickURLs(root string, mapLinks []string, max int) []string {
	base, err := url.Parse(root)
	if err != nil || base.Host == "" {
		return nil
	}
	seen := map[string]struct{}{}
	type scored struct {
		u string
		s int
	}
	var pool []scored

	add := func(raw string, bonus int) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return
		}
		if !sameSite(base, u) {
			return
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			u.Scheme = "https"
		}
		if isLowValueURL(u) {
			return
		}
		key := u.Scheme + "://" + strings.ToLower(u.Hostname()) + path.Clean(u.Path)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		pool = append(pool, scored{u: u.String(), s: urlScore(u) + bonus})
	}

	add(base.String(), 0)

	// Likely high-signal paths even if map missed them.
	seedPaths := []string{
		"/", "/about", "/company", "/careers", "/jobs", "/news", "/press", "/academy", "/training",
	}
	for _, p := range seedPaths {
		u := *base
		u.Path = p
		add(u.String(), 2)
	}

	for _, lk := range mapLinks {
		add(lk, 0)
	}

	sort.Slice(pool, func(i, j int) bool {
		if pool[i].s != pool[j].s {
			return pool[i].s > pool[j].s
		}
		return pool[i].u < pool[j].u
	})
	var out []string
	for _, x := range pool {
		if len(out) >= max {
			break
		}
		out = append(out, x.u)
	}
	return out
}
