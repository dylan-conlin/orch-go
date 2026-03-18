// Package digest produces consumable thinking products from KB artifact changes.
// It scans .kb/threads/, .kb/models/, and .kb/investigations/ for changes
// and packages notable ones as product files in ~/.orch/digest/.
package digest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Product type constants.
const (
	TypeThreadProgression ProductType = "thread_progression"
	TypeModelUpdate       ProductType = "model_update"
	TypeModelProbe        ProductType = "model_probe"
	TypeDecisionBrief     ProductType = "decision_brief"
)

// Product state constants.
const (
	StateNew      ProductState = "new"
	StateRead     ProductState = "read"
	StateStarred  ProductState = "starred"
	StateArchived ProductState = "archived"
)

// Significance levels.
const (
	SignificanceLow    = "low"
	SignificanceMedium = "medium"
	SignificanceHigh   = "high"
)

// Thresholds for product creation.
const (
	// ThreadDeltaWordThreshold is the minimum word delta to create a thread product.
	ThreadDeltaWordThreshold = 200
)

// ProductType is the type of digest product.
type ProductType string

// ProductState is the lifecycle state of a digest product.
type ProductState string

// Product is a single thinking product written to ~/.orch/digest/.
type Product struct {
	ID           string      `json:"id"`
	Type         ProductType `json:"type"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	Significance string      `json:"significance"`
	Source       Source      `json:"source"`
	State        ProductState `json:"state"`
	CreatedAt    time.Time   `json:"created_at"`
	ReadAt       time.Time   `json:"read_at,omitempty"`
	StarredAt    time.Time   `json:"starred_at,omitempty"`
	ArchivedAt   time.Time   `json:"archived_at,omitempty"`
}

// Source is the source artifact that triggered the product.
type Source struct {
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	ChangeType   string `json:"change_type"`
	DeltaWords   int    `json:"delta_words,omitempty"`
}

// State tracks what's been scanned to avoid duplicate products.
type State struct {
	LastScan   time.Time         `json:"last_scan"`
	FileHashes map[string]string `json:"file_hashes"`
	Stats      Stats             `json:"stats"`
}

// Stats tracks aggregate statistics for the digest system.
type Stats struct {
	TotalProduced int `json:"total_produced"`
	TotalRead     int `json:"total_read"`
	TotalStarred  int `json:"total_starred"`
}

// StatsResponse is the API response for /api/digest/stats.
type StatsResponse struct {
	Unread  int `json:"unread"`
	Read    int `json:"read"`
	Starred int `json:"starred"`
	Total   int `json:"total"`
}

// ArtifactChange represents a detected change in a KB artifact.
type ArtifactChange struct {
	Path       string
	ChangeType string // "content_added", "created", "modified"
	DeltaWords int
	Summary    string
}

// Service provides I/O operations for digest scanning.
type Service interface {
	// ScanThreads scans .kb/threads/ for changes since the given hashes.
	ScanThreads(hashes map[string]string) ([]ArtifactChange, map[string]string, error)
	// ScanModels scans .kb/models/ for changes since the given hashes.
	ScanModels(hashes map[string]string) ([]ArtifactChange, map[string]string, error)
	// ScanInvestigations scans .kb/investigations/ for completed investigations.
	ScanInvestigations(hashes map[string]string) ([]ArtifactChange, map[string]string, error)
}

// Result contains the result of a periodic digest scan.
type Result struct {
	Produced int
	Skipped  int
	Scanned  int
	Message  string
	Error    error
}

// ListOpts controls filtering for Store.List.
type ListOpts struct {
	State ProductState
	Type  ProductType
	Limit int
}

// --- Digest Store (filesystem-backed product storage) ---

// Store reads and writes digest product files.
type Store struct {
	dir string
}

// NewStore creates a Store backed by the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Write persists a Product to a JSON file.
func (s *Store) Write(p Product) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create digest dir: %w", err)
	}
	path := filepath.Join(s.dir, p.ID+".json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal product: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// List returns all products matching the given filter options.
func (s *Store) List(opts ListOpts) ([]Product, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read digest dir: %w", err)
	}

	var products []Product
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var p Product
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		if opts.State != "" && p.State != opts.State {
			continue
		}
		if opts.Type != "" && p.Type != opts.Type {
			continue
		}
		products = append(products, p)
	}

	// Sort by created_at descending (newest first)
	sort.Slice(products, func(i, j int) bool {
		return products[i].CreatedAt.After(products[j].CreatedAt)
	})

	if opts.Limit > 0 && len(products) > opts.Limit {
		products = products[:opts.Limit]
	}

	return products, nil
}

// Get returns a single product by ID.
func (s *Store) Get(id string) (*Product, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read product %s: %w", id, err)
	}
	var p Product
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal product %s: %w", id, err)
	}
	return &p, nil
}

// UpdateState transitions a product to a new state.
func (s *Store) UpdateState(id string, state ProductState) error {
	p, err := s.Get(id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	p.State = state
	switch state {
	case StateRead:
		p.ReadAt = now
	case StateStarred:
		p.StarredAt = now
	case StateArchived:
		p.ArchivedAt = now
	}

	return s.Write(*p)
}

// StoreStats returns aggregate statistics about products.
func (s *Store) StoreStats() (StatsResponse, error) {
	products, err := s.List(ListOpts{})
	if err != nil {
		return StatsResponse{}, err
	}

	var stats StatsResponse
	stats.Total = len(products)
	for _, p := range products {
		switch p.State {
		case StateNew:
			stats.Unread++
		case StateRead:
			stats.Read++
		case StateStarred:
			stats.Starred++
		}
	}
	return stats, nil
}

// ArchiveRead archives read products older than the given duration.
// Returns the number of products archived.
func (s *Store) ArchiveRead(olderThan time.Duration) (int, error) {
	products, err := s.List(ListOpts{State: StateRead})
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().UTC().Add(-olderThan)
	archived := 0
	for _, p := range products {
		if p.CreatedAt.Before(cutoff) {
			if err := s.UpdateState(p.ID, StateArchived); err != nil {
				continue
			}
			archived++
		}
	}
	return archived, nil
}

// --- State persistence ---

// LoadState loads the digest scan state from disk.
// Returns an empty state (not error) if the file doesn't exist.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{FileHashes: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("read digest state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal digest state: %w", err)
	}
	if state.FileHashes == nil {
		state.FileHashes = make(map[string]string)
	}
	return &state, nil
}

// SaveState persists the digest scan state to disk.
func SaveState(path string, state *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal digest state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- RunDigest: core scanning logic ---

// RunDigest scans KB artifacts for changes and creates digest products.
// This contains the core logic previously in Daemon.RunPeriodicDigest.
func RunDigest(svc Service, digestDir, statePath string) *Result {
	// Load scan state
	state, err := LoadState(statePath)
	if err != nil {
		return &Result{
			Error:   err,
			Message: fmt.Sprintf("Digest: failed to load state: %v", err),
		}
	}

	store := NewStore(digestDir)
	result := &Result{}

	// Scan all artifact types
	var allChanges []changeWithType

	threadChanges, threadHashes, err := svc.ScanThreads(state.FileHashes)
	if err == nil {
		for _, c := range threadChanges {
			allChanges = append(allChanges, changeWithType{change: c, productType: TypeThreadProgression})
		}
		mergeHashes(state.FileHashes, threadHashes)
	}

	modelChanges, modelHashes, err := svc.ScanModels(state.FileHashes)
	if err == nil {
		for _, c := range modelChanges {
			allChanges = append(allChanges, changeWithType{change: c, productType: TypeModelUpdate})
		}
		mergeHashes(state.FileHashes, modelHashes)
	}

	invChanges, invHashes, err := svc.ScanInvestigations(state.FileHashes)
	if err == nil {
		for _, c := range invChanges {
			allChanges = append(allChanges, changeWithType{change: c, productType: TypeDecisionBrief})
		}
		mergeHashes(state.FileHashes, invHashes)
	}

	result.Scanned = len(allChanges)

	// Create products for significant changes
	now := time.Now().UTC()
	for _, cwt := range allChanges {
		c := cwt.change
		significance := ClassifySignificance(cwt.productType, c)
		if significance == "" {
			result.Skipped++
			continue
		}

		id := fmt.Sprintf("%s-%s-%s",
			now.Format("20060102T1504"),
			string(cwt.productType),
			SanitizeSlug(filepath.Base(c.Path)),
		)

		title := FormatTitle(cwt.productType, c)

		p := Product{
			ID:           id,
			Type:         cwt.productType,
			Title:        title,
			Summary:      c.Summary,
			Significance: significance,
			Source: Source{
				ArtifactType: ArtifactTypeFromProductType(cwt.productType),
				Path:         c.Path,
				ChangeType:   c.ChangeType,
				DeltaWords:   c.DeltaWords,
			},
			State:     StateNew,
			CreatedAt: now,
		}

		if err := store.Write(p); err != nil {
			result.Error = err
			continue
		}
		result.Produced++
	}

	// Update state
	state.LastScan = now
	state.Stats.TotalProduced += result.Produced
	if err := SaveState(statePath, state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save digest state: %v\n", err)
	}

	// Build summary
	if result.Produced > 0 {
		result.Message = fmt.Sprintf("Digest: produced %d product(s) from %d change(s)", result.Produced, result.Scanned)
	} else if result.Scanned > 0 {
		result.Message = fmt.Sprintf("Digest: scanned %d change(s), %d below threshold", result.Scanned, result.Skipped)
	} else {
		result.Message = "Digest: no artifact changes detected"
	}

	return result
}

// --- Internal helpers ---

type changeWithType struct {
	change      ArtifactChange
	productType ProductType
}

// ClassifySignificance determines the significance of a change.
func ClassifySignificance(pType ProductType, c ArtifactChange) string {
	switch pType {
	case TypeThreadProgression:
		if c.DeltaWords < ThreadDeltaWordThreshold {
			return "" // Below threshold
		}
		if c.ChangeType == "resolved" {
			return SignificanceHigh
		}
		return SignificanceMedium

	case TypeModelUpdate:
		if strings.Contains(strings.ToLower(c.Summary), "contradict") {
			return SignificanceHigh
		}
		return SignificanceMedium

	case TypeModelProbe:
		if strings.Contains(strings.ToLower(c.Summary), "contradict") {
			return SignificanceHigh
		}
		return SignificanceLow

	case TypeDecisionBrief:
		return SignificanceHigh

	default:
		return SignificanceMedium
	}
}

// FormatTitle creates a human-readable title for a digest product.
func FormatTitle(pType ProductType, c ArtifactChange) string {
	base := strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))
	switch pType {
	case TypeThreadProgression:
		return fmt.Sprintf("Thread: %s — %s", Humanize(base), c.ChangeType)
	case TypeModelUpdate:
		return fmt.Sprintf("Model: %s updated", Humanize(base))
	case TypeModelProbe:
		return fmt.Sprintf("Probe: %s", Humanize(base))
	case TypeDecisionBrief:
		return fmt.Sprintf("Decision Brief: %s", Humanize(base))
	default:
		return fmt.Sprintf("Digest: %s", Humanize(base))
	}
}

// ArtifactTypeFromProductType maps product type to source artifact type.
func ArtifactTypeFromProductType(pType ProductType) string {
	switch pType {
	case TypeThreadProgression:
		return "thread"
	case TypeModelUpdate, TypeModelProbe:
		return "model"
	case TypeDecisionBrief:
		return "investigation"
	default:
		return "unknown"
	}
}

// Humanize converts a slug to a human-readable string.
func Humanize(slug string) string {
	parts := strings.SplitN(slug, "-", 5)
	if len(parts) >= 4 {
		if len(parts[0]) == 4 && len(parts[1]) == 2 && len(parts[2]) == 2 {
			slug = strings.Join(parts[3:], " ")
		}
	}
	return strings.ReplaceAll(slug, "-", " ")
}

// SanitizeSlug converts a filename to a URL-safe slug.
func SanitizeSlug(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func mergeHashes(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

// FileHash computes a SHA256 hash of a file's contents.
func FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// --- Default production implementation ---

type defaultService struct {
	projectDir string
}

// NewDefaultService creates a production Service.
func NewDefaultService(projectDir string) Service {
	return &defaultService{projectDir: projectDir}
}

func (s *defaultService) ScanThreads(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	threadsDir := filepath.Join(s.projectDir, ".kb", "threads")
	return s.scanDir(threadsDir, ".kb/threads", hashes)
}

func (s *defaultService) ScanModels(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	modelsDir := filepath.Join(s.projectDir, ".kb", "models")
	return s.scanModelDir(modelsDir, hashes)
}

func (s *defaultService) ScanInvestigations(hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	invDir := filepath.Join(s.projectDir, ".kb", "investigations")
	return s.scanInvestigationsDir(invDir, hashes)
}

func (s *defaultService) scanDir(dir, relPrefix string, hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		if strings.HasPrefix(k, relPrefix) {
			newHashes[k] = v
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newHashes, nil
		}
		return nil, nil, err
	}

	var changes []ArtifactChange
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		relPath := filepath.Join(relPrefix, entry.Name())

		hash, err := FileHash(fullPath)
		if err != nil {
			continue
		}

		oldHash, existed := hashes[relPath]
		newHashes[relPath] = hash

		if !existed {
			wordCount := EstimateWordCount(fullPath)
			changes = append(changes, ArtifactChange{
				Path:       relPath,
				ChangeType: "created",
				DeltaWords: wordCount,
				Summary:    ExtractFirstParagraph(fullPath),
			})
		} else if hash != oldHash {
			wordCount := EstimateWordCount(fullPath)
			changes = append(changes, ArtifactChange{
				Path:       relPath,
				ChangeType: "content_added",
				DeltaWords: wordCount,
				Summary:    ExtractFirstParagraph(fullPath),
			})
		}
	}

	return changes, newHashes, nil
}

func (s *defaultService) scanModelDir(dir string, hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		if strings.HasPrefix(k, ".kb/models") {
			newHashes[k] = v
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newHashes, nil
		}
		return nil, nil, err
	}

	var changes []ArtifactChange
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(dir, entry.Name(), "model.md")
		relPath := filepath.Join(".kb/models", entry.Name(), "model.md")

		hash, err := FileHash(modelPath)
		if err != nil {
			continue
		}

		oldHash, existed := hashes[relPath]
		newHashes[relPath] = hash

		if !existed || hash != oldHash {
			changeType := "modified"
			if !existed {
				changeType = "created"
			}
			changes = append(changes, ArtifactChange{
				Path:       relPath,
				ChangeType: changeType,
				Summary:    ExtractFirstParagraph(modelPath),
			})
		}
	}

	return changes, newHashes, nil
}

func (s *defaultService) scanInvestigationsDir(dir string, hashes map[string]string) ([]ArtifactChange, map[string]string, error) {
	newHashes := make(map[string]string)
	for k, v := range hashes {
		if strings.HasPrefix(k, ".kb/investigations") {
			newHashes[k] = v
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newHashes, nil
		}
		return nil, nil, err
	}

	var changes []ArtifactChange
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		relPath := filepath.Join(".kb/investigations", entry.Name())

		hash, err := FileHash(fullPath)
		if err != nil {
			continue
		}

		oldHash, existed := hashes[relPath]
		newHashes[relPath] = hash

		if hash != oldHash || !existed {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			contentStr := string(content)
			isComplete := strings.Contains(contentStr, "Status: Complete") || strings.Contains(contentStr, "**Status:** Complete")
			hasRecs := strings.Contains(contentStr, "## Recommendations") || strings.Contains(contentStr, "## Implementation Recommendations")

			if isComplete && hasRecs {
				changes = append(changes, ArtifactChange{
					Path:       relPath,
					ChangeType: "completed",
					Summary:    ExtractFirstParagraph(fullPath),
				})
			}
		}
	}

	return changes, newHashes, nil
}

// EstimateWordCount estimates the word count of a file.
func EstimateWordCount(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.Fields(string(data)))
}

// ExtractFirstParagraph extracts the first paragraph from a markdown file.
func ExtractFirstParagraph(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	var paragraph strings.Builder
	inParagraph := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inParagraph {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "**") {
			if inParagraph {
				break
			}
			continue
		}
		inParagraph = true
		if paragraph.Len() > 0 {
			paragraph.WriteString(" ")
		}
		paragraph.WriteString(trimmed)
	}

	result := paragraph.String()
	if len(result) > 300 {
		result = result[:297] + "..."
	}
	return result
}
