package api

import (
	"net/http"
	"strings"
)

// DefaultCORSAllowList is used when -cors / CORS allowlist is empty.
const DefaultCORSAllowList = "https://sales.bawana.xyz"

const corsAllowMethods = "GET, POST, OPTIONS"

const corsAllowHeaders = "Content-Type, Authorization"

// ParseCORSAllowList splits a comma-separated list of allowed browser Origins.
// Empty or whitespace-only s defaults to DefaultCORSAllowList.
func ParseCORSAllowList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{DefaultCORSAllowList}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{DefaultCORSAllowList}
	}
	return out
}

func corsEchoOrigin(r *http.Request, allowed []string) string {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return ""
	}
	for _, o := range allowed {
		if origin == o {
			return origin
		}
	}
	return ""
}

// CORSMiddleware applies CORS to all routes. OPTIONS preflight is answered with
// 200 OK and does not invoke next, so method-specific mux routes never return 405
// for OPTIONS.
func CORSMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo := corsEchoOrigin(r, allowedOrigins)
		if echo != "" {
			w.Header().Set("Access-Control-Allow-Origin", echo)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if echo != "" {
				w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
