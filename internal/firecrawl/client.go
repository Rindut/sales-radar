package firecrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func postJSON(ctx context.Context, path string, body any) ([]byte, int, error) {
	key := APIKeyFromEnv()
	if key == "" {
		return nil, 0, fmt.Errorf("firecrawl: missing API key")
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	u := strings.TrimSuffix(apiBaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpTO := HTTPOneShotTimeout()
	t0 := time.Now()
	slog.Info("firecrawl: HTTP POST", "path", path)
	client := &http.Client{Timeout: httpTO}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("firecrawl: HTTP POST transport error", "path", path, "elapsed_ms", time.Since(t0).Milliseconds(), "err", err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	elapsed := time.Since(t0)
	if err != nil {
		slog.Warn("firecrawl: HTTP POST failed", "path", path, "status", resp.StatusCode, "elapsed_ms", elapsed.Milliseconds(), "err", err)
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("firecrawl: HTTP POST non-success", "path", path, "status", resp.StatusCode, "elapsed_ms", elapsed.Milliseconds())
	} else {
		slog.Info("firecrawl: HTTP POST ok", "path", path, "status", resp.StatusCode, "elapsed_ms", elapsed.Milliseconds(), "bytes", len(data))
	}
	return data, resp.StatusCode, nil
}

type mapResponse struct {
	Success bool              `json:"success"`
	Links   []string          `json:"links"`
	Data    mapResponseData   `json:"data"`
	Error   string            `json:"error"`
	Message string            `json:"message"`
}

type mapResponseData struct {
	Links []string `json:"links"`
}

func parseMapLinks(body []byte) ([]string, error) {
	var r mapResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	if !r.Success {
		msg := strings.TrimSpace(r.Error)
		if msg == "" {
			msg = strings.TrimSpace(r.Message)
		}
		if msg == "" {
			msg = "map failed"
		}
		return nil, fmt.Errorf("firecrawl map: %s", msg)
	}
	out := append([]string(nil), r.Links...)
	if len(out) == 0 && r.Data.Links != nil {
		out = append(out, r.Data.Links...)
	}
	return out, nil
}

type scrapeResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Markdown string `json:"markdown"`
	} `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func parseScrapeMarkdown(body []byte) (string, error) {
	var r scrapeResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if !r.Success {
		msg := strings.TrimSpace(r.Error)
		if msg == "" {
			msg = strings.TrimSpace(r.Message)
		}
		if msg == "" {
			msg = "scrape failed"
		}
		return "", fmt.Errorf("firecrawl scrape: %s", msg)
	}
	return strings.TrimSpace(r.Data.Markdown), nil
}
