package api

import (
	"net/http"
	"strings"
)

// DefaultCORSAllowList is used when -cors / CORS allowlist is empty.
// Includes local Next.js dev servers so browser fetch from localhost:3000 works without extra flags.
const DefaultCORSAllowList = "https://sales.bawana.xyz,http://localhost:3000,http://127.0.0.1:3000"

const corsAllowMethods = "GET, POST, PUT, OPTIONS"

const corsAllowHeaders = "Content-Type, Authorization"

// ParseCORSAllowList splits a comma-separated list of allowed browser Origins.
// Empty or whitespace-only s defaults to DefaultCORSAllowList (also comma-separated).
func ParseCORSAllowList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = DefaultCORSAllowList
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
		return []string{"https://sales.bawana.xyz"}
	}
	return out
}

func corsEchoOrigin(r *http.Request, allowed []string) string {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return ""
	}
	for _, o := range allowed {
		o = strings.TrimSpace(o)
		if o == "*" {
			return "*"
		}
		if origin == o {
			return origin
		}
	}
	return ""
}

// CORSMiddleware applies CORS to all routes. Allowed origins get full CORS headers
// on every response (including PUT), not only on OPTIONS — browsers need
// Access-Control-Allow-Origin on the actual request response.
//
// OPTIONS preflight is answered with 200 OK and does not invoke next, so
// method-specific mux routes never return 405 for OPTIONS.
func CORSMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo := corsEchoOrigin(r, allowedOrigins)
		if echo != "" {
			w.Header().Set("Access-Control-Allow-Origin", echo)
			w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
			if echo != "*" {
				w.Header().Add("Vary", "Origin")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
