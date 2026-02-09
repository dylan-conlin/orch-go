// Package friction provides lightweight orchestration friction ledger storage.
package friction

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is one friction incident captured in real time.
type Entry struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Symptom      string    `json:"symptom"`
	Impact       string    `json:"impact"`
	EvidencePath string    `json:"evidence_path"`
	LinkedIssue  string    `json:"linked_issue"`
}

// Summary groups repeated incidents by symptom.
type Summary struct {
	Symptom        string    `json:"symptom"`
	Count          int       `json:"count"`
	LastSeen       time.Time `json:"last_seen"`
	LatestImpact   string    `json:"latest_impact"`
	LatestEvidence string    `json:"latest_evidence"`
	LinkedIssues   []string  `json:"linked_issues"`
}

// Validate ensures required fields are present.
func (e *Entry) Validate() error {
	if strings.TrimSpace(e.Symptom) == "" {
		return errors.New("symptom is required")
	}
	if strings.TrimSpace(e.Impact) == "" {
		return errors.New("impact is required")
	}
	if strings.TrimSpace(e.EvidencePath) == "" {
		return errors.New("evidence_path is required")
	}
	if strings.TrimSpace(e.LinkedIssue) == "" {
		return errors.New("linked_issue is required")
	}
	return nil
}

// Append adds one friction entry to a JSONL ledger file.
func Append(path string, entry Entry) (Entry, error) {
	if err := entry.Validate(); err != nil {
		return Entry{}, err
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = generateID(entry.Timestamp)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Entry{}, fmt.Errorf("create ledger directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return Entry{}, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return Entry{}, fmt.Errorf("marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return Entry{}, fmt.Errorf("write entry: %w", err)
	}

	return entry, nil
}

// Load returns all friction entries from JSONL ledger file.
func Load(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	entries := make([]Entry, 0, 64)
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		var e Entry
		if err := json.Unmarshal([]byte(text), &e); err != nil {
			return nil, fmt.Errorf("parse ledger line %d: %w", line, err)
		}
		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}

	return entries, nil
}

// Summarize groups entries by normalized symptom and returns high-frequency first.
func Summarize(entries []Entry) []Summary {
	type bucket struct {
		summary Summary
		issues  map[string]struct{}
	}

	buckets := make(map[string]*bucket)

	for _, e := range entries {
		key := normalizeSymptom(e.Symptom)
		if key == "" {
			continue
		}

		b, ok := buckets[key]
		if !ok {
			b = &bucket{
				summary: Summary{
					Symptom:        strings.TrimSpace(e.Symptom),
					Count:          0,
					LastSeen:       e.Timestamp,
					LatestImpact:   strings.TrimSpace(e.Impact),
					LatestEvidence: strings.TrimSpace(e.EvidencePath),
					LinkedIssues:   []string{},
				},
				issues: map[string]struct{}{},
			}
			buckets[key] = b
		}

		b.summary.Count++

		if e.Timestamp.After(b.summary.LastSeen) {
			b.summary.LastSeen = e.Timestamp
			b.summary.LatestImpact = strings.TrimSpace(e.Impact)
			b.summary.LatestEvidence = strings.TrimSpace(e.EvidencePath)
			b.summary.Symptom = strings.TrimSpace(e.Symptom)
		}

		issue := strings.TrimSpace(e.LinkedIssue)
		if issue != "" {
			if _, seen := b.issues[issue]; !seen {
				b.issues[issue] = struct{}{}
				b.summary.LinkedIssues = append(b.summary.LinkedIssues, issue)
			}
		}
	}

	result := make([]Summary, 0, len(buckets))
	for _, b := range buckets {
		sort.Strings(b.summary.LinkedIssues)
		result = append(result, b.summary)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	return result
}

func generateID(ts time.Time) string {
	return fmt.Sprintf("fr-%s", ts.UTC().Format("20060102-150405.000000000"))
}

func normalizeSymptom(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.Join(strings.Fields(v), " ")
	return v
}
