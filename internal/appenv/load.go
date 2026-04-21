package appenv

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Load initializes environment variables from .env when present and logs Apollo readiness.
func Load() {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env load skipped: %v", err)
	} else {
		log.Printf(".env loaded")
	}
	apiKey := strings.TrimSpace(os.Getenv("APOLLO_API_KEY"))
	log.Printf("Apollo API key loaded: %t", apiKey != "")
}
