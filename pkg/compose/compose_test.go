package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseBrief(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orch-go-abc12.md")
	content := `# Brief: orch-go-abc12

## Frame

The system had a problem with authentication tokens expiring during long sessions.
Users reported being logged out unexpectedly during complex orchestration workflows.

## Resolution

The token refresh mechanism was missing a grace period. When the access token expired,
the refresh token request raced with concurrent API calls. Adding a 30-second pre-expiry
refresh window eliminated the race condition. The fix required changes to the OAuth client
and the session middleware.

## Tension

The grace period of 30 seconds is arbitrary. In high-throughput scenarios with many concurrent
agents, the refresh might still race. Should we implement a mutex-based token refresh instead?
`
	os.WriteFile(path, []byte(content), 0644)

	b, err := ParseBrief(path)
	if err != nil {
		t.Fatalf("ParseBrief failed: %v", err)
	}

	if b.ID != "orch-go-abc12" {
		t.Errorf("ID = %q, want %q", b.ID, "orch-go-abc12")
	}

	if !strings.Contains(b.Frame, "authentication tokens") {
		t.Errorf("Frame missing expected content, got: %s", b.Frame)
	}

	if !strings.Contains(b.Resolution, "token refresh mechanism") {
		t.Errorf("Resolution missing expected content, got: %s", b.Resolution)
	}

	if !strings.Contains(b.Tension, "grace period") {
		t.Errorf("Tension missing expected content, got: %s", b.Tension)
	}

	if len(b.Keywords) == 0 {
		t.Error("Expected keywords to be extracted")
	}
}

func TestParseBriefFallbackID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orch-go-xyz99.md")
	content := `## Frame

Some content here.

## Resolution

More content.

## Tension

A question.
`
	os.WriteFile(path, []byte(content), 0644)

	b, err := ParseBrief(path)
	if err != nil {
		t.Fatalf("ParseBrief failed: %v", err)
	}

	if b.ID != "orch-go-xyz99" {
		t.Errorf("ID = %q, want %q (fallback to filename)", b.ID, "orch-go-xyz99")
	}
}

func TestExtractKeywords(t *testing.T) {
	text := "The system authentication token refresh mechanism expired during orchestration"
	kw := ExtractKeywords(text)

	// Should include significant words
	kwSet := make(map[string]bool)
	for _, w := range kw {
		kwSet[w] = true
	}

	expected := []string{"system", "authentication", "token", "refresh", "mechanism", "expired", "orchestration"}
	for _, e := range expected {
		if !kwSet[e] {
			t.Errorf("Expected keyword %q not found in %v", e, kw)
		}
	}

	// Should NOT include stopwords
	unexpected := []string{"the", "during"}
	for _, u := range unexpected {
		if kwSet[u] {
			t.Errorf("Stopword %q should not be in keywords", u)
		}
	}
}

func TestKeywordOverlap(t *testing.T) {
	a := []string{"authentication", "dashboard", "orchestration", "system", "token"}
	b := []string{"authentication", "middleware", "session", "system", "token"}

	overlap := KeywordOverlap(a, b)

	if len(overlap) != 3 {
		t.Errorf("Expected 3 overlapping keywords, got %d: %v", len(overlap), overlap)
	}

	overlapSet := make(map[string]bool)
	for _, w := range overlap {
		overlapSet[w] = true
	}

	for _, expected := range []string{"authentication", "system", "token"} {
		if !overlapSet[expected] {
			t.Errorf("Expected %q in overlap", expected)
		}
	}
}

func TestClusterBriefs(t *testing.T) {
	// Create briefs with known keyword overlap (MinKeywordOverlap = 3)
	briefs := []*Brief{
		{ID: "brief-1", Keywords: []string{"authentication", "dashboard", "oauth", "session", "token"}},
		{ID: "brief-2", Keywords: []string{"authentication", "middleware", "oauth", "refresh", "token"}},
		{ID: "brief-3", Keywords: []string{"authentication", "grace", "oauth", "period", "token"}},
		{ID: "brief-4", Keywords: []string{"clustering", "composition", "digest", "thread"}},
		{ID: "brief-5", Keywords: []string{"clustering", "composition", "knowledge", "thread"}},
		{ID: "brief-6", Keywords: []string{"unrelated", "different", "words", "here"}},
	}

	clusters := ClusterBriefs(briefs)

	if len(clusters) < 1 {
		t.Fatalf("Expected at least 1 cluster, got %d", len(clusters))
	}

	// The auth-related briefs should cluster together (share 3+ keywords: authentication, oauth, token)
	foundAuthCluster := false
	for _, c := range clusters {
		ids := make(map[string]bool)
		for _, b := range c.Briefs {
			ids[b.ID] = true
		}
		if ids["brief-1"] && ids["brief-2"] && ids["brief-3"] {
			foundAuthCluster = true
		}
	}

	if !foundAuthCluster {
		t.Error("Expected auth-related briefs (1,2,3) to cluster together")
	}

	// brief-6 should be unclustered
	unclustered := UnclusteredBriefs(briefs, clusters)
	unclusteredIDs := make(map[string]bool)
	for _, b := range unclustered {
		unclusteredIDs[b.ID] = true
	}
	if !unclusteredIDs["brief-6"] {
		t.Error("Expected brief-6 to be unclustered")
	}
}

func TestClusterBriefsEmpty(t *testing.T) {
	clusters := ClusterBriefs(nil)
	if clusters != nil {
		t.Error("Expected nil clusters for nil input")
	}
}

func TestLoadBriefs(t *testing.T) {
	dir := t.TempDir()

	// Write two valid briefs
	for _, name := range []string{"orch-go-aaa11.md", "orch-go-bbb22.md"} {
		content := `# Brief: ` + strings.TrimSuffix(name, ".md") + `

## Frame

Some frame content about system behavior.

## Resolution

What was found or built.

## Tension

An open question.
`
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	}

	// Write a non-brief markdown file (should be skipped)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Not a brief"), 0644)

	briefs, err := LoadBriefs(dir)
	if err != nil {
		t.Fatalf("LoadBriefs failed: %v", err)
	}

	if len(briefs) != 2 {
		t.Errorf("Expected 2 briefs, got %d", len(briefs))
	}
}

func TestLoadThreads(t *testing.T) {
	dir := t.TempDir()

	content := `---
title: "Epistemic dishonesty — the system conflates didn't-check with nothing-there"
status: forming
created: 2026-03-26
---

# Epistemic dishonesty — the system conflates didn't-check with nothing-there

Five briefs from today share a pattern about system verification and evidence.
`
	os.WriteFile(filepath.Join(dir, "2026-03-26-epistemic-dishonesty.md"), []byte(content), 0644)

	threads, err := LoadThreads(dir)
	if err != nil {
		t.Fatalf("LoadThreads failed: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(threads))
	}

	if !strings.Contains(threads[0].Title, "Epistemic dishonesty") {
		t.Errorf("Thread title = %q, expected to contain 'Epistemic dishonesty'", threads[0].Title)
	}

	if len(threads[0].Keywords) == 0 {
		t.Error("Expected thread keywords to be extracted")
	}
}

func TestLoadThreadsMissingDir(t *testing.T) {
	threads, err := LoadThreads("/nonexistent/path")
	if err != nil {
		t.Errorf("Expected nil error for missing dir, got: %v", err)
	}
	if threads != nil {
		t.Errorf("Expected nil threads for missing dir, got %d", len(threads))
	}
}

func TestMatchClusterToThreads(t *testing.T) {
	cluster := &Cluster{
		Name:           "system / verification / evidence",
		SharedKeywords: []string{"evidence", "system", "verification"},
		Briefs: []*Brief{
			{ID: "b1", Keywords: []string{"evidence", "system", "verification", "pattern", "check"}},
			{ID: "b2", Keywords: []string{"evidence", "system", "verification", "absence", "proof"}},
		},
	}

	threads := []*ThreadInfo{
		{
			Title:    "Epistemic dishonesty",
			Keywords: []string{"absence", "check", "conflates", "dishonesty", "epistemic", "evidence", "pattern", "system", "verification"},
		},
		{
			Title:    "Unrelated thread",
			Keywords: []string{"cooking", "recipes", "kitchen"},
		},
	}

	matches := MatchClusterToThreads(cluster, threads)
	if len(matches) == 0 {
		t.Fatal("Expected at least one thread match")
	}

	if matches[0].Thread.Title != "Epistemic dishonesty" {
		t.Errorf("Best match = %q, expected 'Epistemic dishonesty'", matches[0].Thread.Title)
	}
}

func TestWriteDigest(t *testing.T) {
	dir := t.TempDir()
	digestsDir := filepath.Join(dir, "digests")

	digest := &Digest{
		Date:           mustParseDate("2026-03-27"),
		BriefsComposed: 10,
		ClustersFound:  2,
		Clusters: []*DigestCluster{
			{
				Cluster: &Cluster{
					Name:           "authentication / oauth / token",
					SharedKeywords: []string{"authentication", "oauth", "token"},
					Rationale:      "3 briefs share 3 keywords (authentication, oauth, token).",
					Briefs: []*Brief{
						{ID: "b1", Tension: "Should we use mutex-based refresh?"},
						{ID: "b2", Tension: "Grace period may be too short."},
					},
				},
				ThreadMatches: []ThreadMatch{
					{
						Thread:         &ThreadInfo{Title: "Auth middleware rewrite"},
						SharedKeywords: []string{"authentication", "oauth", "token"},
						Score:          3,
					},
				},
			},
		},
		Unclustered: []*Brief{
			{ID: "b5", Frame: "A brief about something unrelated to any cluster."},
		},
		TensionOrphans: []TensionEntry{
			{BriefID: "b5", Text: "Is this even worth investigating?"},
		},
		EpistemicStatus: DefaultEpistemicStatus,
	}

	path, err := WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("WriteDigest failed: %v", err)
	}

	if !strings.HasSuffix(path, "2026-03-27-digest.md") {
		t.Errorf("Path = %q, expected to end with 2026-03-27-digest.md", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading digest: %v", err)
	}
	content := string(data)

	// Verify frontmatter
	if !strings.Contains(content, "briefs_composed: 10") {
		t.Error("Missing briefs_composed in frontmatter")
	}
	if !strings.Contains(content, "clusters_found: 2") {
		t.Error("Missing clusters_found in frontmatter")
	}
	if !strings.Contains(content, "epistemic_status: unverified-clustering") {
		t.Error("Missing epistemic_status in frontmatter")
	}

	// Verify epistemic status section
	if !strings.Contains(content, "## Epistemic Status") {
		t.Error("Missing Epistemic Status section")
	}
	if !strings.Contains(content, "NOT been verified by a human") {
		t.Error("Missing epistemic warning text")
	}

	// Verify cluster content
	if !strings.Contains(content, "## Cluster 1:") {
		t.Error("Missing cluster heading")
	}
	if !strings.Contains(content, "b1") && !strings.Contains(content, "b2") {
		t.Error("Missing brief IDs in cluster")
	}

	// Verify harvested tensions
	if !strings.Contains(content, "### Harvested tensions") {
		t.Error("Missing harvested tensions section")
	}
	if !strings.Contains(content, "mutex-based refresh") {
		t.Error("Missing tension content from cluster")
	}

	// Verify thread proposal
	if !strings.Contains(content, "### Draft thread proposal") {
		t.Error("Missing thread proposal section")
	}
	if !strings.Contains(content, "Auth middleware rewrite") {
		t.Error("Missing thread reference in proposal")
	}

	// Verify unclustered section
	if !strings.Contains(content, "## Unclustered Briefs") {
		t.Error("Missing unclustered briefs section")
	}

	// Verify tension orphans
	if !strings.Contains(content, "## Tension Orphans") {
		t.Error("Missing tension orphans section")
	}
	if !strings.Contains(content, "Is this even worth investigating?") {
		t.Error("Missing orphan tension content")
	}
}

func TestComposeEndToEnd(t *testing.T) {
	dir := t.TempDir()
	briefsDir := filepath.Join(dir, "briefs")
	threadsDir := filepath.Join(dir, "threads")
	os.MkdirAll(briefsDir, 0755)
	os.MkdirAll(threadsDir, 0755)

	// Create briefs that should cluster
	authBriefs := []struct {
		id      string
		frame   string
		resolve string
		tension string
	}{
		{
			id:      "orch-go-aaa11",
			frame:   "The authentication system failed when OAuth tokens expired during long orchestration sessions with multiple agents.",
			resolve: "Added token refresh mechanism with pre-expiry grace period to the OAuth middleware authentication layer.",
			tension: "The grace period is arbitrary — should we measure actual token refresh latency?",
		},
		{
			id:      "orch-go-bbb22",
			frame:   "OAuth token rotation caused authentication failures across concurrent agent sessions.",
			resolve: "Implemented mutex-based token refresh in the authentication middleware to prevent OAuth race conditions.",
			tension: "Mutex may cause contention under high agent concurrency.",
		},
		{
			id:      "orch-go-ccc33",
			frame:   "Authentication middleware dropped requests when the OAuth token cache was stale.",
			resolve: "Added cache invalidation to the authentication layer with proper OAuth token lifecycle management.",
			tension: "Cache TTL interacts with token expiry in ways we haven't fully mapped.",
		},
	}

	for _, ab := range authBriefs {
		content := "# Brief: " + ab.id + "\n\n## Frame\n\n" + ab.frame + "\n\n## Resolution\n\n" + ab.resolve + "\n\n## Tension\n\n" + ab.tension + "\n"
		os.WriteFile(filepath.Join(briefsDir, ab.id+".md"), []byte(content), 0644)
	}

	// Add an unrelated brief
	unrelated := "# Brief: orch-go-ddd44\n\n## Frame\n\nThe dashboard rendering was slow.\n\n## Resolution\n\nOptimized the React component tree.\n\n## Tension\n\nShould we consider server-side rendering?\n"
	os.WriteFile(filepath.Join(briefsDir, "orch-go-ddd44.md"), []byte(unrelated), 0644)

	// Create a thread
	thread := `---
title: "Auth middleware rewrite — OAuth token lifecycle management"
status: forming
---

# Auth middleware rewrite — OAuth token lifecycle management

Authentication and OAuth token management across the middleware layer.
`
	os.WriteFile(filepath.Join(threadsDir, "2026-03-26-auth-middleware.md"), []byte(thread), 0644)

	digest, err := Compose(briefsDir, threadsDir)
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if digest.BriefsComposed != 4 {
		t.Errorf("BriefsComposed = %d, want 4", digest.BriefsComposed)
	}

	if digest.ClustersFound < 1 {
		t.Errorf("ClustersFound = %d, want >= 1", digest.ClustersFound)
	}

	// Verify the auth briefs clustered
	foundAuthCluster := false
	for _, dc := range digest.Clusters {
		ids := make(map[string]bool)
		for _, b := range dc.Briefs {
			ids[b.ID] = true
		}
		if ids["orch-go-aaa11"] && ids["orch-go-bbb22"] && ids["orch-go-ccc33"] {
			foundAuthCluster = true
			// Check thread matching
			if len(dc.ThreadMatches) == 0 {
				t.Error("Expected auth cluster to match auth thread")
			}
		}
	}
	if !foundAuthCluster {
		t.Error("Expected auth briefs to form a cluster")
	}

	// Write and verify digest file
	digestsDir := filepath.Join(dir, "digests")
	path, err := WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("WriteDigest failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading digest: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "## Epistemic Status") {
		t.Error("Digest missing epistemic status")
	}
	if !strings.Contains(content, "unverified-clustering") {
		t.Error("Digest missing epistemic_status in frontmatter")
	}
}

func TestWeightedKeywords(t *testing.T) {
	text := "token token token refresh refresh authentication"
	weighted := WeightedKeywords(text)

	if len(weighted) < 3 {
		t.Fatalf("Expected at least 3 weighted keywords, got %d", len(weighted))
	}

	if weighted[0].Word != "token" || weighted[0].Count != 3 {
		t.Errorf("Top keyword = %v, want {token, 3}", weighted[0])
	}
}

func TestFilterCommonKeywords(t *testing.T) {
	// Create 12 briefs (above the <10 threshold) where some keywords
	// appear in too many or too few briefs
	briefs := make([]*Brief, 12)
	for i := 0; i < 12; i++ {
		briefs[i] = &Brief{
			ID:       fmt.Sprintf("brief-%d", i),
			Keywords: []string{"ubiquitous", "common", fmt.Sprintf("unique-%d", i)},
		}
	}
	// "ubiquitous" appears in all 12 (100%), "common" in all 12
	// "unique-N" appears in only 1 each
	// Add mid-band keywords to a few briefs
	for i := 0; i < 4; i++ {
		briefs[i].Keywords = append(briefs[i].Keywords, "cluster-signal")
	}

	FilterCommonKeywords(briefs, 0.20) // 20% of 12 = 2.4, so >2 gets filtered

	// "ubiquitous" and "common" should be removed (in all 12 > 2)
	// "unique-N" should be removed (in only 1 < 2)
	// "cluster-signal" should survive (in 4, which is <=2... wait 4 > 2)
	// Actually 20% of 12 = 2.4 → int(2.4) = 2. So words in >2 briefs get cut.
	// cluster-signal is in 4 briefs > 2, so it gets cut too.

	// With maxFreq=0.20 and 12 briefs, threshold=2. All shared words are in >2 briefs.
	// Let me adjust: use maxFreq=0.50 so threshold=6.
	// Reset and try with higher threshold
	briefs2 := make([]*Brief, 12)
	for i := 0; i < 12; i++ {
		briefs2[i] = &Brief{
			ID:       fmt.Sprintf("brief-%d", i),
			Keywords: []string{"everywhere"},
		}
	}
	// Add "target-keyword" to exactly 4 briefs
	for i := 0; i < 4; i++ {
		briefs2[i].Keywords = append(briefs2[i].Keywords, "target-keyword")
	}
	// Add unique words
	for i := 0; i < 12; i++ {
		briefs2[i].Keywords = append(briefs2[i].Keywords, fmt.Sprintf("solo-%d", i))
	}

	FilterCommonKeywords(briefs2, 0.50) // threshold = 6

	// "everywhere" in 12/12 > 6 → removed
	// "solo-N" in 1/12 < 2 → removed
	// "target-keyword" in 4/12 → 2 <= 4 <= 6 → kept
	for i := 0; i < 4; i++ {
		found := false
		for _, kw := range briefs2[i].Keywords {
			if kw == "target-keyword" {
				found = true
			}
			if kw == "everywhere" {
				t.Errorf("Brief %d still has 'everywhere' (should be filtered)", i)
			}
		}
		if !found {
			t.Errorf("Brief %d missing 'target-keyword' (should survive mid-band)", i)
		}
	}
}

func TestFilterCommonKeywordsSmallCorpus(t *testing.T) {
	// With <10 briefs, filtering should be skipped
	briefs := []*Brief{
		{ID: "a", Keywords: []string{"shared", "word", "alpha"}},
		{ID: "b", Keywords: []string{"shared", "word", "beta"}},
	}

	FilterCommonKeywords(briefs, 0.20)

	// Keywords should be unchanged
	if len(briefs[0].Keywords) != 3 {
		t.Errorf("Expected 3 keywords (no filtering for small corpus), got %d", len(briefs[0].Keywords))
	}
}

// mustParseDate creates a time.Time from a date string for test fixtures.
func mustParseDate(s string) (t time.Time) {
	t, _ = time.Parse("2006-01-02", s)
	return
}
