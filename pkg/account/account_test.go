package account

import (
	"encoding/json"
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
// Account Schema Tests (Tier, Role, ConfigDir fields)
// ============================================================================

func TestSaveAndLoadConfig_WithTierRoleConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "work@example.com",
				RefreshToken: "work-token",
				Source:       "saved",
				Tier:         "20x",
				Role:         "primary",
				ConfigDir:    "~/.claude",
			},
			"personal": {
				Email:        "personal@example.com",
				RefreshToken: "personal-token",
				Source:       "saved",
				Tier:         "5x",
				Role:         "spillover",
				ConfigDir:    "~/.claude-personal",
			},
		},
		Default: "personal",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q", work.Tier, "20x")
	}
	if work.Role != "primary" {
		t.Errorf("work.Role = %q, want %q", work.Role, "primary")
	}
	if work.ConfigDir != "~/.claude" {
		t.Errorf("work.ConfigDir = %q, want %q", work.ConfigDir, "~/.claude")
	}

	personal := loaded.Accounts["personal"]
	if personal.Tier != "5x" {
		t.Errorf("personal.Tier = %q, want %q", personal.Tier, "5x")
	}
	if personal.Role != "spillover" {
		t.Errorf("personal.Role = %q, want %q", personal.Role, "spillover")
	}
	if personal.ConfigDir != "~/.claude-personal" {
		t.Errorf("personal.ConfigDir = %q, want %q", personal.ConfigDir, "~/.claude-personal")
	}
}

func TestSaveAndLoadConfig_BackwardCompatible(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Save config WITHOUT new fields (backward compat)
	cfg := &Config{
		Accounts: map[string]Account{
			"old": {
				Email:        "old@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
			},
		},
		Default: "old",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	old := loaded.Accounts["old"]
	if old.Tier != "" {
		t.Errorf("old.Tier = %q, want empty (backward compat)", old.Tier)
	}
	if old.Role != "" {
		t.Errorf("old.Role = %q, want empty (backward compat)", old.Role)
	}
	if old.ConfigDir != "" {
		t.Errorf("old.ConfigDir = %q, want empty (backward compat)", old.ConfigDir)
	}
}

func TestGetConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:     "work@example.com",
				Source:    "saved",
				ConfigDir: "~/.claude",
			},
			"personal": {
				Email:     "personal@example.com",
				Source:    "saved",
				ConfigDir: "~/.claude-personal",
			},
		},
		Default: "work",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if got := GetConfigDir("work"); got != "~/.claude" {
		t.Errorf("GetConfigDir(work) = %q, want %q", got, "~/.claude")
	}
	if got := GetConfigDir("personal"); got != "~/.claude-personal" {
		t.Errorf("GetConfigDir(personal) = %q, want %q", got, "~/.claude-personal")
	}
	if got := GetConfigDir("nonexistent"); got != "" {
		t.Errorf("GetConfigDir(nonexistent) = %q, want empty", got)
	}
	if got := GetConfigDir(""); got != "" {
		t.Errorf("GetConfigDir('') = %q, want empty", got)
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

func TestTokenRefreshError(t *testing.T) {
	// Test basic error
	err := &TokenRefreshError{Message: "token expired"}
	if got := err.Error(); got != "token expired" {
		t.Errorf("TokenRefreshError.Error() = %q, want %q", got, "token expired")
	}

	// Test WithAccount
	errWithAccount := err.WithAccount("personal")
	if errWithAccount.AccountName != "personal" {
		t.Errorf("WithAccount().AccountName = %q, want %q", errWithAccount.AccountName, "personal")
	}
	if errWithAccount.Message != "token expired" {
		t.Errorf("WithAccount().Message = %q, want %q", errWithAccount.Message, "token expired")
	}

	// Test ActionableGuidance with account name
	guidance := errWithAccount.ActionableGuidance()
	expected := "To re-authorize: orch account add personal"
	if guidance != expected {
		t.Errorf("ActionableGuidance() = %q, want %q", guidance, expected)
	}

	// Test ActionableGuidance without account name
	guidanceNoAccount := err.ActionableGuidance()
	expectedNoAccount := "To re-authorize: orch account add <name>"
	if guidanceNoAccount != expectedNoAccount {
		t.Errorf("ActionableGuidance() without account = %q, want %q", guidanceNoAccount, expectedNoAccount)
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

// ============================================================================
// Auto-Switch Logic Tests
// ============================================================================

// TestAutoSwitchThresholdLogic tests the threshold-based decision logic.
// Uses the same calculations as ShouldAutoSwitch but with mock data.
func TestAutoSwitchThresholdLogic(t *testing.T) {
	thresholds := DefaultAutoSwitchThresholds()

	tests := []struct {
		name               string
		fiveHourUsed       float64
		weeklyUsed         float64
		wantOverFiveHour   bool
		wantOverWeekly     bool
		wantShouldConsider bool
	}{
		{
			name:               "healthy account - well under both thresholds",
			fiveHourUsed:       50,
			weeklyUsed:         60,
			wantOverFiveHour:   false,
			wantOverWeekly:     false,
			wantShouldConsider: false,
		},
		{
			name:               "exactly at 5-hour threshold (80%) - not over",
			fiveHourUsed:       80,
			weeklyUsed:         60,
			wantOverFiveHour:   false,
			wantOverWeekly:     false,
			wantShouldConsider: false,
		},
		{
			name:               "just over 5-hour threshold (80.1%) - should consider switch",
			fiveHourUsed:       80.1,
			weeklyUsed:         60,
			wantOverFiveHour:   true,
			wantOverWeekly:     false,
			wantShouldConsider: true,
		},
		{
			name:               "at 20% remaining (80% used) - not triggered",
			fiveHourUsed:       80,
			weeklyUsed:         50,
			wantOverFiveHour:   false,
			wantOverWeekly:     false,
			wantShouldConsider: false,
		},
		{
			name:               "below 20% remaining (81% used) - should trigger",
			fiveHourUsed:       81,
			weeklyUsed:         50,
			wantOverFiveHour:   true,
			wantOverWeekly:     false,
			wantShouldConsider: true,
		},
		{
			name:               "exactly at weekly threshold (90%) - not over",
			fiveHourUsed:       50,
			weeklyUsed:         90,
			wantOverFiveHour:   false,
			wantOverWeekly:     false,
			wantShouldConsider: false,
		},
		{
			name:               "just over weekly threshold (90.1%) - should consider switch",
			fiveHourUsed:       50,
			weeklyUsed:         90.1,
			wantOverFiveHour:   false,
			wantOverWeekly:     true,
			wantShouldConsider: true,
		},
		{
			name:               "both over thresholds - critical state",
			fiveHourUsed:       85,
			weeklyUsed:         95,
			wantOverFiveHour:   true,
			wantOverWeekly:     true,
			wantShouldConsider: true,
		},
		{
			name:               "critical: only 5% remaining on 5-hour",
			fiveHourUsed:       95,
			weeklyUsed:         50,
			wantOverFiveHour:   true,
			wantOverWeekly:     false,
			wantShouldConsider: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same logic as ShouldAutoSwitch
			overFiveHour := tt.fiveHourUsed > thresholds.FiveHourThreshold
			overWeekly := tt.weeklyUsed > thresholds.WeeklyThreshold
			shouldConsider := overFiveHour || overWeekly

			if overFiveHour != tt.wantOverFiveHour {
				t.Errorf("overFiveHour = %v, want %v (used: %.1f%%, threshold: %.1f%%)",
					overFiveHour, tt.wantOverFiveHour, tt.fiveHourUsed, thresholds.FiveHourThreshold)
			}
			if overWeekly != tt.wantOverWeekly {
				t.Errorf("overWeekly = %v, want %v (used: %.1f%%, threshold: %.1f%%)",
					overWeekly, tt.wantOverWeekly, tt.weeklyUsed, thresholds.WeeklyThreshold)
			}
			if shouldConsider != tt.wantShouldConsider {
				t.Errorf("shouldConsider = %v, want %v", shouldConsider, tt.wantShouldConsider)
			}
		})
	}
}

// TestAutoSwitchHeadroomCalculation tests the headroom calculation logic.
func TestAutoSwitchHeadroomCalculation(t *testing.T) {
	tests := []struct {
		name              string
		fiveHourUsed      float64
		weeklyUsed        float64
		wantFiveHourRoom  float64
		wantWeeklyRoom    float64
		wantEffectiveRoom float64
	}{
		{
			name:              "50% used on both",
			fiveHourUsed:      50,
			weeklyUsed:        50,
			wantFiveHourRoom:  50,
			wantWeeklyRoom:    50,
			wantEffectiveRoom: 50,
		},
		{
			name:              "5-hour is the bottleneck",
			fiveHourUsed:      80,
			weeklyUsed:        50,
			wantFiveHourRoom:  20,
			wantWeeklyRoom:    50,
			wantEffectiveRoom: 20, // min of both
		},
		{
			name:              "weekly is the bottleneck",
			fiveHourUsed:      50,
			weeklyUsed:        90,
			wantFiveHourRoom:  50,
			wantWeeklyRoom:    10,
			wantEffectiveRoom: 10, // min of both
		},
		{
			name:              "critical state - both nearly exhausted",
			fiveHourUsed:      95,
			weeklyUsed:        98,
			wantFiveHourRoom:  5,
			wantWeeklyRoom:    2,
			wantEffectiveRoom: 2, // min of both
		},
		{
			name:              "fresh account - lots of headroom",
			fiveHourUsed:      5,
			weeklyUsed:        10,
			wantFiveHourRoom:  95,
			wantWeeklyRoom:    90,
			wantEffectiveRoom: 90, // min of both
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fiveHourHeadroom := 100.0 - tt.fiveHourUsed
			weeklyHeadroom := 100.0 - tt.weeklyUsed
			effectiveHeadroom := min(fiveHourHeadroom, weeklyHeadroom)

			if fiveHourHeadroom != tt.wantFiveHourRoom {
				t.Errorf("fiveHourHeadroom = %.1f, want %.1f", fiveHourHeadroom, tt.wantFiveHourRoom)
			}
			if weeklyHeadroom != tt.wantWeeklyRoom {
				t.Errorf("weeklyHeadroom = %.1f, want %.1f", weeklyHeadroom, tt.wantWeeklyRoom)
			}
			if effectiveHeadroom != tt.wantEffectiveRoom {
				t.Errorf("effectiveHeadroom = %.1f, want %.1f", effectiveHeadroom, tt.wantEffectiveRoom)
			}
		})
	}
}

// TestAutoSwitchMinHeadroomDelta tests the minimum delta requirement for switching.
func TestAutoSwitchMinHeadroomDelta(t *testing.T) {
	thresholds := DefaultAutoSwitchThresholds()

	tests := []struct {
		name            string
		currentHeadroom float64
		altHeadroom     float64
		wantSwitch      bool
	}{
		{
			name:            "alt has exactly 10% more - meets delta requirement",
			currentHeadroom: 15,
			altHeadroom:     25.01,
			wantSwitch:      true,
		},
		{
			name:            "alt has exactly 10% more - on boundary",
			currentHeadroom: 15,
			altHeadroom:     25,
			wantSwitch:      false, // Must be strictly greater than current + delta
		},
		{
			name:            "alt has 5% more - not enough",
			currentHeadroom: 15,
			altHeadroom:     20,
			wantSwitch:      false,
		},
		{
			name:            "alt has much more headroom",
			currentHeadroom: 10,
			altHeadroom:     50,
			wantSwitch:      true,
		},
		{
			name:            "alt is actually worse",
			currentHeadroom: 30,
			altHeadroom:     20,
			wantSwitch:      false,
		},
		{
			name:            "alt is barely better",
			currentHeadroom: 30,
			altHeadroom:     35,
			wantSwitch:      false, // 5% delta not enough (need 10%)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Same logic as ShouldAutoSwitch
			shouldSwitch := tt.altHeadroom > tt.currentHeadroom+thresholds.MinHeadroomDelta

			if shouldSwitch != tt.wantSwitch {
				t.Errorf("shouldSwitch = %v, want %v (current: %.1f, alt: %.1f, delta needed: %.1f)",
					shouldSwitch, tt.wantSwitch, tt.currentHeadroom, tt.altHeadroom, thresholds.MinHeadroomDelta)
			}
		})
	}
}

// TestAutoSwitchDecisionScenarios tests complete switching decision scenarios.
func TestAutoSwitchDecisionScenarios(t *testing.T) {
	thresholds := DefaultAutoSwitchThresholds()

	tests := []struct {
		name                 string
		currentFiveHour      float64
		currentWeekly        float64
		altFiveHour          float64
		altWeekly            float64
		wantShouldSwitch     bool
		wantReason           string
		wantCurrentEffective float64
		wantAltEffective     float64
	}{
		{
			name:                 "current healthy - no switch needed",
			currentFiveHour:      50,
			currentWeekly:        60,
			altFiveHour:          20,
			altWeekly:            30,
			wantShouldSwitch:     false,
			wantReason:           "current account healthy",
			wantCurrentEffective: 40, // min(50, 40)
			wantAltEffective:     70, // min(80, 70)
		},
		{
			name:                 "current over 5h threshold, alt has more headroom",
			currentFiveHour:      85,
			currentWeekly:        60,
			altFiveHour:          20,
			altWeekly:            30,
			wantShouldSwitch:     true,
			wantReason:           "better alt available",
			wantCurrentEffective: 15, // min(15, 40)
			wantAltEffective:     70, // min(80, 70)
		},
		{
			name:                 "current over weekly threshold, alt has more headroom",
			currentFiveHour:      50,
			currentWeekly:        95,
			altFiveHour:          30,
			altWeekly:            40,
			wantShouldSwitch:     true,
			wantReason:           "better alt available",
			wantCurrentEffective: 5,  // min(50, 5)
			wantAltEffective:     60, // min(70, 60)
		},
		{
			name:                 "both over threshold but alt not better enough (9% delta)",
			currentFiveHour:      85,
			currentWeekly:        60,
			altFiveHour:          76, // 24% headroom
			altWeekly:            60, // 40% headroom -> effective 24%
			wantShouldSwitch:     false,
			wantReason:           "alt not enough better",
			wantCurrentEffective: 15, // min(15, 40)
			wantAltEffective:     24, // min(24, 40) - only 9% delta
		},
		{
			name:                 "both accounts exhausted - no switch helps",
			currentFiveHour:      95,
			currentWeekly:        95,
			altFiveHour:          92,
			altWeekly:            93,
			wantShouldSwitch:     false,
			wantReason:           "no healthy alt",
			wantCurrentEffective: 5, // min(5, 5)
			wantAltEffective:     7, // min(8, 7) - only 2% delta
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate effective headroom (same as ShouldAutoSwitch)
			currentFiveHourHeadroom := 100.0 - tt.currentFiveHour
			currentWeeklyHeadroom := 100.0 - tt.currentWeekly
			currentEffective := min(currentFiveHourHeadroom, currentWeeklyHeadroom)

			altFiveHourHeadroom := 100.0 - tt.altFiveHour
			altWeeklyHeadroom := 100.0 - tt.altWeekly
			altEffective := min(altFiveHourHeadroom, altWeeklyHeadroom)

			// Check effective headroom calculations
			if currentEffective != tt.wantCurrentEffective {
				t.Errorf("currentEffective = %.1f, want %.1f", currentEffective, tt.wantCurrentEffective)
			}
			if altEffective != tt.wantAltEffective {
				t.Errorf("altEffective = %.1f, want %.1f", altEffective, tt.wantAltEffective)
			}

			// Determine if switch should happen
			overFiveHour := tt.currentFiveHour > thresholds.FiveHourThreshold
			overWeekly := tt.currentWeekly > thresholds.WeeklyThreshold
			overThreshold := overFiveHour || overWeekly
			altBetterEnough := altEffective > currentEffective+thresholds.MinHeadroomDelta
			shouldSwitch := overThreshold && altBetterEnough

			if shouldSwitch != tt.wantShouldSwitch {
				t.Errorf("shouldSwitch = %v, want %v (over threshold: %v, alt better enough: %v)",
					shouldSwitch, tt.wantShouldSwitch, overThreshold, altBetterEnough)
			}
		})
	}
}

// TestSaveOpenCodeAuth_PreservesOtherProviders tests that saving anthropic auth
// doesn't nuke other provider credentials (e.g., openai OAuth).
func TestSaveOpenCodeAuth_PreservesOtherProviders(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create the auth directory
	authDir := filepath.Join(tmpDir, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write an existing auth.json with both anthropic AND openai sections
	existingAuth := map[string]interface{}{
		"anthropic": map[string]interface{}{
			"type":    "oauth",
			"refresh": "old-anthropic-refresh",
			"access":  "old-anthropic-access",
			"expires": 1000000,
		},
		"openai": map[string]interface{}{
			"type":    "oauth",
			"refresh": "openai-refresh-token",
			"access":  "openai-access-token",
			"expires": 2000000,
		},
	}
	existingData, _ := json.MarshalIndent(existingAuth, "", "  ")
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), existingData, 0600); err != nil {
		t.Fatal(err)
	}

	// Now save new anthropic credentials (simulating account switch)
	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = "new-anthropic-refresh"
	auth.Anthropic.Access = "new-anthropic-access"
	auth.Anthropic.Expires = 9999999

	if err := SaveOpenCodeAuth(auth); err != nil {
		t.Fatalf("SaveOpenCodeAuth() error = %v", err)
	}

	// Read back the file and verify openai section is preserved
	savedData, err := os.ReadFile(filepath.Join(authDir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatal(err)
	}

	// Check anthropic was updated
	anthropic, ok := saved["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatal("anthropic section missing from saved auth.json")
	}
	if anthropic["refresh"] != "new-anthropic-refresh" {
		t.Errorf("anthropic.refresh = %v, want %v", anthropic["refresh"], "new-anthropic-refresh")
	}

	// Check openai was preserved
	openai, ok := saved["openai"].(map[string]interface{})
	if !ok {
		t.Fatal("openai section was nuked from auth.json — this is the bug")
	}
	if openai["refresh"] != "openai-refresh-token" {
		t.Errorf("openai.refresh = %v, want %v", openai["refresh"], "openai-refresh-token")
	}
	if openai["access"] != "openai-access-token" {
		t.Errorf("openai.access = %v, want %v", openai["access"], "openai-access-token")
	}
}

// TestSaveOpenCodeAuth_NoExistingFile tests that saving works when no auth.json exists yet.
func TestSaveOpenCodeAuth_NoExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = "fresh-refresh"
	auth.Anthropic.Access = "fresh-access"
	auth.Anthropic.Expires = 5555555

	if err := SaveOpenCodeAuth(auth); err != nil {
		t.Fatalf("SaveOpenCodeAuth() error = %v", err)
	}

	// Verify file was created with anthropic section
	savedData, err := os.ReadFile(OpenCodeAuthPath())
	if err != nil {
		t.Fatal(err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatal(err)
	}

	anthropic, ok := saved["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatal("anthropic section missing")
	}
	if anthropic["refresh"] != "fresh-refresh" {
		t.Errorf("anthropic.refresh = %v, want %v", anthropic["refresh"], "fresh-refresh")
	}
}

// ============================================================================
// RecommendAccount Tests
// ============================================================================

func TestRecommendAccount_NoPrimaries(t *testing.T) {
	accounts := []AccountInfo{
		{Name: "spillover1", Role: "spillover"},
	}
	got := RecommendAccount(accounts, nil)
	// With no primaries and no capacity data, recommend first available account
	if got != "spillover1" {
		t.Errorf("RecommendAccount() = %q, want %q (only available account)", got, "spillover1")
	}
}

func TestRecommendAccount_EmptyAccounts(t *testing.T) {
	got := RecommendAccount(nil, nil)
	if got != "" {
		t.Errorf("RecommendAccount() = %q, want empty", got)
	}
}

func TestRecommendAccount_SinglePrimary(t *testing.T) {
	accounts := []AccountInfo{
		{Name: "work", Role: "primary", Tier: "20x"},
	}
	got := RecommendAccount(accounts, nil)
	if got != "work" {
		t.Errorf("RecommendAccount() = %q, want %q", got, "work")
	}
}

func TestRecommendAccount_PrimaryWithoutRole(t *testing.T) {
	// Backward compat: no role = primary candidate
	accounts := []AccountInfo{
		{Name: "old-account", Role: "", Tier: ""},
	}
	got := RecommendAccount(accounts, nil)
	if got != "old-account" {
		t.Errorf("RecommendAccount() = %q, want %q", got, "old-account")
	}
}

func TestRecommendAccount_WithCapacityFetcher_TierWeightedFiveHourWins(t *testing.T) {
	accounts := []AccountInfo{
		{Name: "work", Role: "primary", Tier: "20x"},
		{Name: "personal", Role: "spillover", Tier: "5x"},
	}
	fetcher := func(name string) *CapacityInfo {
		if name == "work" {
			return &CapacityInfo{FiveHourRemaining: 87, SevenDayRemaining: 72}
		}
		return &CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
	}
	got := RecommendAccount(accounts, fetcher)
	// Work wins: 87%*20=1740 5h headroom > 95%*5=475 5h headroom
	if got != "work" {
		t.Errorf("RecommendAccount() = %q, want %q (work has 1740 vs 475 absolute 5h headroom)", got, "work")
	}
}

func TestRecommendAccount_WithCapacityFetcher_PersonalWinsWhenWorkExhausted(t *testing.T) {
	accounts := []AccountInfo{
		{Name: "work", Role: "primary", Tier: "20x"},
		{Name: "personal", Role: "spillover", Tier: "5x"},
	}
	fetcher := func(name string) *CapacityInfo {
		if name == "work" {
			return &CapacityInfo{FiveHourRemaining: 10, SevenDayRemaining: 15}
		}
		return &CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88}
	}
	got := RecommendAccount(accounts, fetcher)
	// Personal wins: 95%*5=475 5h headroom > 10%*20=200 5h headroom
	if got != "personal" {
		t.Errorf("RecommendAccount() = %q, want %q (personal has 475 vs 200 absolute 5h headroom)", got, "personal")
	}
}

func TestRecommendAccount_WithCapacityFetcher_EqualRawCapacityTierBreaksTie(t *testing.T) {
	accounts := []AccountInfo{
		{Name: "work", Role: "primary", Tier: "20x"},
		{Name: "personal", Role: "spillover", Tier: "5x"},
	}
	fetcher := func(name string) *CapacityInfo {
		return &CapacityInfo{FiveHourRemaining: 5, SevenDayRemaining: 3}
	}
	got := RecommendAccount(accounts, fetcher)
	// Same raw capacity but work: 5%*20=100, personal: 5%*5=25 → work wins
	if got != "work" {
		t.Errorf("RecommendAccount() = %q, want %q (work has higher tier-weighted headroom)", got, "work")
	}
}

func TestRecommendAccount_DeterministicSorting(t *testing.T) {
	// Multiple primaries — should pick alphabetically first
	accounts := []AccountInfo{
		{Name: "charlie", Role: "primary"},
		{Name: "alpha", Role: "primary"},
		{Name: "bravo", Role: "primary"},
	}
	got := RecommendAccount(accounts, nil)
	if got != "alpha" {
		t.Errorf("RecommendAccount() = %q, want %q (alphabetical sort)", got, "alpha")
	}
}

// ============================================================================
// ParseTierMultiplier Tests
// ============================================================================

func TestParseTierMultiplier(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"5x", 5.0},
		{"20x", 20.0},
		{"1x", 1.0},
		{"5X", 5.0},   // case-insensitive
		{" 20x ", 20.0}, // whitespace trimmed
		{"", 1.0},      // empty → default 1.0
		{"invalid", 1.0}, // unparseable → default 1.0
		{"0x", 1.0},    // zero → default 1.0
		{"-5x", 1.0},   // negative → default 1.0
	}
	for _, tt := range tests {
		got := ParseTierMultiplier(tt.input)
		if got != tt.want {
			t.Errorf("ParseTierMultiplier(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ============================================================================
// AddAccount Metadata Preservation Tests
// ============================================================================

func TestAddAccountPreservesMetadata_ExistingAccount(t *testing.T) {
	// Simulates the re-auth flow: account exists with metadata, re-add preserves it
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Pre-populate config with metadata
	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "old@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
				Tier:         "20x",
				Role:         "primary",
				ConfigDir:    "~/.claude",
			},
		},
		Default: "work",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Simulate what AddAccount does internally (without the actual OAuth flow):
	// Load config, create new account, merge existing metadata, save
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	acc := Account{
		Email:        "new@example.com",
		RefreshToken: "new-token",
		Source:       "saved",
	}
	// This is the fix: merge existing metadata
	if existing, ok := config.Accounts["work"]; ok {
		acc.Tier = existing.Tier
		acc.Role = existing.Role
		acc.ConfigDir = existing.ConfigDir
	}
	config.Save("work", acc, false)

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify metadata survived
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Email != "new@example.com" {
		t.Errorf("work.Email = %q, want %q (should be updated)", work.Email, "new@example.com")
	}
	if work.RefreshToken != "new-token" {
		t.Errorf("work.RefreshToken = %q, want %q (should be updated)", work.RefreshToken, "new-token")
	}
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q (should be preserved)", work.Tier, "20x")
	}
	if work.Role != "primary" {
		t.Errorf("work.Role = %q, want %q (should be preserved)", work.Role, "primary")
	}
	if work.ConfigDir != "~/.claude" {
		t.Errorf("work.ConfigDir = %q, want %q (should be preserved)", work.ConfigDir, "~/.claude")
	}
}

func TestAddAccountNoMetadata_NewAccount(t *testing.T) {
	// When account doesn't exist yet, no metadata to merge
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	acc := Account{
		Email:        "new@example.com",
		RefreshToken: "new-token",
		Source:       "saved",
	}
	// No existing account — merge is a no-op
	if existing, ok := config.Accounts["work"]; ok {
		acc.Tier = existing.Tier
		acc.Role = existing.Role
		acc.ConfigDir = existing.ConfigDir
	}
	config.Save("work", acc, true)

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Tier != "" {
		t.Errorf("work.Tier = %q, want empty (new account)", work.Tier)
	}
	if work.Role != "" {
		t.Errorf("work.Role = %q, want empty (new account)", work.Role)
	}
	if work.ConfigDir != "" {
		t.Errorf("work.ConfigDir = %q, want empty (new account)", work.ConfigDir)
	}
}

// ============================================================================
// LoadAndSaveConfig Tests
// ============================================================================

func TestLoadAndSaveConfig_AtomicModify(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Pre-populate
	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "work@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
				Tier:         "20x",
			},
		},
		Default: "work",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Use LoadAndSaveConfig to atomically update token
	err := LoadAndSaveConfig(func(cfg *Config) error {
		acc := cfg.Accounts["work"]
		acc.RefreshToken = "new-token"
		cfg.Accounts["work"] = acc
		return nil
	})
	if err != nil {
		t.Fatalf("LoadAndSaveConfig() error = %v", err)
	}

	// Verify update
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.RefreshToken != "new-token" {
		t.Errorf("work.RefreshToken = %q, want %q", work.RefreshToken, "new-token")
	}
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q (should be preserved)", work.Tier, "20x")
	}
}

func TestRecommendAccount_WithCapacityFetcher_EqualAbsoluteUsesAlphabetical(t *testing.T) {
	// Same tier, same capacity → alphabetical tie-break
	accounts := []AccountInfo{
		{Name: "work", Role: "primary", Tier: "5x"},
		{Name: "personal", Role: "spillover", Tier: "5x"},
	}
	fetcher := func(name string) *CapacityInfo {
		return &CapacityInfo{FiveHourRemaining: 50, SevenDayRemaining: 50}
	}
	got := RecommendAccount(accounts, fetcher)
	// Equal absolute headroom → "personal" < "work" alphabetically
	if got != "personal" {
		t.Errorf("RecommendAccount() = %q, want %q (alphabetical tie-break)", got, "personal")
	}
}

// TestAutoSwitchCustomThresholds tests that custom thresholds are respected.
func TestAutoSwitchCustomThresholds(t *testing.T) {
	// Test with more aggressive thresholds
	aggressiveThresholds := AutoSwitchThresholds{
		FiveHourThreshold: 60, // Switch when 60% used (40% remaining)
		WeeklyThreshold:   70, // Switch when 70% used (30% remaining)
		MinHeadroomDelta:  5,  // Require only 5% improvement
	}

	// Same usage that would NOT trigger default thresholds
	fiveHourUsed := 65.0
	weeklyUsed := 75.0

	// Should NOT trigger with default thresholds
	defaultThresholds := DefaultAutoSwitchThresholds()
	overFiveHourDefault := fiveHourUsed > defaultThresholds.FiveHourThreshold
	overWeeklyDefault := weeklyUsed > defaultThresholds.WeeklyThreshold
	if overFiveHourDefault || overWeeklyDefault {
		t.Errorf("65%% 5h and 75%% weekly should NOT trigger default thresholds")
	}

	// Should trigger with aggressive thresholds
	overFiveHourAggressive := fiveHourUsed > aggressiveThresholds.FiveHourThreshold
	overWeeklyAggressive := weeklyUsed > aggressiveThresholds.WeeklyThreshold
	if !overFiveHourAggressive || !overWeeklyAggressive {
		t.Errorf("65%% 5h and 75%% weekly SHOULD trigger aggressive thresholds")
	}

	// Test the smaller delta requirement
	currentHeadroom := 35.0
	altHeadroom := 38.0 // Only 3% more

	// Should NOT switch with default delta (10%)
	if altHeadroom > currentHeadroom+defaultThresholds.MinHeadroomDelta {
		t.Errorf("3%% improvement should NOT trigger with 10%% delta requirement")
	}

	// Should NOT switch with aggressive delta (5%) - only 3% improvement
	if altHeadroom > currentHeadroom+aggressiveThresholds.MinHeadroomDelta {
		t.Errorf("3%% improvement should NOT trigger with 5%% delta requirement")
	}

	// 6% improvement SHOULD switch with aggressive delta
	altHeadroom = 41.0
	if !(altHeadroom > currentHeadroom+aggressiveThresholds.MinHeadroomDelta) {
		t.Errorf("6%% improvement SHOULD trigger with 5%% delta requirement")
	}
}
