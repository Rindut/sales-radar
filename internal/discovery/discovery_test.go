package discovery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"salesradar/internal/domain"
	"salesradar/internal/googlesearch"
)

func TestBatchLimit(t *testing.T) {
	tests := []struct {
		name string
		params domain.RunParams
		want int
	}{
		{
			name: "zero uses default 50",
			params: domain.RunParams{MaxLeadsThisRun: 0, SourceAllowlist: []domain.Source{domain.SourceGoogle}},
			want:   domain.MaxLeadsPerRunDefault,
		},
		{
			name: "negative uses default 50",
			params: domain.RunParams{MaxLeadsThisRun: -3, SourceAllowlist: []domain.Source{domain.SourceGoogle}},
			want:   domain.MaxLeadsPerRunDefault,
		},
		{
			name: "explicit 50 unchanged",
			params: domain.RunParams{MaxLeadsThisRun: 50, SourceAllowlist: []domain.Source{domain.SourceGoogle}},
			want:   50,
		},
		{
			name: "hard cap 100",
			params: domain.RunParams{MaxLeadsThisRun: 500, SourceAllowlist: []domain.Source{domain.SourceGoogle}},
			want:   domain.MaxLeadsPerRunCap,
		},
		{
			name: "exact cap",
			params: domain.RunParams{MaxLeadsThisRun: 100, SourceAllowlist: []domain.Source{domain.SourceGoogle}},
			want:   100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BatchLimit(tt.params); got != tt.want {
				t.Fatalf("BatchLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiscover_emptyAllowlist_nil(t *testing.T) {
	ctx := context.Background()
	out, err := Discover(ctx, domain.RunParams{
		MaxLeadsThisRun: 50,
		SourceAllowlist: nil,
	})
	if !errors.Is(err, ErrEmptySourceAllowlist) {
		t.Fatalf("err = %v, want ErrEmptySourceAllowlist", err)
	}
	if out != nil {
		t.Fatalf("expected nil slice, got len=%d", len(out))
	}
}

func TestDiscover_emptyAllowlist_emptySlice(t *testing.T) {
	ctx := context.Background()
	out, err := Discover(ctx, domain.RunParams{
		MaxLeadsThisRun: 50,
		SourceAllowlist: []domain.Source{},
	})
	if !errors.Is(err, ErrEmptySourceAllowlist) {
		t.Fatalf("err = %v, want ErrEmptySourceAllowlist", err)
	}
	if out != nil {
		t.Fatalf("expected nil slice, got len=%d", len(out))
	}
}

func TestDiscover_respectsBatchLimitAndSources(t *testing.T) {
	ctx := context.Background()
	allow := []domain.Source{domain.SourceLinkedIn, domain.SourceApollo}
	out, err := Discover(ctx, domain.RunParams{
		MaxLeadsThisRun: 7,
		SourceAllowlist: allow,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 7 {
		t.Fatalf("len = %d, want 7", len(out))
	}
	for i, c := range out {
		if c.DiscoveryID == "" || c.SourceRef == "" || c.UnstructuredContext == "" {
			t.Fatalf("candidate %d missing required fields: %+v", i, c)
		}
		want := allow[i%len(allow)]
		if c.Source != want {
			t.Fatalf("index %d: source = %s, want %s", i, c.Source, want)
		}
	}
}

func TestDiscover_hardCap(t *testing.T) {
	ctx := context.Background()
	out, err := Discover(ctx, domain.RunParams{
		MaxLeadsThisRun: 999,
		SourceAllowlist: []domain.Source{domain.SourceGoogle},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > domain.MaxLeadsPerRunCap {
		t.Fatalf("len = %d, want <= cap %d", len(out), domain.MaxLeadsPerRunCap)
	}
	if len(out) == 0 {
		t.Fatalf("len = %d, want at least 1", len(out))
	}
}

func TestDiscover_contextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Discover(ctx, domain.RunParams{
		MaxLeadsThisRun: 10,
		SourceAllowlist: []domain.Source{domain.SourceJobPortal},
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// mockFixtures must stay ICP-oriented, not a generic FAANG-style list.
func TestMockCandidates_notGenericBigCompanyList(t *testing.T) {
	banned := []string{
		"Apple", "Microsoft", "Amazon", "Google LLC", "Meta Platforms",
		"just a big company", "Fortune 500 list",
	}
	for i := 0; i < len(mockFixtures)*2; i++ {
		c := mockCandidateAt(i, domain.SourceLinkedIn)
		lower := strings.ToLower(c.UnstructuredContext)
		for _, b := range banned {
			if strings.Contains(lower, strings.ToLower(b)) {
				t.Fatalf("fixture sounds generic at idx %d: %q", i, c.UnstructuredContext)
			}
		}
	}
}

func TestSeedMinimumTarget(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{in: -1, want: 0},
		{in: 0, want: 0},
		{in: 10, want: 10},
		{in: 30, want: 30},
		{in: 80, want: 30},
	}
	for _, tt := range tests {
		if got := seedMinimumTarget(tt.in); got != tt.want {
			t.Fatalf("seedMinimumTarget(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestEmptyOutputReason(t *testing.T) {
	if got := emptyOutputReason(sourceWebsite); got == "" {
		t.Fatal("expected website empty-output reason")
	}
	if got := emptyOutputReason(sourceJob); got == "" {
		t.Fatal("expected job empty-output reason")
	}
	if got := emptyOutputReason(sourceSeed); got != "" {
		t.Fatalf("seed reason = %q, want empty", got)
	}
}

func TestBuildPhase1Sources_seedOnly_seedOff_includesGoogle(t *testing.T) {
	cfg := googlesearch.Config{APIKey: "k", CX: "cx"}
	tog := domain.DiscoverySourceToggles{Seed: false, Google: true}
	sources, skipped := buildPhase1SourcesWithSkips("seed_only", cfg, tog)
	if len(sources) != 1 {
		t.Fatalf("len(sources)=%d, want 1 (google only when seed disabled in seed_only mode)", len(sources))
	}
	_ = skipped
}

func TestBuildPhase1Sources_seedOnly_seedOn_googleNotAdded(t *testing.T) {
	cfg := googlesearch.Config{APIKey: "k", CX: "cx"}
	tog := domain.DiscoverySourceToggles{Seed: true, Google: true}
	sources, _ := buildPhase1SourcesWithSkips("seed_only", cfg, tog)
	if len(sources) != 1 {
		t.Fatalf("len(sources)=%d, want 1 (seed only)", len(sources))
	}
}

func TestDiscoverWithStatus_seedOff_googleNotConfigured_directoryFallback(t *testing.T) {
	ctx := context.Background()
	toggles := domain.DiscoverySourceToggles{
		Seed: false, Google: true, WebsiteCrawl: false, JobSignal: false,
		Apollo: false, LinkedIn: false,
	}
	res, err := DiscoverWithStatus(ctx, domain.RunParams{
		MaxLeadsThisRun: 15,
		SourceAllowlist: []domain.Source{domain.SourceGoogle},
		SourceToggles:   &toggles,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("expected curated directory fallback when Google is not configured and seed is off")
	}
}

func TestDiscoverWithStatus_multiSource_bothPrimaryOff_directoryFallback(t *testing.T) {
	ctx := context.Background()
	toggles := domain.DiscoverySourceToggles{
		Seed: false, Google: false, WebsiteCrawl: false, JobSignal: false,
		Apollo: false, LinkedIn: false,
	}
	res, err := DiscoverWithStatus(ctx, domain.RunParams{
		MaxLeadsThisRun: 10,
		SourceAllowlist: []domain.Source{domain.SourceGoogle},
		SourceToggles:   &toggles,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("expected directory fallback when all primary phase-1 sources are disabled")
	}
}
