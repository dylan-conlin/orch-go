package findingdedup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFindings_InvestigationFormat(t *testing.T) {
	content := `# Investigation: Test

## Findings

### Finding 1: Cache invalidation is broken

**Evidence:** The cache TTL is set to 0, causing stale reads.
**Source:** pkg/cache/cache.go:42
**Significance:** All downstream consumers see stale data.

### Finding 2: Retry logic has no backoff

**Evidence:** Retries happen immediately with no delay.
**Source:** pkg/http/client.go:88
**Significance:** This causes thundering herd on failures.
`
	findings := ExtractFindings("test-inv.md", content)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if findings[0].Title != "Cache invalidation is broken" {
		t.Errorf("finding 0 title = %q", findings[0].Title)
	}
	if findings[0].SourceFile != "test-inv.md" {
		t.Errorf("finding 0 source = %q", findings[0].SourceFile)
	}
	if findings[1].Title != "Retry logic has no backoff" {
		t.Errorf("finding 1 title = %q", findings[1].Title)
	}

	// Body should contain evidence text
	if !containsStr(findings[0].Body, "cache TTL is set to 0") {
		t.Errorf("finding 0 body missing evidence: %q", findings[0].Body)
	}
}

func TestExtractFindings_SynthesisKnowledge(t *testing.T) {
	content := `# Session Synthesis

## TLDR
Did some work.

## Knowledge (What Was Learned)

### Constraints Discovered
- Daemon polls every 30s, not configurable
- OpenCode SSE drops events under load

### Externalized via kb quick
- kb quick "daemon poll interval"
`
	findings := ExtractFindings("SYNTHESIS.md", content)
	if len(findings) < 1 {
		t.Fatalf("expected at least 1 finding from knowledge section, got %d", len(findings))
	}
}

func TestExtractFindings_SynthesisEvidence(t *testing.T) {
	content := `# Session Synthesis

## Evidence (What Was Observed)
- Token refresh fails silently when account is locked
- SSE reconnection takes 15s average
- Agents stall when OpenCode server restarts

## Knowledge
Nothing new.
`
	findings := ExtractFindings("SYNTHESIS.md", content)
	if len(findings) < 1 {
		t.Fatalf("expected findings from evidence section, got %d", len(findings))
	}
}

func TestTokenize(t *testing.T) {
	text := "The cache TTL is set to 0, causing stale reads."
	tokens := tokenize(text)

	// Should lowercase and remove stopwords
	for _, tok := range tokens {
		if tok == "the" || tok == "is" || tok == "to" {
			t.Errorf("stopword %q not removed", tok)
		}
	}
	// Should contain content words
	hasCache := false
	for _, tok := range tokens {
		if tok == "cache" {
			hasCache = true
		}
	}
	if !hasCache {
		t.Error("expected 'cache' in tokens")
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		min  float64
		max  float64
	}{
		{
			name: "identical",
			a:    "the cache invalidation is broken",
			b:    "the cache invalidation is broken",
			min:  0.99,
			max:  1.01,
		},
		{
			name: "similar rewording",
			a:    "cache invalidation does not work correctly causing stale data",
			b:    "cache invalidation is broken leading to stale data reads",
			min:  0.3,
			max:  0.9,
		},
		{
			name: "completely different",
			a:    "the database migration failed on production",
			b:    "frontend button color needs updating to blue",
			min:  0.0,
			max:  0.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokA := tokenize(tt.a)
			tokB := tokenize(tt.b)
			sim := jaccardSimilarity(tokA, tokB)
			if sim < tt.min || sim > tt.max {
				t.Errorf("similarity = %.3f, want [%.2f, %.2f]", sim, tt.min, tt.max)
			}
		})
	}
}

func TestFindDuplicateClusters(t *testing.T) {
	findings := []Finding{
		{Title: "A", Body: "cache invalidation causes stale data in downstream consumers", SourceFile: "inv-1.md"},
		{Title: "B", Body: "stale cache data propagates to downstream services causing errors", SourceFile: "inv-2.md"},
		{Title: "C", Body: "cache staleness affects downstream data consumers", SourceFile: "inv-3.md"},
		{Title: "D", Body: "the database migration script has incorrect column types", SourceFile: "inv-4.md"},
	}

	d := NewDetector()
	d.MinClusterSize = 3
	d.Threshold = 0.25
	clusters := d.FindClusters(findings)

	// Should cluster A, B, C together (cache/stale/downstream theme)
	// D should not be in any cluster
	foundCacheCluster := false
	for _, c := range clusters {
		if len(c.Findings) >= 3 {
			foundCacheCluster = true
			// Verify D is not in this cluster
			for _, f := range c.Findings {
				if f.Title == "D" {
					t.Error("finding D should not be in the cache cluster")
				}
			}
		}
	}
	if !foundCacheCluster {
		t.Errorf("expected a cluster with 3+ findings about cache staleness, got %d clusters: %v", len(clusters), clusters)
	}
}

func TestFindDuplicateClusters_BelowMinSize(t *testing.T) {
	findings := []Finding{
		{Title: "A", Body: "cache invalidation broken", SourceFile: "inv-1.md"},
		{Title: "B", Body: "cache invalidation broken", SourceFile: "inv-2.md"},
	}

	d := NewDetector()
	d.MinClusterSize = 3
	clusters := d.FindClusters(findings)

	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters (below MinClusterSize=3), got %d", len(clusters))
	}
}

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	// Create 4 investigation files with 3 having similar findings
	writeFile(t, dir, "inv-1.md", `# Investigation: A

## Findings

### Finding 1: Agents stall when server restarts

**Evidence:** OpenCode server restart kills SSE connections, agents lose state.
**Source:** logs
**Significance:** Agent reliability degrades during infrastructure events.
`)

	writeFile(t, dir, "inv-2.md", `# Investigation: B

## Findings

### Finding 1: Server restart causes agent failures

**Evidence:** When OpenCode restarts, SSE streams drop and agents become unresponsive.
**Source:** monitoring
**Significance:** Infrastructure instability propagates to agent layer.
`)

	writeFile(t, dir, "inv-3.md", `# Investigation: C

## Findings

### Finding 1: Agent stalls during server infrastructure events

**Evidence:** SSE connection drops when server restarts cause agents to stall and lose context.
**Source:** incident report
**Significance:** Server restarts create cascading agent failures.
`)

	writeFile(t, dir, "inv-4.md", `# Investigation: D

## Findings

### Finding 1: Dashboard CSS is misaligned on mobile

**Evidence:** Button overlap on narrow viewports.
**Source:** screenshot
**Significance:** Minor UI issue.
`)

	d := NewDetector()
	d.MinClusterSize = 3
	d.Threshold = 0.25
	clusters, err := d.ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(clusters) < 1 {
		t.Fatal("expected at least 1 cluster from similar server-restart findings")
	}

	// The cluster should contain findings from inv-1, inv-2, inv-3 but not inv-4
	cluster := clusters[0]
	sources := map[string]bool{}
	for _, f := range cluster.Findings {
		sources[f.SourceFile] = true
	}
	if sources["inv-4.md"] {
		t.Error("inv-4 should not be in the cluster")
	}
	if !sources["inv-1.md"] || !sources["inv-2.md"] || !sources["inv-3.md"] {
		t.Errorf("expected inv-1,2,3 in cluster, got sources: %v", sources)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && len(haystack) >= len(needle) &&
		findSubstring(haystack, needle)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
