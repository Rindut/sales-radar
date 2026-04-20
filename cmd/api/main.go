// Command api serves Sales Radar JSON API only (for api.sales.bawana.xyz or local dev).
// Auth: none by default; add middleware at internal/api or here when required.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"salesradar/internal/api"
	"salesradar/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "data/salesradar.db", "SQLite database file path")
	cors := flag.String("cors", "", "Comma-separated allowed browser Origins (Access-Control-Allow-Origin). Default: https://sales.bawana.xyz")
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

	handler := api.CORSMiddleware(api.ParseCORSAllowList(*cors), mux)

	log.Printf("Sales Radar API listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
