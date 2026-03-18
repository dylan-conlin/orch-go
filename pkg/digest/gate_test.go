package digest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFeedbackState_New(t *testing.T) {
	f := NewFeedbackState()
	if f.TypeStats == nil {
		t.Fatal("TypeStats should be initialized")
	}
	if f.Thresholds == nil {
		t.Fatal("Thresholds should be initialized")
	}
}

func TestRecordProduct(t *testing.T) {
	f := NewFeedbackState()
	f.RecordProduct("thread_progression")
	f.RecordProduct("thread_progression")
	f.RecordProduct("model_update")

	if f.TotalProduced() != 3 {
		t.Errorf("TotalProduced = %d, want 3", f.TotalProduced())
	}
	if f.TypeStats["thread_progression"].Produced != 2 {
		t.Errorf("thread_progression.Produced = %d, want 2", f.TypeStats["thread_progression"].Produced)
	}
	if f.TypeStats["model_update"].Produced != 1 {
		t.Errorf("model_update.Produced = %d, want 1", f.TypeStats["model_update"].Produced)
	}
}

func TestRecordRead(t *testing.T) {
	f := NewFeedbackState()
	f.RecordProduct("thread_progression")
	f.RecordProduct("thread_progression")
	f.RecordRead("thread_progression")

	if f.TotalRead() != 1 {
		t.Errorf("TotalRead = %d, want 1", f.TotalRead())
	}
	ts := f.TypeStats["thread_progression"]
	if ts.Read != 1 {
		t.Errorf("Read = %d, want 1", ts.Read)
	}
	if ts.ReadRate() != 0.5 {
		t.Errorf("ReadRate = %f, want 0.5", ts.ReadRate())
	}
}

func TestRecordStar(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 10; i++ {
		f.RecordProduct("decision_brief")
	}
	for i := 0; i < 9; i++ {
		f.RecordStar("decision_brief")
	}

	if f.TotalStarred() != 9 {
		t.Errorf("TotalStarred = %d, want 9", f.TotalStarred())
	}
	ts := f.TypeStats["decision_brief"]
	if ts.Starred != 9 {
		t.Errorf("Starred = %d, want 9", ts.Starred)
	}
	if ts.StarRate() != 0.9 {
		t.Errorf("StarRate = %f, want 0.9", ts.StarRate())
	}
}

func TestReadRateZeroProduced(t *testing.T) {
	ts := &TypeStats{}
	if ts.ReadRate() != 0 {
		t.Errorf("ReadRate with 0 produced = %f, want 0", ts.ReadRate())
	}
	if ts.StarRate() != 0 {
		t.Errorf("StarRate with 0 produced = %f, want 0", ts.StarRate())
	}
}

func TestAdaptThresholds_LowReadRate(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 20; i++ {
		f.RecordProduct("model_probe")
	}
	for i := 0; i < 3; i++ {
		f.RecordRead("model_probe")
	}
	f.TypeStats["model_probe"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)

	f.AdaptThresholds()

	threshold, ok := f.Thresholds["model_probe"]
	if !ok {
		t.Fatal("expected threshold for model_probe")
	}
	if threshold.MinSignificance != "high" {
		t.Errorf("MinSignificance = %q, want %q (low read rate should raise threshold)", threshold.MinSignificance, "high")
	}
}

func TestAdaptThresholds_HighStarRate(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 20; i++ {
		f.RecordProduct("decision_brief")
	}
	for i := 0; i < 17; i++ {
		f.RecordStar("decision_brief")
	}
	f.TypeStats["decision_brief"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)

	f.AdaptThresholds()

	threshold, ok := f.Thresholds["decision_brief"]
	if !ok {
		t.Fatal("expected threshold for decision_brief")
	}
	if threshold.MinSignificance != "low" {
		t.Errorf("MinSignificance = %q, want %q (high star rate should lower threshold)", threshold.MinSignificance, "low")
	}
}

func TestAdaptThresholds_SkipsImmatureTypes(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 3; i++ {
		f.RecordProduct("pattern_alert")
	}

	f.AdaptThresholds()

	if _, ok := f.Thresholds["pattern_alert"]; ok {
		t.Error("should not set threshold for immature type (< 10 products)")
	}
}

func TestAdaptThresholds_SkipsTooRecent(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 20; i++ {
		f.RecordProduct("thread_progression")
	}

	f.AdaptThresholds()

	if _, ok := f.Thresholds["thread_progression"]; ok {
		t.Error("should not set threshold for type seen less than 14 days")
	}
}

func TestAdaptThresholds_NormalReadRate(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 20; i++ {
		f.RecordProduct("model_update")
	}
	for i := 0; i < 10; i++ {
		f.RecordRead("model_update")
	}
	f.TypeStats["model_update"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)

	f.AdaptThresholds()

	threshold, ok := f.Thresholds["model_update"]
	if !ok {
		t.Fatal("expected threshold for model_update")
	}
	if threshold.MinSignificance != "medium" {
		t.Errorf("MinSignificance = %q, want %q (normal rates keep medium)", threshold.MinSignificance, "medium")
	}
}

func TestAdaptThresholds_TracksPreviousThreshold(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 20; i++ {
		f.RecordProduct("model_probe")
	}
	for i := 0; i < 2; i++ {
		f.RecordRead("model_probe")
	}
	f.TypeStats["model_probe"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)
	f.AdaptThresholds()

	// Improve read rate
	for i := 0; i < 20; i++ {
		f.RecordProduct("model_probe")
		f.RecordRead("model_probe")
	}
	f.AdaptThresholds()

	threshold := f.Thresholds["model_probe"]
	if threshold.PreviousThreshold != "high" {
		t.Errorf("PreviousThreshold = %q, want %q", threshold.PreviousThreshold, "high")
	}
}

func TestShouldSurface_NoThreshold(t *testing.T) {
	f := NewFeedbackState()
	if !f.ShouldSurface("thread_progression", "low") {
		t.Error("should surface low significance when no threshold set")
	}
	if !f.ShouldSurface("thread_progression", "medium") {
		t.Error("should surface medium significance when no threshold set")
	}
}

func TestShouldSurface_HighThreshold(t *testing.T) {
	f := NewFeedbackState()
	f.Thresholds["model_probe"] = &AdaptiveThreshold{
		MinSignificance: "high",
		Reason:          "low read rate",
	}

	if f.ShouldSurface("model_probe", "low") {
		t.Error("should NOT surface low when threshold is high")
	}
	if f.ShouldSurface("model_probe", "medium") {
		t.Error("should NOT surface medium when threshold is high")
	}
	if !f.ShouldSurface("model_probe", "high") {
		t.Error("should surface high when threshold is high")
	}
}

func TestShouldSurface_MediumThreshold(t *testing.T) {
	f := NewFeedbackState()
	f.Thresholds["model_update"] = &AdaptiveThreshold{
		MinSignificance: "medium",
	}

	if f.ShouldSurface("model_update", "low") {
		t.Error("should NOT surface low when threshold is medium")
	}
	if !f.ShouldSurface("model_update", "medium") {
		t.Error("should surface medium when threshold is medium")
	}
	if !f.ShouldSurface("model_update", "high") {
		t.Error("should surface high when threshold is medium")
	}
}

func TestShouldSurface_LowThreshold(t *testing.T) {
	f := NewFeedbackState()
	f.Thresholds["decision_brief"] = &AdaptiveThreshold{
		MinSignificance: "low",
	}

	if !f.ShouldSurface("decision_brief", "low") {
		t.Error("should surface low when threshold is low")
	}
}

func TestSignificanceLevelFn(t *testing.T) {
	tests := []struct {
		sig  string
		want int
	}{
		{"low", 1},
		{"medium", 2},
		{"high", 3},
		{"unknown", 0},
	}
	for _, tt := range tests {
		if got := SignificanceLevel(tt.sig); got != tt.want {
			t.Errorf("SignificanceLevel(%q) = %d, want %d", tt.sig, got, tt.want)
		}
	}
}

func TestReadRateByType(t *testing.T) {
	f := NewFeedbackState()
	for i := 0; i < 10; i++ {
		f.RecordProduct("thread_progression")
	}
	for i := 0; i < 7; i++ {
		f.RecordRead("thread_progression")
	}
	for i := 0; i < 5; i++ {
		f.RecordProduct("model_update")
	}
	f.RecordRead("model_update")

	rates := f.ReadRateByType()
	if rates["thread_progression"] != 0.7 {
		t.Errorf("thread_progression rate = %f, want 0.7", rates["thread_progression"])
	}
	if rates["model_update"] != 0.2 {
		t.Errorf("model_update rate = %f, want 0.2", rates["model_update"])
	}
}

func TestFeedbackState_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feedback.json")

	f := NewFeedbackState()
	f.RecordProduct("thread_progression")
	f.RecordProduct("thread_progression")
	f.RecordRead("thread_progression")
	f.RecordStar("thread_progression")
	f.Thresholds["model_probe"] = &AdaptiveThreshold{
		MinSignificance: "high",
		Reason:          "low read rate (15%)",
	}

	if err := SaveFeedbackState(path, f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadFeedbackState(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.TotalProduced() != 2 {
		t.Errorf("loaded TotalProduced = %d, want 2", loaded.TotalProduced())
	}
	if loaded.TotalRead() != 1 {
		t.Errorf("loaded TotalRead = %d, want 1", loaded.TotalRead())
	}
	if loaded.TotalStarred() != 1 {
		t.Errorf("loaded TotalStarred = %d, want 1", loaded.TotalStarred())
	}
	ts := loaded.TypeStats["thread_progression"]
	if ts == nil || ts.Produced != 2 {
		t.Error("loaded TypeStats missing or wrong")
	}
	th := loaded.Thresholds["model_probe"]
	if th == nil || th.MinSignificance != "high" {
		t.Error("loaded Thresholds missing or wrong")
	}
}

func TestLoadFeedbackState_NotExists(t *testing.T) {
	f, err := LoadFeedbackState("/nonexistent/path/state.json")
	if err != nil {
		t.Fatalf("should not error for missing file: %v", err)
	}
	if f.TypeStats == nil {
		t.Error("should return initialized state")
	}
}

func TestLoadFeedbackState_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{invalid"), 0644)

	_, err := LoadFeedbackState(path)
	if err == nil {
		t.Error("should error on invalid JSON")
	}
}

func TestFeedbackState_JSON(t *testing.T) {
	f := NewFeedbackState()
	f.RecordProduct("thread_progression")
	f.RecordRead("thread_progression")

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	if _, ok := raw["type_stats"]; !ok {
		t.Error("JSON should have type_stats field")
	}
	if _, ok := raw["thresholds"]; !ok {
		t.Error("JSON should have thresholds field")
	}
}

func TestFeedbackState_Integration(t *testing.T) {
	f := NewFeedbackState()

	// Simulate 2 weeks of model_probe products with low engagement
	for i := 0; i < 15; i++ {
		f.RecordProduct("model_probe")
	}
	for i := 0; i < 2; i++ {
		f.RecordRead("model_probe")
	}
	f.TypeStats["model_probe"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)

	// Simulate 2 weeks of decision_brief products with high engagement
	for i := 0; i < 12; i++ {
		f.RecordProduct("decision_brief")
	}
	for i := 0; i < 11; i++ {
		f.RecordStar("decision_brief")
	}
	f.TypeStats["decision_brief"].FirstSeen = time.Now().Add(-15 * 24 * time.Hour)

	f.AdaptThresholds()

	// Low-engagement type: only high significance surfaces
	if f.ShouldSurface("model_probe", "low") {
		t.Error("model_probe low should be filtered")
	}
	if f.ShouldSurface("model_probe", "medium") {
		t.Error("model_probe medium should be filtered")
	}
	if !f.ShouldSurface("model_probe", "high") {
		t.Error("model_probe high should surface")
	}

	// High-engagement type: everything surfaces
	if !f.ShouldSurface("decision_brief", "low") {
		t.Error("decision_brief low should surface (high star rate)")
	}

	// Unknown type: no threshold, everything surfaces
	if !f.ShouldSurface("weekly_digest", "low") {
		t.Error("unknown type should surface everything")
	}
}
