package entropy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveReport writes the entropy report as JSON to a timestamped file in dir.
// Creates dir if it doesn't exist. Returns the written file path.
func SaveReport(report *Report, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := fmt.Sprintf("entropy-%s.json", report.GeneratedAt.Format("2006-01-02"))
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}

	return path, nil
}
