package account

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NotExist(t *testing.T) {
	// Save original path and restore after test
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if len(cfg.Accounts) != 0 {
		t.Errorf("LoadConfig() accounts = %d, want 0", len(cfg.Accounts))
	}
}

func TestConfigOperations(t *testing.T) {
	cfg := &Config{
		Accounts: make(map[string]Account),
	}

	// Test Save
	acc := Account{
		Email:        "test@example.com",
		RefreshToken: "test-token",
		Source:       "saved",
	}
	cfg.Save("test", acc, true)

	if cfg.Default != "test" {
		t.Errorf("After Save with setDefault=true, Default = %q, want %q", cfg.Default, "test")
	}

	// Test Get
	got, err := cfg.Get("test")
	if err != nil {
		t.Errorf("Get(test) error = %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Get(test).Email = %q, want %q", got.Email, "test@example.com")
	}

	// Test Get not found
	_, err = cfg.Get("notexist")
	if err == nil {
		t.Error("Get(notexist) should return error")
	}

	// Test List
	names := cfg.List()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("List() = %v, want [test]", names)
	}

	// Test Remove
	err = cfg.Remove("test")
	if err != nil {
		t.Errorf("Remove(test) error = %v", err)
	}

	if len(cfg.Accounts) != 0 {
		t.Errorf("After Remove, accounts = %d, want 0", len(cfg.Accounts))
	}

	// Default should be cleared
	if cfg.Default != "" {
		t.Errorf("After Remove, Default = %q, want empty", cfg.Default)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create .orch directory
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"personal": {
				Email:        "user@example.com",
				RefreshToken: "token123",
				Source:       "saved",
			},
		},
		Default: "personal",
	}

	// Save
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Default != "personal" {
		t.Errorf("Loaded Default = %q, want %q", loaded.Default, "personal")
	}

	acc, ok := loaded.Accounts["personal"]
	if !ok {
		t.Fatal("Loaded config missing 'personal' account")
	}

	if acc.Email != "user@example.com" {
		t.Errorf("Loaded Email = %q, want %q", acc.Email, "user@example.com")
	}
}

// ============================================================================
// Capacity Info Tests
// ============================================================================

func TestCapacityInfo_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		capacity CapacityInfo
		want     bool
	}{
		{
			name:     "healthy capacity",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60},
			want:     true,
		},
		{
			name:     "low 5-hour",
			capacity: CapacityInfo{FiveHourRemaining: 15, SevenDayRemaining: 60},
			want:     false,
		},
		{
			name:     "low 7-day",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 15},
			want:     false,
		},
		{
			name:     "both low",
			capacity: CapacityInfo{FiveHourRemaining: 10, SevenDayRemaining: 10},
			want:     false,
		},
		{
			name:     "with error",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60, Error: "some error"},
			want:     false,
		},
		{
			name:     "exactly at threshold",
			capacity: CapacityInfo{FiveHourRemaining: 20, SevenDayRemaining: 20},
			want:     false, // threshold is >20, not >=20
		},
		{
			name:     "just above threshold",
			capacity: CapacityInfo{FiveHourRemaining: 21, SevenDayRemaining: 21},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.capacity.IsHealthy(); got != tt.want {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapacityInfo_IsLow(t *testing.T) {
	tests := []struct {
		name     string
		capacity CapacityInfo
		want     bool
	}{
		{
			name:     "healthy capacity",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60},
			want:     false,
		},
		{
			name:     "low 5-hour",
			capacity: CapacityInfo{FiveHourRemaining: 15, SevenDayRemaining: 60},
			want:     true,
		},
		{
			name:     "low 7-day",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 15},
			want:     true,
		},
		{
			name:     "with error",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60, Error: "some error"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.capacity.IsLow(); got != tt.want {
				t.Errorf("IsLow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapacityInfo_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		capacity CapacityInfo
		want     bool
	}{
		{
			name:     "healthy capacity",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60},
			want:     false,
		},
		{
			name:     "low but not critical",
			capacity: CapacityInfo{FiveHourRemaining: 10, SevenDayRemaining: 10},
			want:     false,
		},
		{
			name:     "critical 5-hour",
			capacity: CapacityInfo{FiveHourRemaining: 3, SevenDayRemaining: 60},
			want:     true,
		},
		{
			name:     "critical 7-day",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 2},
			want:     true,
		},
		{
			name:     "with error",
			capacity: CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 60, Error: "some error"},
			want:     true,
		},
		{
			name:     "exactly at threshold",
			capacity: CapacityInfo{FiveHourRemaining: 5, SevenDayRemaining: 5},
			want:     false, // threshold is <5, not <=5
		},
		{
			name:     "just below threshold",
			capacity: CapacityInfo{FiveHourRemaining: 4.9, SevenDayRemaining: 50},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.capacity.IsCritical(); got != tt.want {
				t.Errorf("IsCritical() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentCapacity_NoAuthFile(t *testing.T) {
	// Use temp directory with no auth file
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	capacity, err := GetCurrentCapacity()
	if err == nil {
		t.Error("GetCurrentCapacity() should return error when no auth file exists")
	}

	if capacity == nil {
		t.Fatal("GetCurrentCapacity() should return CapacityInfo even on error")
	}

	if capacity.Error == "" {
		t.Error("CapacityInfo.Error should be set when auth file doesn't exist")
	}
}

func TestGetAccountCapacity_NotFound(t *testing.T) {
	// Use temp directory with empty config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	capacity, err := GetAccountCapacity("nonexistent")
	if err == nil {
		t.Error("GetAccountCapacity() should return error for nonexistent account")
	}

	if capacity == nil {
		t.Fatal("GetAccountCapacity() should return CapacityInfo even on error")
	}

	if capacity.Error == "" {
		t.Error("CapacityInfo.Error should be set for nonexistent account")
	}
}

func TestCapacityError(t *testing.T) {
	err := &CapacityError{Message: "test error message"}
	if got := err.Error(); got != "test error message" {
		t.Errorf("CapacityError.Error() = %q, want %q", got, "test error message")
	}
}

// ============================================================================
// Auto-Switch Tests
// ============================================================================

func TestDefaultAutoSwitchThresholds(t *testing.T) {
	thresholds := DefaultAutoSwitchThresholds()

	if thresholds.FiveHourThreshold != 80 {
		t.Errorf("FiveHourThreshold = %v, want 80", thresholds.FiveHourThreshold)
	}

	if thresholds.WeeklyThreshold != 90 {
		t.Errorf("WeeklyThreshold = %v, want 90", thresholds.WeeklyThreshold)
	}

	if thresholds.MinHeadroomDelta != 10 {
		t.Errorf("MinHeadroomDelta = %v, want 10", thresholds.MinHeadroomDelta)
	}
}

func TestAutoSwitchResult_NoSwitch(t *testing.T) {
	result := &AutoSwitchResult{
		Switched: false,
		Reason:   "current account healthy",
	}

	if result.Switched {
		t.Error("Switched should be false")
	}

	if result.ToAccount != "" {
		t.Errorf("ToAccount should be empty, got %q", result.ToAccount)
	}
}

func TestAutoSwitchResult_WithSwitch(t *testing.T) {
	result := &AutoSwitchResult{
		Switched:    true,
		FromAccount: "personal",
		ToAccount:   "work",
		Reason:      "switching due to low headroom",
		FromCapacity: &CapacityInfo{
			FiveHourUsed: 85,
			SevenDayUsed: 92,
		},
		ToCapacity: &CapacityInfo{
			FiveHourUsed: 20,
			SevenDayUsed: 30,
		},
	}

	if !result.Switched {
		t.Error("Switched should be true")
	}

	if result.FromAccount != "personal" {
		t.Errorf("FromAccount = %q, want %q", result.FromAccount, "personal")
	}

	if result.ToAccount != "work" {
		t.Errorf("ToAccount = %q, want %q", result.ToAccount, "work")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{10, 20, 10},
		{20, 10, 10},
		{15, 15, 15},
		{0, 100, 0},
		{-10, 10, -10},
	}

	for _, tt := range tests {
		if got := min(tt.a, tt.b); got != tt.want {
			t.Errorf("min(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
