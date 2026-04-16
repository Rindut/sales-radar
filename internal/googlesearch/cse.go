// Package googlesearch calls the Google Custom Search JSON API (Programmable Search Engine).
// Requires SALESRADAR_GOOGLE_API_KEY and SALESRADAR_GOOGLE_CX (search engine ID).
package googlesearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const endpoint = "https://www.googleapis.com/customsearch/v1"

// Config holds Google Custom Search credentials.
type Config struct {
	APIKey string
	CX     string
}

// ConfigFromEnv reads SALESRADAR_GOOGLE_API_KEY and SALESRADAR_GOOGLE_CX.
func ConfigFromEnv() Config {
	return Config{
		APIKey: strings.TrimSpace(os.Getenv("SALESRADAR_GOOGLE_API_KEY")),
		CX:     strings.TrimSpace(os.Getenv("SALESRADAR_GOOGLE_CX")),
	}
}

// Configured is true when live Google search can run.
func (c Config) Configured() bool {
	return c.APIKey != "" && c.CX != ""
}

// Result is one organic search hit.
type Result struct {
	Title   string
	Link    string
	Snippet string
}

type apiResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Search runs one or more CSE requests and returns up to maxResults (API supports max 10 per request).
func (c Config) Search(ctx context.Context, query string, maxResults int) ([]Result, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("googlesearch: missing API key or CX")
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 20 {
		maxResults = 20
	}
	client := &http.Client{Timeout: 45 * time.Second}
	var out []Result
	for start := 1; len(out) < maxResults; start += 10 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		perRequest := maxResults - len(out)
		if perRequest > 10 {
			perRequest = 10
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		qv := u.Query()
		qv.Set("key", c.APIKey)
		qv.Set("cx", c.CX)
		qv.Set("q", query)
		qv.Set("num", strconv.Itoa(perRequest))
		qv.Set("start", strconv.Itoa(start))
		u.RawQuery = qv.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var dec apiResponse
		err = json.NewDecoder(resp.Body).Decode(&dec)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if dec.Error != nil {
			return nil, fmt.Errorf("googlesearch: API error %d: %s", dec.Error.Code, dec.Error.Message)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("googlesearch: HTTP %d", resp.StatusCode)
		}
		if len(dec.Items) == 0 {
			break
		}
		for _, it := range dec.Items {
			out = append(out, Result{
				Title:   strings.TrimSpace(it.Title),
				Link:    strings.TrimSpace(it.Link),
				Snippet: strings.TrimSpace(it.Snippet),
			})
			if len(out) >= maxResults {
				break
			}
		}
		if len(dec.Items) < perRequest {
			break
		}
	}
	return out, nil
}
