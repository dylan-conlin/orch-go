// Package daemon provides autonomous overnight processing capabilities.
// Digest produces consumable thinking products from KB artifact changes.
// The daemon scans .kb/threads/, .kb/models/, and .kb/investigations/
// for changes and packages notable ones as product files in ~/.orch/digest/.
package daemon

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
	DigestTypeThreadProgression DigestProductType = "thread_progression"
	DigestTypeModelUpdate       DigestProductType = "model_update"
	DigestTypeModelProbe        DigestProductType = "model_probe"
	DigestTypeDecisionBrief     DigestProductType = "decision_brief"
)

// Product state constants.
const (
	DigestStateNew      DigestProductState = "new"
	DigestStateRead     DigestProductState = "read"
	DigestStateStarred  DigestProductState = "starred"
	DigestStateArchived DigestProductState = "archived"
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

// DigestProductType is the type of digest product.
type DigestProductType string

// DigestProductState is the lifecycle state of a digest product.
type DigestProductState string

// DigestProduct is a single thinking product written to ~/.orch/digest/.
type DigestProduct struct {
	ID           string             `json:"id"`
	Type         DigestProductType  `json:"type"`
	Title        string             `json:"title"`
	Summary      string             `json:"summary"`
	Significance string             `json:"significance"`
	Source       DigestSource       `json:"source"`
	State        DigestProductState `json:"state"`
	CreatedAt    time.Time          `json:"created_at"`
	ReadAt       time.Time          `json:"read_at,omitempty"`
	StarredAt    time.Time          `json:"starred_at,omitempty"`
	ArchivedAt   time.Time          `json:"archived_at,omitempty"`
}

// DigestSource is the source artifact that triggered the product.
type DigestSource struct {
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	ChangeType   string `json:"change_type"`
	DeltaWords   int    `json:"delta_words,omitempty"`
}

// DigestState tracks what's been scanned to avoid duplicate products.
type DigestState struct {
	LastScan   time.Time         `json:"last_scan"`
	FileHashes map[string]string `json:"file_hashes"`
	Stats      DigestStats       `json:"stats"`
}

// DigestStats tracks aggregate statistics for the digest system.
type DigestStats struct {
	TotalProduced int `json:"total_produced"`
	TotalRead     int `json:"total_read"`
	TotalStarred  int `json:"total_starred"`
}

// DigestStatsResponse is the API response for /api/digest/stats.
type DigestStatsResponse struct {
	Unread  int `json:"unread"`
	Read    int `json:"read"`
	Starred int `json:"starred"`
	Total   int `json:"total"`
}

// DigestArtifactChange represents a detected change in a KB artifact.
type DigestArtifactChange struct {
	Path       string
	ChangeType string // "content_added", "created", "modified"
	DeltaWords int
	Summary    string
}

// DigestService provides I/O operations for digest scanning.
type DigestService interface {
	// ScanThreads scans .kb/threads/ for changes since the given hashes.
	ScanThreads(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error)
	// ScanModels scans .kb/models/ for changes since the given hashes.
	ScanModels(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error)
	// ScanInvestigations scans .kb/investigations/ for completed investigations.
	ScanInvestigations(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error)
}

// DigestResult contains the result of a periodic digest scan.
type DigestResult struct {
	Produced int
	Skipped  int
	Scanned  int
	Message  string
	Error    error
}

// DigestListOpts controls filtering for DigestStore.List.
type DigestListOpts struct {
	State DigestProductState
	Type  DigestProductType
	Limit int
}

// --- Digest Store (filesystem-backed product storage) ---

// DigestStore reads and writes digest product files.
type DigestStore struct {
	dir string
}

// NewDigestStore creates a DigestStore backed by the given directory.
func NewDigestStore(dir string) *DigestStore {
	return &DigestStore{dir: dir}
}

// Write persists a DigestProduct to a JSON file.
func (s *DigestStore) Write(p DigestProduct) error {
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
func (s *DigestStore) List(opts DigestListOpts) ([]DigestProduct, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read digest dir: %w", err)
	}

	var products []DigestProduct
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var p DigestProduct
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
func (s *DigestStore) Get(id string) (*DigestProduct, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read product %s: %w", id, err)
	}
	var p DigestProduct
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal product %s: %w", id, err)
	}
	return &p, nil
}

// UpdateState transitions a product to a new state.
func (s *DigestStore) UpdateState(id string, state DigestProductState) error {
	p, err := s.Get(id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	p.State = state
	switch state {
	case DigestStateRead:
		p.ReadAt = now
	case DigestStateStarred:
		p.StarredAt = now
	case DigestStateArchived:
		p.ArchivedAt = now
	}

	return s.Write(*p)
}

// Stats returns aggregate statistics about products.
func (s *DigestStore) Stats() (DigestStatsResponse, error) {
	products, err := s.List(DigestListOpts{})
	if err != nil {
		return DigestStatsResponse{}, err
	}

	var stats DigestStatsResponse
	stats.Total = len(products)
	for _, p := range products {
		switch p.State {
		case DigestStateNew:
			stats.Unread++
		case DigestStateRead:
			stats.Read++
		case DigestStateStarred:
			stats.Starred++
		}
	}
	return stats, nil
}

// ArchiveRead archives read products older than the given duration.
// Returns the number of products archived.
func (s *DigestStore) ArchiveRead(olderThan time.Duration) (int, error) {
	products, err := s.List(DigestListOpts{State: DigestStateRead})
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().UTC().Add(-olderThan)
	archived := 0
	for _, p := range products {
		if p.CreatedAt.Before(cutoff) {
			if err := s.UpdateState(p.ID, DigestStateArchived); err != nil {
				continue
			}
			archived++
		}
	}
	return archived, nil
}

// --- Digest State persistence ---

// LoadDigestState loads the digest scan state from disk.
// Returns an empty state (not error) if the file doesn't exist.
func LoadDigestState(path string) (*DigestState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DigestState{FileHashes: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("read digest state: %w", err)
	}

	var state DigestState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal digest state: %w", err)
	}
	if state.FileHashes == nil {
		state.FileHashes = make(map[string]string)
	}
	return &state, nil
}

// SaveDigestState persists the digest scan state to disk.
func SaveDigestState(path string, state *DigestState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal digest state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Daemon periodic task ---

// RunPeriodicDigest scans KB artifacts for changes and creates digest products.
func (d *Daemon) RunPeriodicDigest() *DigestResult {
	if !d.Scheduler.IsDue(TaskDigest) {
		return nil
	}

	svc := d.Digest
	if svc == nil {
		return &DigestResult{
			Error:   fmt.Errorf("digest service not configured"),
			Message: "Digest: service not configured",
		}
	}

	// Load scan state
	state, err := LoadDigestState(d.DigestStatePath)
	if err != nil {
		return &DigestResult{
			Error:   err,
			Message: fmt.Sprintf("Digest: failed to load state: %v", err),
		}
	}

	store := NewDigestStore(d.DigestDir)
	result := &DigestResult{}

	// Scan all artifact types
	var allChanges []digestChangeWithType

	threadChanges, threadHashes, err := svc.ScanThreads(state.FileHashes)
	if err == nil {
		for _, c := range threadChanges {
			allChanges = append(allChanges, digestChangeWithType{change: c, productType: DigestTypeThreadProgression})
		}
		mergeHashes(state.FileHashes, threadHashes)
	}

	modelChanges, modelHashes, err := svc.ScanModels(state.FileHashes)
	if err == nil {
		for _, c := range modelChanges {
			allChanges = append(allChanges, digestChangeWithType{change: c, productType: DigestTypeModelUpdate})
		}
		mergeHashes(state.FileHashes, modelHashes)
	}

	invChanges, invHashes, err := svc.ScanInvestigations(state.FileHashes)
	if err == nil {
		for _, c := range invChanges {
			allChanges = append(allChanges, digestChangeWithType{change: c, productType: DigestTypeDecisionBrief})
		}
		mergeHashes(state.FileHashes, invHashes)
	}

	result.Scanned = len(allChanges)

	// Create products for significant changes
	now := time.Now().UTC()
	for _, cwt := range allChanges {
		c := cwt.change
		significance := classifySignificance(cwt.productType, c)
		if significance == "" {
			result.Skipped++
			continue
		}

		id := fmt.Sprintf("%s-%s-%s",
			now.Format("20060102T1504"),
			string(cwt.productType),
			sanitizeSlug(filepath.Base(c.Path)),
		)

		title := formatTitle(cwt.productType, c)

		p := DigestProduct{
			ID:           id,
			Type:         cwt.productType,
			Title:        title,
			Summary:      c.Summary,
			Significance: significance,
			Source: DigestSource{
				ArtifactType: artifactTypeFromProductType(cwt.productType),
				Path:         c.Path,
				ChangeType:   c.ChangeType,
				DeltaWords:   c.DeltaWords,
			},
			State:     DigestStateNew,
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
	if err := SaveDigestState(d.DigestStatePath, state); err != nil {
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

	d.Scheduler.MarkRun(TaskDigest)
	return result
}

// --- Internal helpers ---

type digestChangeWithType struct {
	change      DigestArtifactChange
	productType DigestProductType
}

func classifySignificance(pType DigestProductType, c DigestArtifactChange) string {
	switch pType {
	case DigestTypeThreadProgression:
		if c.DeltaWords < ThreadDeltaWordThreshold {
			return "" // Below threshold
		}
		if c.ChangeType == "resolved" {
			return SignificanceHigh
		}
		return SignificanceMedium

	case DigestTypeModelUpdate:
		if strings.Contains(strings.ToLower(c.Summary), "contradict") {
			return SignificanceHigh
		}
		return SignificanceMedium

	case DigestTypeModelProbe:
		if strings.Contains(strings.ToLower(c.Summary), "contradict") {
			return SignificanceHigh
		}
		return SignificanceLow

	case DigestTypeDecisionBrief:
		return SignificanceHigh

	default:
		return SignificanceMedium
	}
}

func formatTitle(pType DigestProductType, c DigestArtifactChange) string {
	base := strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))
	switch pType {
	case DigestTypeThreadProgression:
		return fmt.Sprintf("Thread: %s — %s", humanize(base), c.ChangeType)
	case DigestTypeModelUpdate:
		return fmt.Sprintf("Model: %s updated", humanize(base))
	case DigestTypeModelProbe:
		return fmt.Sprintf("Probe: %s", humanize(base))
	case DigestTypeDecisionBrief:
		return fmt.Sprintf("Decision Brief: %s", humanize(base))
	default:
		return fmt.Sprintf("Digest: %s", humanize(base))
	}
}

func artifactTypeFromProductType(pType DigestProductType) string {
	switch pType {
	case DigestTypeThreadProgression:
		return "thread"
	case DigestTypeModelUpdate, DigestTypeModelProbe:
		return "model"
	case DigestTypeDecisionBrief:
		return "investigation"
	default:
		return "unknown"
	}
}

func humanize(slug string) string {
	parts := strings.SplitN(slug, "-", 5)
	if len(parts) >= 4 {
		if len(parts[0]) == 4 && len(parts[1]) == 2 && len(parts[2]) == 2 {
			slug = strings.Join(parts[3:], " ")
		}
	}
	return strings.ReplaceAll(slug, "-", " ")
}

func sanitizeSlug(name string) string {
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

// fileHash computes a SHA256 hash of a file's contents.
func fileHash(path string) (string, error) {
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

type defaultDigestService struct {
	projectDir string
}

// NewDefaultDigestService creates a production DigestService.
func NewDefaultDigestService(projectDir string) DigestService {
	return &defaultDigestService{projectDir: projectDir}
}

func (s *defaultDigestService) ScanThreads(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
	threadsDir := filepath.Join(s.projectDir, ".kb", "threads")
	return s.scanDir(threadsDir, ".kb/threads", hashes)
}

func (s *defaultDigestService) ScanModels(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
	modelsDir := filepath.Join(s.projectDir, ".kb", "models")
	return s.scanModelDir(modelsDir, hashes)
}

func (s *defaultDigestService) ScanInvestigations(hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
	invDir := filepath.Join(s.projectDir, ".kb", "investigations")
	return s.scanInvestigationsDir(invDir, hashes)
}

func (s *defaultDigestService) scanDir(dir, relPrefix string, hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
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

	var changes []DigestArtifactChange
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		relPath := filepath.Join(relPrefix, entry.Name())

		hash, err := fileHash(fullPath)
		if err != nil {
			continue
		}

		oldHash, existed := hashes[relPath]
		newHashes[relPath] = hash

		if !existed {
			wordCount := estimateWordCount(fullPath)
			changes = append(changes, DigestArtifactChange{
				Path:       relPath,
				ChangeType: "created",
				DeltaWords: wordCount,
				Summary:    extractFirstParagraph(fullPath),
			})
		} else if hash != oldHash {
			wordCount := estimateWordCount(fullPath)
			changes = append(changes, DigestArtifactChange{
				Path:       relPath,
				ChangeType: "content_added",
				DeltaWords: wordCount,
				Summary:    extractFirstParagraph(fullPath),
			})
		}
	}

	return changes, newHashes, nil
}

func (s *defaultDigestService) scanModelDir(dir string, hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
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

	var changes []DigestArtifactChange
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(dir, entry.Name(), "model.md")
		relPath := filepath.Join(".kb/models", entry.Name(), "model.md")

		hash, err := fileHash(modelPath)
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
			changes = append(changes, DigestArtifactChange{
				Path:       relPath,
				ChangeType: changeType,
				Summary:    extractFirstParagraph(modelPath),
			})
		}
	}

	return changes, newHashes, nil
}

func (s *defaultDigestService) scanInvestigationsDir(dir string, hashes map[string]string) ([]DigestArtifactChange, map[string]string, error) {
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

	var changes []DigestArtifactChange
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		relPath := filepath.Join(".kb/investigations", entry.Name())

		hash, err := fileHash(fullPath)
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
				changes = append(changes, DigestArtifactChange{
					Path:       relPath,
					ChangeType: "completed",
					Summary:    extractFirstParagraph(fullPath),
				})
			}
		}
	}

	return changes, newHashes, nil
}

func estimateWordCount(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.Fields(string(data)))
}

func extractFirstParagraph(path string) string {
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
