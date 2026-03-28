package orient

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanRecentDigests_NoDir(t *testing.T) {
	result := ScanRecentDigests("/nonexistent", time.Time{})
	if result != nil {
		t.Errorf("expected nil for nonexistent dir, got %+v", result)
	}
}

func TestScanRecentDigests_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := ScanRecentDigests(dir, time.Time{})
	if result != nil {
		t.Errorf("expected nil for empty dir, got %+v", result)
	}
}

func TestScanRecentDigests_FindsDigestNewerThanSession(t *testing.T) {
	dir := t.TempDir()

	digest := `---
date: 2026-03-27
briefs_composed: 87
clusters_found: 9
epistemic_status: unverified-clustering
---

## Cluster 1: dead / checks / claude
Some content here.
`
	os.WriteFile(filepath.Join(dir, "2026-03-27-digest.md"), []byte(digest), 0644)

	// Previous session was 2026-03-25
	prevDate, _ := time.Parse("2006-01-02", "2026-03-25")
	result := ScanRecentDigests(dir, prevDate)

	if result == nil {
		t.Fatal("expected digest summary, got nil")
	}
	if result.BriefsComposed != 87 {
		t.Errorf("expected 87 briefs_composed, got %d", result.BriefsComposed)
	}
	if result.ClustersFound != 9 {
		t.Errorf("expected 9 clusters_found, got %d", result.ClustersFound)
	}
	if result.DigestCount != 1 {
		t.Errorf("expected 1 digest, got %d", result.DigestCount)
	}
}

func TestScanRecentDigests_IgnoresOlderDigests(t *testing.T) {
	dir := t.TempDir()

	digest := `---
date: 2026-03-20
briefs_composed: 10
clusters_found: 2
---

## Cluster 1
Content.
`
	os.WriteFile(filepath.Join(dir, "2026-03-20-digest.md"), []byte(digest), 0644)

	// Previous session was 2026-03-25 — digest is older
	prevDate, _ := time.Parse("2006-01-02", "2026-03-25")
	result := ScanRecentDigests(dir, prevDate)

	if result != nil {
		t.Errorf("expected nil for older digest, got %+v", result)
	}
}

func TestScanRecentDigests_IncludesSameDayDigest(t *testing.T) {
	dir := t.TempDir()

	digest := `---
date: 2026-03-25
briefs_composed: 15
clusters_found: 3
---
`
	os.WriteFile(filepath.Join(dir, "2026-03-25-digest.md"), []byte(digest), 0644)

	// Previous session was also 2026-03-25 — same day should be included
	prevDate, _ := time.Parse("2006-01-02", "2026-03-25")
	result := ScanRecentDigests(dir, prevDate)

	if result == nil {
		t.Fatal("expected digest for same-day, got nil")
	}
	if result.BriefsComposed != 15 {
		t.Errorf("expected 15 briefs, got %d", result.BriefsComposed)
	}
}

func TestScanRecentDigests_AggregatesMultipleDigests(t *testing.T) {
	dir := t.TempDir()

	digest1 := `---
date: 2026-03-26
briefs_composed: 20
clusters_found: 4
---
`
	digest2 := `---
date: 2026-03-27
briefs_composed: 30
clusters_found: 5
---
`
	os.WriteFile(filepath.Join(dir, "2026-03-26-digest.md"), []byte(digest1), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-27-digest.md"), []byte(digest2), 0644)

	prevDate, _ := time.Parse("2006-01-02", "2026-03-25")
	result := ScanRecentDigests(dir, prevDate)

	if result == nil {
		t.Fatal("expected digest summary, got nil")
	}
	if result.DigestCount != 2 {
		t.Errorf("expected 2 digests, got %d", result.DigestCount)
	}
	if result.BriefsComposed != 50 {
		t.Errorf("expected 50 total briefs, got %d", result.BriefsComposed)
	}
	if result.ClustersFound != 9 {
		t.Errorf("expected 9 total clusters, got %d", result.ClustersFound)
	}
}

func TestScanRecentDigests_ZeroPrevDate(t *testing.T) {
	dir := t.TempDir()

	digest := `---
date: 2026-03-27
briefs_composed: 40
clusters_found: 6
---
`
	os.WriteFile(filepath.Join(dir, "2026-03-27-digest.md"), []byte(digest), 0644)

	// Zero time means no previous session — show all digests
	result := ScanRecentDigests(dir, time.Time{})

	if result == nil {
		t.Fatal("expected digest summary, got nil")
	}
	if result.BriefsComposed != 40 {
		t.Errorf("expected 40 briefs, got %d", result.BriefsComposed)
	}
}

func TestFormatDigestSummary_Nil(t *testing.T) {
	result := FormatDigestSummary(nil)
	if result != "" {
		t.Errorf("expected empty for nil, got %q", result)
	}
}

func TestFormatDigestSummary_Basic(t *testing.T) {
	summary := &DigestSummary{
		DigestCount:    1,
		BriefsComposed: 87,
		ClustersFound:  9,
	}
	result := FormatDigestSummary(summary)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(result, "87 briefs") {
		t.Errorf("expected brief count in output, got %q", result)
	}
	if !contains(result, "9 themes") {
		t.Errorf("expected cluster count in output, got %q", result)
	}
}

func TestFormatDigestSummary_WithMaintenance(t *testing.T) {
	summary := &DigestSummary{
		DigestCount:      1,
		BriefsComposed:   87,
		ClustersFound:    9,
		MaintenanceCount: 12,
	}
	result := FormatDigestSummary(summary)
	if !contains(result, "12 maintenance") {
		t.Errorf("expected maintenance count in output, got %q", result)
	}
}

func TestFormatDigestSummary_MultipleDigests(t *testing.T) {
	summary := &DigestSummary{
		DigestCount:    3,
		BriefsComposed: 120,
		ClustersFound:  15,
	}
	result := FormatDigestSummary(summary)
	if !contains(result, "120 briefs") {
		t.Errorf("expected aggregated brief count, got %q", result)
	}
}

func TestDigestedBriefIDs(t *testing.T) {
	dir := t.TempDir()
	digest := `---
date: 2026-03-28
briefs_composed: 4
clusters_found: 1
---

## Cluster 1: auth / routing / drift

**Briefs:** orch-go-a1, orch-go-b2

## Unclustered Briefs

- **orch-go-c3** - Summary
- **orch-go-d4** - Summary
`
	if err := os.WriteFile(filepath.Join(dir, "2026-03-28-digest.md"), []byte(digest), 0644); err != nil {
		t.Fatalf("write digest: %v", err)
	}

	ids := DigestedBriefIDs(dir)
	for _, id := range []string{"orch-go-a1", "orch-go-b2", "orch-go-c3", "orch-go-d4"} {
		if !ids[id] {
			t.Fatalf("expected %s to be marked digested", id)
		}
	}
}
