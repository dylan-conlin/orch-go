// Package claims provides machine-readable claim tracking for knowledge models.
//
// Each model directory can have a claims.yaml file that indexes the model's
// testable claims as atoms — machine-readable units that drive orient surfacing,
// daemon probe generation, and completion pipeline claim updates.
//
// claims.yaml is an overlay, not the source of truth. The model.md prose
// remains authoritative; claims.yaml is a structured index for consumption.
package claims

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Confidence represents the lifecycle state of a claim.
type Confidence string

const (
	Unconfirmed Confidence = "unconfirmed"
	Confirmed   Confidence = "confirmed"
	Contested   Confidence = "contested"
	Stale       Confidence = "stale"
)

// ClaimType categorizes the nature of a claim.
type ClaimType string

const (
	TypeObservation    ClaimType = "observation"
	TypeMechanism      ClaimType = "mechanism"
	TypeGeneralization ClaimType = "generalization"
	TypeInvariant      ClaimType = "invariant"
)

// Scope indicates how broadly a claim applies.
type Scope string

const (
	ScopeLocal     Scope = "local"
	ScopeBounded   Scope = "bounded"
	ScopeUniversal Scope = "universal"
)

// Priority indicates how important a claim is for probe generation.
type Priority string

const (
	PriorityCore       Priority = "core"
	PrioritySupporting Priority = "supporting"
	PriorityPeripheral Priority = "peripheral"
)

// Evidence records a single piece of supporting or contradicting evidence.
type Evidence struct {
	Source   string `yaml:"source"`
	Date     string `yaml:"date"` // YYYY-MM-DD
	Verdict  string `yaml:"verdict"` // confirms, contradicts, extends
	External bool   `yaml:"external,omitempty"`
}

// Tension records a cross-model conflict or relationship.
type Tension struct {
	Claim string `yaml:"claim"` // e.g., "MH-05"
	Model string `yaml:"model"` // e.g., "measurement-honesty"
	Type  string `yaml:"type"`  // extends, confirms, contradicts
	Note  string `yaml:"note"`
}

// Claim is a single testable assertion extracted from a model.
type Claim struct {
	ID            string     `yaml:"id"`
	Text          string     `yaml:"text"`
	Type          ClaimType  `yaml:"type"`
	Scope         Scope      `yaml:"scope"`
	Confidence    Confidence `yaml:"confidence"`
	Priority      Priority   `yaml:"priority"`
	Evidence      []Evidence `yaml:"evidence,omitempty"`
	LastValidated string     `yaml:"last_validated,omitempty"` // YYYY-MM-DD
	DomainTags    []string   `yaml:"domain_tags,omitempty"`
	FalsifiesIf   string     `yaml:"falsifies_if,omitempty"`
	Tensions      []Tension  `yaml:"tensions,omitempty"`
	ModelMdRef    string     `yaml:"model_md_ref,omitempty"`
}

// File represents the contents of a claims.yaml file.
type File struct {
	Model     string  `yaml:"model"`
	Version   int     `yaml:"version"`
	LastAudit string  `yaml:"last_audit,omitempty"` // YYYY-MM-DD
	Claims    []Claim `yaml:"claims"`
}

// StalenessThresholdDays is the number of days after which a confirmed claim
// becomes stale if no new evidence has been added.
const StalenessThresholdDays = 30

// LoadFile reads and parses a claims.yaml file.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read claims file: %w", err)
	}
	return Parse(data)
}

// Parse unmarshals YAML bytes into a File.
func Parse(data []byte) (*File, error) {
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse claims yaml: %w", err)
	}
	return &f, nil
}

// Marshal serializes a File to YAML bytes.
func Marshal(f *File) ([]byte, error) {
	return yaml.Marshal(f)
}

// SaveFile writes a File to disk as YAML.
func SaveFile(path string, f *File) error {
	data, err := Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal claims: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ScanAll reads claims.yaml from all model directories under modelsDir.
// Returns a map of model name to File. Skips models without claims.yaml.
func ScanAll(modelsDir string) (map[string]*File, error) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read models dir: %w", err)
	}

	result := make(map[string]*File)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(modelsDir, entry.Name(), "claims.yaml")
		f, err := LoadFile(path)
		if err != nil {
			continue // skip models without claims.yaml
		}
		result[entry.Name()] = f
	}
	return result, nil
}

// IsStale returns true if the claim's last_validated date is older than
// StalenessThresholdDays from now.
func (c *Claim) IsStale(now time.Time) bool {
	if c.LastValidated == "" {
		return true
	}
	validated, err := time.Parse("2006-01-02", c.LastValidated)
	if err != nil {
		return true
	}
	return now.Sub(validated) > time.Duration(StalenessThresholdDays)*24*time.Hour
}

// IsProbeEligible returns true if this claim should be considered for probe
// generation: unconfirmed or stale, and priority is core or supporting.
// Claims with recent evidence (within StalenessThresholdDays) are skipped
// even if confidence is still unconfirmed — a probe already ran.
func (c *Claim) IsProbeEligible(now time.Time) bool {
	if c.Priority == PriorityPeripheral {
		return false
	}
	// If evidence was collected recently, don't re-probe regardless of confidence.
	// This prevents re-spawning probes for claims where a probe completed but
	// confidence wasn't updated (e.g., verdict was "indirectly supported").
	if c.HasRecentEvidence(now) {
		return false
	}
	if c.Confidence == Unconfirmed || c.Confidence == Stale {
		return true
	}
	if c.Confidence == Confirmed && c.IsStale(now) {
		return true
	}
	return false
}

// HasRecentEvidence returns true if the claim has any evidence entry with a
// date within StalenessThresholdDays of now.
func (c *Claim) HasRecentEvidence(now time.Time) bool {
	threshold := time.Duration(StalenessThresholdDays) * 24 * time.Hour
	for _, e := range c.Evidence {
		if e.Date == "" {
			continue
		}
		d, err := time.Parse("2006-01-02", e.Date)
		if err != nil {
			continue
		}
		if now.Sub(d) <= threshold {
			return true
		}
	}
	return false
}

// HasDomainOverlap returns true if any of the claim's domain tags match
// any of the given keywords.
func (c *Claim) HasDomainOverlap(keywords []string) bool {
	for _, tag := range c.DomainTags {
		for _, kw := range keywords {
			if tag == kw {
				return true
			}
		}
	}
	return false
}

// ModelClaimStatus summarizes claim confidence counts for a single model.
type ModelClaimStatus struct {
	ModelName    string
	Total        int
	Confirmed    int
	Unconfirmed  int
	Contested    int
	Stale        int
	CoreUntested int // core priority + (unconfirmed or stale)
}

// RecentDisconfirmation represents a claim contradicted by evidence within a recent window.
type RecentDisconfirmation struct {
	ModelName string
	ClaimID   string
	ClaimText string
	Source    string
	Date      string
}

// Edge represents a notable claim-level finding for orient surfacing.
type Edge struct {
	Type      string // "tension", "stale_active", "unconfirmed_core"
	ClaimID   string
	ClaimText string
	ModelName string
	Detail    string
}

// CollectEdges scans all claims files and returns notable edges for orient.
// Returns at most maxEdges edges, prioritizing tensions > stale-in-active > unconfirmed-core.
func CollectEdges(files map[string]*File, now time.Time, activeKeywords []string, maxEdges int) []Edge {
	var tensions, staleActive, unconfirmedCore []Edge

	for modelName, f := range files {
		for _, c := range f.Claims {
			// Tensions
			for _, t := range c.Tensions {
				if t.Type == "contradicts" || t.Type == "extends" {
					tensions = append(tensions, Edge{
						Type:      "tension",
						ClaimID:   c.ID,
						ClaimText: c.Text,
						ModelName: modelName,
						Detail:    fmt.Sprintf("%s vs %s (%s): %s", c.ID, t.Claim, t.Model, t.Note),
					})
				}
			}

			// Stale in active area
			if c.Confidence == Confirmed && c.IsStale(now) && c.HasDomainOverlap(activeKeywords) {
				staleActive = append(staleActive, Edge{
					Type:      "stale_active",
					ClaimID:   c.ID,
					ClaimText: c.Text,
					ModelName: modelName,
					Detail:    fmt.Sprintf("%s (%s): last validated %s, domain overlap with active work", c.ID, truncate(c.Text, 60), c.LastValidated),
				})
			}

			// Unconfirmed core
			if c.Confidence == Unconfirmed && c.Priority == PriorityCore {
				unconfirmedCore = append(unconfirmedCore, Edge{
					Type:      "unconfirmed_core",
					ClaimID:   c.ID,
					ClaimText: c.Text,
					ModelName: modelName,
					Detail:    fmt.Sprintf("%s (%s): untested core claim in %s", c.ID, truncate(c.Text, 60), modelName),
				})
			}
		}
	}

	var result []Edge
	// Tensions first (max 2)
	for i := 0; i < len(tensions) && i < 2 && len(result) < maxEdges; i++ {
		result = append(result, tensions[i])
	}
	// Stale in active area (max 2)
	for i := 0; i < len(staleActive) && i < 2 && len(result) < maxEdges; i++ {
		result = append(result, staleActive[i])
	}
	// Unconfirmed core (max 1)
	for i := 0; i < len(unconfirmedCore) && i < 1 && len(result) < maxEdges; i++ {
		result = append(result, unconfirmedCore[i])
	}

	return result
}

// FormatEdges renders edges as text for orient output.
func FormatEdges(edges []Edge) string {
	if len(edges) == 0 {
		return ""
	}

	var b string
	b += "Knowledge Edges:\n"

	var lastType string
	for _, e := range edges {
		if e.Type != lastType {
			switch e.Type {
			case "tension":
				b += "  Tensions:\n"
			case "stale_active":
				b += "  Stale in active area:\n"
			case "unconfirmed_core":
				b += "  Unconfirmed core:\n"
			}
			lastType = e.Type
		}
		b += fmt.Sprintf("    - %s\n", e.Detail)
	}
	b += "\n"
	return b
}

// CollectClaimStatus computes per-model claim summaries from all claims files.
// Only returns models that have at least one untested (unconfirmed/stale) claim.
func CollectClaimStatus(files map[string]*File, now time.Time) []ModelClaimStatus {
	var result []ModelClaimStatus

	for modelName, f := range files {
		s := ModelClaimStatus{
			ModelName: modelName,
			Total:     len(f.Claims),
		}
		for _, c := range f.Claims {
			switch c.Confidence {
			case Confirmed:
				if c.IsStale(now) {
					s.Stale++
					if c.Priority == PriorityCore {
						s.CoreUntested++
					}
				} else {
					s.Confirmed++
				}
			case Unconfirmed:
				s.Unconfirmed++
				if c.Priority == PriorityCore {
					s.CoreUntested++
				}
			case Contested:
				s.Contested++
			case Stale:
				s.Stale++
				if c.Priority == PriorityCore {
					s.CoreUntested++
				}
			}
		}
		if s.Unconfirmed > 0 || s.Stale > 0 || s.Contested > 0 {
			result = append(result, s)
		}
	}

	// Sort by core untested descending for priority surfacing
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CoreUntested > result[i].CoreUntested {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// CollectRecentDisconfirmations finds claims with "contradicts" evidence within
// the given day window from now.
func CollectRecentDisconfirmations(files map[string]*File, now time.Time, days int) []RecentDisconfirmation {
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	var result []RecentDisconfirmation

	for modelName, f := range files {
		for _, c := range f.Claims {
			for _, ev := range c.Evidence {
				if ev.Verdict != "contradicts" {
					continue
				}
				d, err := time.Parse("2006-01-02", ev.Date)
				if err != nil {
					continue
				}
				if d.Before(cutoff) {
					continue
				}
				result = append(result, RecentDisconfirmation{
					ModelName: modelName,
					ClaimID:   c.ID,
					ClaimText: c.Text,
					Source:    ev.Source,
					Date:      ev.Date,
				})
			}
		}
	}

	return result
}

// FormatClaimSurface renders the full Knowledge Edges section for orient,
// combining claim status summaries, recent disconfirmations, and notable edges.
func FormatClaimSurface(statuses []ModelClaimStatus, disconfirmations []RecentDisconfirmation, edges []Edge) string {
	if len(statuses) == 0 && len(disconfirmations) == 0 && len(edges) == 0 {
		return ""
	}

	var b string
	b += "Knowledge Edges:\n"

	// Untested claims summary
	if len(statuses) > 0 {
		b += "  Untested claims:\n"
		for _, s := range statuses {
			detail := fmt.Sprintf("%d/%d confirmed", s.Confirmed, s.Total)
			if s.CoreUntested > 0 {
				detail += fmt.Sprintf(", %d untested core", s.CoreUntested)
			}
			if s.Contested > 0 {
				detail += fmt.Sprintf(", %d contested", s.Contested)
			}
			b += fmt.Sprintf("    - %s: %s\n", s.ModelName, detail)
		}
	}

	// Recently disconfirmed
	if len(disconfirmations) > 0 {
		b += "  Recently disconfirmed:\n"
		for _, d := range disconfirmations {
			b += fmt.Sprintf("    - %s (%s): %s (%s)\n", d.ClaimID, d.ModelName, truncate(d.Source, 60), d.Date)
		}
	}

	// Existing edge types (tensions, stale-in-active, unconfirmed-core)
	var lastType string
	for _, e := range edges {
		if e.Type != lastType {
			switch e.Type {
			case "tension":
				b += "  Tensions:\n"
			case "stale_active":
				b += "  Stale in active area:\n"
			case "unconfirmed_core":
				b += "  Unconfirmed core:\n"
			}
			lastType = e.Type
		}
		b += fmt.Sprintf("    - %s\n", e.Detail)
	}

	b += "\n"
	return b
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
