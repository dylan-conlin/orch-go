package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DigestTypeStats tracks per-type feedback metrics for the adaptive quality gate.
type DigestTypeStats struct {
	Produced  int       `json:"produced"`
	Read      int       `json:"read"`
	Starred   int       `json:"starred"`
	FirstSeen time.Time `json:"first_seen"`
}

// ReadRate returns the fraction of produced products that were read.
func (ts *DigestTypeStats) ReadRate() float64 {
	if ts.Produced == 0 {
		return 0
	}
	return float64(ts.Read) / float64(ts.Produced)
}

// StarRate returns the fraction of produced products that were starred.
func (ts *DigestTypeStats) StarRate() float64 {
	if ts.Produced == 0 {
		return 0
	}
	return float64(ts.Starred) / float64(ts.Produced)
}

// AdaptiveThreshold controls what significance level is required to surface a product type.
type AdaptiveThreshold struct {
	MinSignificance   string    `json:"min_significance"`
	Reason            string    `json:"reason"`
	LastAdaptedAt     time.Time `json:"last_adapted_at"`
	PreviousThreshold string    `json:"previous_threshold,omitempty"`
}

const (
	// MinProductsForAdaptation is the minimum sample size before adapting thresholds.
	MinProductsForAdaptation = 10
	// MaturityWindowDays is how long a product type must exist before adapting.
	MaturityWindowDays = 14
	// LowReadRateThreshold triggers raising significance requirements.
	LowReadRateThreshold = 0.20
	// HighStarRateThreshold triggers lowering significance requirements.
	HighStarRateThreshold = 0.80
)

// DigestFeedbackState tracks read/star feedback and adaptive thresholds for digest products.
// Stored alongside scan state in digest-state.json.
type DigestFeedbackState struct {
	TypeStats  map[string]*DigestTypeStats   `json:"type_stats"`
	Thresholds map[string]*AdaptiveThreshold `json:"thresholds"`
}

// NewDigestFeedbackState creates an initialized feedback state.
func NewDigestFeedbackState() *DigestFeedbackState {
	return &DigestFeedbackState{
		TypeStats:  make(map[string]*DigestTypeStats),
		Thresholds: make(map[string]*AdaptiveThreshold),
	}
}

// RecordProduct increments the produced counter for a product type.
func (f *DigestFeedbackState) RecordProduct(productType string) {
	ts := f.getOrCreateType(productType)
	ts.Produced++
}

// RecordRead increments the read counter for a product type.
func (f *DigestFeedbackState) RecordRead(productType string) {
	ts := f.getOrCreateType(productType)
	ts.Read++
}

// RecordStar increments the starred counter for a product type.
func (f *DigestFeedbackState) RecordStar(productType string) {
	ts := f.getOrCreateType(productType)
	ts.Starred++
}

func (f *DigestFeedbackState) getOrCreateType(productType string) *DigestTypeStats {
	if f.TypeStats == nil {
		f.TypeStats = make(map[string]*DigestTypeStats)
	}
	ts, ok := f.TypeStats[productType]
	if !ok {
		ts = &DigestTypeStats{FirstSeen: time.Now()}
		f.TypeStats[productType] = ts
	}
	return ts
}

// AdaptThresholds adjusts significance thresholds based on read/star feedback.
// Only adapts types with >= MinProductsForAdaptation products and >= MaturityWindowDays age.
//
// Rules from architect design:
//   - Types with < 20% read rate → raise to "high" (surface less)
//   - Types with > 80% star rate → lower to "low" (surface more)
//   - Otherwise → set to "medium" (default)
func (f *DigestFeedbackState) AdaptThresholds() {
	if f.Thresholds == nil {
		f.Thresholds = make(map[string]*AdaptiveThreshold)
	}

	now := time.Now()
	maturityCutoff := now.Add(-MaturityWindowDays * 24 * time.Hour)

	for typeName, ts := range f.TypeStats {
		if ts.Produced < MinProductsForAdaptation {
			continue
		}
		if ts.FirstSeen.After(maturityCutoff) {
			continue
		}

		newSignificance := "medium"
		reason := fmt.Sprintf("read_rate=%.0f%% star_rate=%.0f%%", ts.ReadRate()*100, ts.StarRate()*100)

		// Check star rate first: starring implies engagement even if read tracking
		// is incomplete (user may star without explicitly marking as read).
		if ts.StarRate() > HighStarRateThreshold {
			newSignificance = "low"
			reason = fmt.Sprintf("high star rate (%.0f%% > %d%%)", ts.StarRate()*100, int(HighStarRateThreshold*100))
		} else if ts.ReadRate() < LowReadRateThreshold {
			newSignificance = "high"
			reason = fmt.Sprintf("low read rate (%.0f%% < %d%%)", ts.ReadRate()*100, int(LowReadRateThreshold*100))
		}

		prev := ""
		if existing, ok := f.Thresholds[typeName]; ok {
			prev = existing.MinSignificance
		}

		f.Thresholds[typeName] = &AdaptiveThreshold{
			MinSignificance:   newSignificance,
			Reason:            reason,
			LastAdaptedAt:     now,
			PreviousThreshold: prev,
		}
	}
}

// ShouldSurface returns true if a product with the given type and significance
// should be surfaced to the user based on current adaptive thresholds.
func (f *DigestFeedbackState) ShouldSurface(productType, significance string) bool {
	if f.Thresholds == nil {
		return true
	}
	threshold, ok := f.Thresholds[productType]
	if !ok {
		return true
	}
	return significanceLevel(significance) >= significanceLevel(threshold.MinSignificance)
}

// significanceLevel converts significance string to comparable int.
func significanceLevel(sig string) int {
	switch sig {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		return 0
	}
}

// ReadRateByType returns read rates for all tracked types.
func (f *DigestFeedbackState) ReadRateByType() map[string]float64 {
	rates := make(map[string]float64)
	for name, ts := range f.TypeStats {
		rates[name] = ts.ReadRate()
	}
	return rates
}

// TotalProduced returns the total number of products produced across all types.
func (f *DigestFeedbackState) TotalProduced() int {
	total := 0
	for _, ts := range f.TypeStats {
		total += ts.Produced
	}
	return total
}

// TotalRead returns the total number of products read across all types.
func (f *DigestFeedbackState) TotalRead() int {
	total := 0
	for _, ts := range f.TypeStats {
		total += ts.Read
	}
	return total
}

// TotalStarred returns the total number of products starred across all types.
func (f *DigestFeedbackState) TotalStarred() int {
	total := 0
	for _, ts := range f.TypeStats {
		total += ts.Starred
	}
	return total
}

// DigestFeedbackStatePath returns the default path for digest feedback state.
func DigestFeedbackStatePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".orch", "digest-feedback.json")
}

// SaveDigestFeedbackState writes feedback state to the given path.
func SaveDigestFeedbackState(path string, f *DigestFeedbackState) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create digest feedback dir: %w", err)
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal digest feedback state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write digest feedback state: %w", err)
	}
	return nil
}

// LoadDigestFeedbackState reads feedback state from the given path.
// Returns a new initialized state if the file doesn't exist.
func LoadDigestFeedbackState(path string) (*DigestFeedbackState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewDigestFeedbackState(), nil
		}
		return nil, fmt.Errorf("read digest feedback state: %w", err)
	}

	var f DigestFeedbackState
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse digest feedback state: %w", err)
	}

	if f.TypeStats == nil {
		f.TypeStats = make(map[string]*DigestTypeStats)
	}
	if f.Thresholds == nil {
		f.Thresholds = make(map[string]*AdaptiveThreshold)
	}
	return &f, nil
}
