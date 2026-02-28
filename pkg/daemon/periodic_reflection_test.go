package daemon

import (
	"fmt"
	"testing"
	"time"
)

func TestDaemon_ShouldRunReflection_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  false,
			ReflectInterval: time.Hour,
		},
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when disabled")
	}
}

func TestDaemon_ShouldRunReflection_ZeroInterval(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: 0,
		},
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when interval is 0")
	}
}

func TestDaemon_ShouldRunReflection_NeverRun(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
	}

	// lastReflect is zero time (never run)
	if !d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return true when never run before")
	}
}

func TestDaemon_ShouldRunReflection_IntervalElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Now().Add(-2 * time.Hour), // 2 hours ago
	}

	if !d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return true when interval has elapsed")
	}
}

func TestDaemon_ShouldRunReflection_IntervalNotElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Now().Add(-30 * time.Minute), // 30 minutes ago
	}

	if d.ShouldRunReflection() {
		t.Error("ShouldRunReflection() should return false when interval has not elapsed")
	}
}

func TestDaemon_RunPeriodicReflection_NotDue(t *testing.T) {
	reflectCalled := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: true,
		},
		lastReflect: time.Now(), // Just ran
		Reflector: &mockReflector{
			ReflectFunc: func(createIssues bool) (*ReflectResult, error) {
				reflectCalled = true
				return &ReflectResult{}, nil
			},
		},
	}

	result := d.RunPeriodicReflection()
	if result != nil {
		t.Error("RunPeriodicReflection() should return nil when not due")
	}
	if reflectCalled {
		t.Error("Reflector.Reflect should not be called when not due")
	}
}

func TestDaemon_RunPeriodicReflection_Due(t *testing.T) {
	reflectCalled := false
	createIssuesValue := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: true,
		},
		lastReflect: time.Now().Add(-2 * time.Hour), // 2 hours ago (due)
		Reflector: &mockReflector{
			ReflectFunc: func(createIssues bool) (*ReflectResult, error) {
				reflectCalled = true
				createIssuesValue = createIssues
				return &ReflectResult{
					Suggestions: &ReflectSuggestions{
						Synthesis: []SynthesisSuggestion{{Topic: "test", Count: 5}},
					},
					Saved:   true,
					Message: "Test reflection",
				}, nil
			},
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result when due")
	}
	if !reflectCalled {
		t.Error("Reflector.Reflect should be called when due")
	}
	if !createIssuesValue {
		t.Error("createIssues should be true based on config")
	}
	if d.lastReflect.IsZero() {
		t.Error("lastReflect should be updated after running")
	}
}

func TestDaemon_RunPeriodicReflection_OpenEnabled(t *testing.T) {
	openCalled := false
	d := &Daemon{
		Config: Config{
			ReflectEnabled:     true,
			ReflectInterval:    time.Hour,
			ReflectOpenEnabled: true,
		},
		lastReflect: time.Now().Add(-2 * time.Hour),
		Reflector: &mockReflector{
			ReflectFunc: func(createIssues bool) (*ReflectResult, error) {
				return &ReflectResult{}, nil
			},
			ReflectOpenFunc: func() error {
				openCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result when due")
	}
	if !openCalled {
		t.Error("Reflector.ReflectOpen should be called when ReflectOpenEnabled is true")
	}
}

func TestDaemon_RunPeriodicReflection_Error(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:      true,
			ReflectInterval:     time.Hour,
			ReflectCreateIssues: false,
		},
		lastReflect: time.Time{}, // Never run
		Reflector: &mockReflector{
			ReflectFunc: func(createIssues bool) (*ReflectResult, error) {
				return nil, fmt.Errorf("kb reflect failed")
			},
		},
	}

	result := d.RunPeriodicReflection()
	if result == nil {
		t.Fatal("RunPeriodicReflection() should return result on error")
	}
	if result.Error == nil {
		t.Error("Result should have error")
	}
}

func TestDaemon_LastReflectTime(t *testing.T) {
	now := time.Now()
	d := &Daemon{
		lastReflect: now,
	}

	if !d.LastReflectTime().Equal(now) {
		t.Errorf("LastReflectTime() = %v, want %v", d.LastReflectTime(), now)
	}
}

func TestDaemon_NextReflectTime_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  false,
			ReflectInterval: time.Hour,
		},
	}

	next := d.NextReflectTime()
	if !next.IsZero() {
		t.Error("NextReflectTime() should return zero time when disabled")
	}
}

func TestDaemon_NextReflectTime_NeverRun(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: time.Time{}, // Never run
	}

	next := d.NextReflectTime()
	// Should return approximately now (due immediately)
	if time.Until(next) > time.Second {
		t.Error("NextReflectTime() should return ~now when never run")
	}
}

func TestDaemon_NextReflectTime_AfterRun(t *testing.T) {
	now := time.Now()
	d := &Daemon{
		Config: Config{
			ReflectEnabled:  true,
			ReflectInterval: time.Hour,
		},
		lastReflect: now,
	}

	next := d.NextReflectTime()
	expectedNext := now.Add(time.Hour)
	// Allow 1 second tolerance
	if next.Sub(expectedNext).Abs() > time.Second {
		t.Errorf("NextReflectTime() = %v, want ~%v", next, expectedNext)
	}
}

func TestDefaultConfig_IncludesReflect(t *testing.T) {
	config := DefaultConfig()

	if !config.ReflectEnabled {
		t.Error("DefaultConfig().ReflectEnabled should be true")
	}
	if config.ReflectInterval != time.Hour {
		t.Errorf("DefaultConfig().ReflectInterval = %v, want 1h", config.ReflectInterval)
	}
	if !config.ReflectCreateIssues {
		t.Error("DefaultConfig().ReflectCreateIssues should be true")
	}
	if !config.ReflectOpenEnabled {
		t.Error("DefaultConfig().ReflectOpenEnabled should be true")
	}
	if !config.ReflectModelDriftEnabled {
		t.Error("DefaultConfig().ReflectModelDriftEnabled should be true")
	}
	if config.ReflectModelDriftInterval != 4*time.Hour {
		t.Errorf("DefaultConfig().ReflectModelDriftInterval = %v, want 4h", config.ReflectModelDriftInterval)
	}
}

func TestNewWithConfig_InitializesReflector(t *testing.T) {
	config := Config{
		ReflectEnabled:  true,
		ReflectInterval: time.Hour,
	}
	d := NewWithConfig(config)

	if d.Reflector == nil {
		t.Error("NewWithConfig() should initialize Reflector")
	}
}
