package kbmetrics

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ModelSizeReport describes a model that exceeds the size threshold.
type ModelSizeReport struct {
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	SizeBytes   int64   `json:"size_bytes"`
	SizeKB      float64 `json:"size_kb"`
	LastUpdated string  `json:"last_updated,omitempty"` // YYYY-MM-DD or ""
	DaysSince   int     `json:"days_since_update"`      // -1 if unknown
	NeedsReview bool    `json:"needs_review"`           // oversized + stale
}

// AuditModelSize scans .kb/models/ and .kb/global/models/ for model.md files
// exceeding thresholdBytes. Models that also haven't been updated in
// stalenessDays are flagged NeedsReview.
func AuditModelSize(kbDir string, thresholdBytes int64, stalenessDays int) ([]ModelSizeReport, error) {
	modelsDir := filepath.Join(kbDir, "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("models directory not found: %s", modelsDir)
	}

	var reports []ModelSizeReport

	// Scan project models
	projectReports, err := scanModelsDir(modelsDir, thresholdBytes, stalenessDays)
	if err != nil {
		return nil, err
	}
	reports = append(reports, projectReports...)

	// Scan global models if present
	globalModelsDir := filepath.Join(kbDir, "global", "models")
	if _, err := os.Stat(globalModelsDir); err == nil {
		globalReports, err := scanModelsDir(globalModelsDir, thresholdBytes, stalenessDays)
		if err != nil {
			return nil, err
		}
		reports = append(reports, globalReports...)
	}

	// Sort by size descending (largest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].SizeBytes > reports[j].SizeBytes
	})

	return reports, nil
}

// scanModelsDir walks a models directory and returns reports for oversized models.
func scanModelsDir(modelsDir string, thresholdBytes int64, stalenessDays int) ([]ModelSizeReport, error) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("read models dir: %w", err)
	}

	var reports []ModelSizeReport
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(modelsDir, entry.Name(), "model.md")
		info, err := os.Stat(modelPath)
		if err != nil {
			continue // skip dirs without model.md
		}

		size := info.Size()
		if size < thresholdBytes {
			continue // under threshold
		}

		// Read content for Last Updated
		data, err := os.ReadFile(modelPath)
		if err != nil {
			continue
		}

		lastUpdated := extractLastUpdated(string(data))

		daysSince := -1
		if lastUpdated != "" {
			if t, err := time.Parse("2006-01-02", lastUpdated); err == nil {
				daysSince = int(math.Floor(now.Sub(t).Hours() / 24))
			}
		}

		needsReview := daysSince == -1 || daysSince > stalenessDays

		reports = append(reports, ModelSizeReport{
			Name:        entry.Name(),
			Path:        modelPath,
			SizeBytes:   size,
			SizeKB:      math.Round(float64(size)/1024*10) / 10,
			LastUpdated: lastUpdated,
			DaysSince:   daysSince,
			NeedsReview: needsReview,
		})
	}

	return reports, nil
}

// FormatModelSizeText produces a human-readable model size audit report.
func FormatModelSizeText(reports []ModelSizeReport) string {
	var b strings.Builder

	needsReview := 0
	for _, r := range reports {
		if r.NeedsReview {
			needsReview++
		}
	}

	b.WriteString(fmt.Sprintf("Model Size Audit — %d oversized (>30KB), %d need review\n", len(reports), needsReview))
	b.WriteString(strings.Repeat("=", 60) + "\n\n")

	if len(reports) == 0 {
		b.WriteString("No models exceed 30KB.\n")
		return b.String()
	}

	for _, r := range reports {
		flag := "  "
		if r.NeedsReview {
			flag = "* "
		}

		updated := r.LastUpdated
		if updated == "" {
			updated = "unknown"
		}

		reviewTag := ""
		if r.NeedsReview {
			reviewTag = " <- REVIEW"
		}

		b.WriteString(fmt.Sprintf("%s%-40s %6.1f KB  updated: %-12s (%dd ago)%s\n",
			flag, r.Name, r.SizeKB, updated, r.DaysSince, reviewTag))
	}

	if needsReview > 0 {
		b.WriteString(fmt.Sprintf("\n* %d model(s) exceed 30KB and haven't been consolidated in 2+ weeks.\n", needsReview))
		b.WriteString("  Route to architect for synthesis/pruning.\n")
	}

	return b.String()
}
