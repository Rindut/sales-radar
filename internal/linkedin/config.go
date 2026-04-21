package linkedin

import (
	"os"
	"strings"
)

// APIKeyFromEnv returns LINKEDIN_API_KEY when present.
func APIKeyFromEnv() string {
	return strings.TrimSpace(os.Getenv("LINKEDIN_API_KEY"))
}

// Configured reports whether a LinkedIn credential is configured.
func Configured() bool {
	return APIKeyFromEnv() != ""
}
