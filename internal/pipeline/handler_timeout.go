package pipeline

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// HandlerRequestTimeout is the maximum wall time for POST /pipeline/run (Generate Leads).
// Default 45 minutes; prevents the HTTP handler from running indefinitely.
func HandlerRequestTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	sec := 2700 // 45m
	if s := strings.TrimSpace(os.Getenv("SALESRADAR_PIPELINE_HANDLER_TIMEOUT_SEC")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 60 {
			sec = n
		}
	}
	return context.WithTimeout(parent, time.Duration(sec)*time.Second)
}
