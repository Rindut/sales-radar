package failpoint

import (
	"os"
	"strings"
)

const (
	enableEnv            = "SALESRADAR_ENABLE_FAILPOINTS"
	websiteCrawlEnv      = "SALESRADAR_FAILPOINT_WEBSITE_CRAWL"
	corePipelineErrorEnv = "SALESRADAR_FAILPOINT_CORE_PIPELINE"
)

type WebsiteCrawlMode string

const (
	WebsiteCrawlNone    WebsiteCrawlMode = ""
	WebsiteCrawlSuccess WebsiteCrawlMode = "success"
	WebsiteCrawlTimeout WebsiteCrawlMode = "timeout"
	WebsiteCrawlError   WebsiteCrawlMode = "error"
)

func Enabled() bool {
	return strings.TrimSpace(os.Getenv(enableEnv)) == "1"
}

func WebsiteCrawl() WebsiteCrawlMode {
	if !Enabled() {
		return WebsiteCrawlNone
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(websiteCrawlEnv))) {
	case string(WebsiteCrawlSuccess):
		return WebsiteCrawlSuccess
	case string(WebsiteCrawlTimeout):
		return WebsiteCrawlTimeout
	case string(WebsiteCrawlError):
		return WebsiteCrawlError
	default:
		return WebsiteCrawlNone
	}
}

func CorePipelineError() bool {
	if !Enabled() {
		return false
	}
	return strings.ToLower(strings.TrimSpace(os.Getenv(corePipelineErrorEnv))) == "error"
}
