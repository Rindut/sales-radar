// Command api serves Sales Radar JSON API only (for api.sales.bawana.xyz or local dev).
// Auth: none by default; add middleware at internal/api or here when required.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"salesradar/internal/api"
	"salesradar/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "data/salesradar.db", "SQLite database file path")
	cors := flag.String("cors", "", "If set, Access-Control-Allow-Origin value (e.g. https://sales.bawana.xyz). Use * for dev only.")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	dsn := "file:" + *dbPath + "?_pragma=busy_timeout(5000)"
	db, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	api.Register(mux, db)

	var handler http.Handler = mux
	if strings.TrimSpace(*cors) != "" {
		handler = corsMiddleware(*cors, mux)
	}

	log.Printf("Sales Radar API listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

func corsMiddleware(allowOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
