// Package port provides port allocation registry for orch-go projects.
package port

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")

	// New registry should work with non-existent file
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if reg == nil {
		t.Fatal("New() returned nil registry")
	}
}

func TestPortRanges(t *testing.T) {
	tests := []struct {
		purpose string
		start   int
		end     int
	}{
		{PurposeVite, 5173, 5199},
		{PurposeAPI, 3333, 3399},
	}

	for _, tt := range tests {
		t.Run(tt.purpose, func(t *testing.T) {
			r := GetRange(tt.purpose)
			if r.Start != tt.start {
				t.Errorf("GetRange(%s).Start = %d, want %d", tt.purpose, r.Start, tt.start)
			}
			if r.End != tt.end {
				t.Errorf("GetRange(%s).End = %d, want %d", tt.purpose, r.End, tt.end)
			}
		})
	}
}

func TestAllocatePort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate a vite port for project1
	port, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}
	if port < 5173 || port > 5199 {
		t.Errorf("Allocated port %d outside vite range [5173-5199]", port)
	}

	// First port should be start of range
	if port != 5173 {
		t.Errorf("First allocation should be 5173, got %d", port)
	}

	// Allocate an API port for project1
	apiPort, err := reg.Allocate("project1", "api", PurposeAPI)
	if err != nil {
		t.Fatalf("Allocate() API failed: %v", err)
	}
	if apiPort < 3333 || apiPort > 3399 {
		t.Errorf("Allocated API port %d outside api range [3333-3399]", apiPort)
	}
	if apiPort != 3333 {
		t.Errorf("First API allocation should be 3333, got %d", apiPort)
	}
}

func TestAllocateSameServiceReturnsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate first time
	port1, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	// Allocate same project/service again should return same port
	port2, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() second call failed: %v", err)
	}

	if port1 != port2 {
		t.Errorf("Same project/service should return same port: got %d and %d", port1, port2)
	}
}

func TestAllocateDifferentProjectsGetDifferentPorts(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate for project1
	port1, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() project1 failed: %v", err)
	}

	// Allocate for project2
	port2, err := reg.Allocate("project2", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() project2 failed: %v", err)
	}

	if port1 == port2 {
		t.Errorf("Different projects should get different ports: both got %d", port1)
	}
}

func TestListAllocations(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Empty list initially
	allocs := reg.List()
	if len(allocs) != 0 {
		t.Errorf("List() should be empty initially, got %d", len(allocs))
	}

	// Allocate some ports
	reg.Allocate("project1", "web", PurposeVite)
	reg.Allocate("project1", "api", PurposeAPI)
	reg.Allocate("project2", "web", PurposeVite)

	allocs = reg.List()
	if len(allocs) != 3 {
		t.Errorf("List() should have 3 allocations, got %d", len(allocs))
	}
}

func TestListByProject(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate ports for multiple projects
	reg.Allocate("project1", "web", PurposeVite)
	reg.Allocate("project1", "api", PurposeAPI)
	reg.Allocate("project2", "web", PurposeVite)

	// List for project1
	allocs := reg.ListByProject("project1")
	if len(allocs) != 2 {
		t.Errorf("ListByProject(project1) should have 2 allocations, got %d", len(allocs))
	}

	// List for project2
	allocs = reg.ListByProject("project2")
	if len(allocs) != 1 {
		t.Errorf("ListByProject(project2) should have 1 allocation, got %d", len(allocs))
	}
}

func TestRelease(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate a port
	port, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	// Verify it's allocated
	allocs := reg.List()
	if len(allocs) != 1 {
		t.Errorf("Should have 1 allocation, got %d", len(allocs))
	}

	// Release it
	ok := reg.Release("project1", "web")
	if !ok {
		t.Error("Release() should return true for existing allocation")
	}

	// Verify it's released
	allocs = reg.List()
	if len(allocs) != 0 {
		t.Errorf("Should have 0 allocations after release, got %d", len(allocs))
	}

	// Release again should return false
	ok = reg.Release("project1", "web")
	if ok {
		t.Error("Release() should return false for non-existent allocation")
	}

	// After release, can reallocate the same port
	port2, err := reg.Allocate("project3", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() after release failed: %v", err)
	}
	if port2 != port {
		t.Errorf("Released port should be reusable: got %d, want %d", port2, port)
	}
}

func TestReleaseByPort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate a port
	port, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	// Release by port number
	ok := reg.ReleaseByPort(port)
	if !ok {
		t.Error("ReleaseByPort() should return true for allocated port")
	}

	// Verify it's released
	allocs := reg.List()
	if len(allocs) != 0 {
		t.Errorf("Should have 0 allocations after release, got %d", len(allocs))
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")

	// Create registry and allocate
	reg1, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	port1, err := reg1.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}
	if err := reg1.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Create new registry pointing to same file
	reg2, err := New(path)
	if err != nil {
		t.Fatalf("New() second instance failed: %v", err)
	}

	// Should have same allocation
	allocs := reg2.List()
	if len(allocs) != 1 {
		t.Fatalf("New registry should have 1 allocation from file, got %d", len(allocs))
	}
	if allocs[0].Port != port1 {
		t.Errorf("Persisted port mismatch: got %d, want %d", allocs[0].Port, port1)
	}
}

func TestFindAllocation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate a port
	port, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	// Find by project/service
	alloc := reg.Find("project1", "web")
	if alloc == nil {
		t.Fatal("Find() returned nil for existing allocation")
	}
	if alloc.Port != port {
		t.Errorf("Find() returned wrong port: got %d, want %d", alloc.Port, port)
	}

	// Find non-existent
	alloc = reg.Find("project2", "web")
	if alloc != nil {
		t.Errorf("Find() should return nil for non-existent allocation, got %+v", alloc)
	}
}

func TestFindByPort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate a port
	port, err := reg.Allocate("project1", "web", PurposeVite)
	if err != nil {
		t.Fatalf("Allocate() failed: %v", err)
	}

	// Find by port number
	alloc := reg.FindByPort(port)
	if alloc == nil {
		t.Fatal("FindByPort() returned nil for existing allocation")
	}
	if alloc.Project != "project1" || alloc.Service != "web" {
		t.Errorf("FindByPort() returned wrong allocation: got %+v", alloc)
	}

	// Find non-existent port
	alloc = reg.FindByPort(9999)
	if alloc != nil {
		t.Errorf("FindByPort() should return nil for non-existent port, got %+v", alloc)
	}
}

func TestRangeExhaustion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Vite range is 5173-5199 = 27 ports
	// Allocate all of them
	viteRange := GetRange(PurposeVite)
	numPorts := viteRange.End - viteRange.Start + 1

	for i := 0; i < numPorts; i++ {
		_, err := reg.Allocate("project"+string(rune('a'+i)), "web", PurposeVite)
		if err != nil {
			t.Fatalf("Allocate() %d failed: %v", i, err)
		}
	}

	// Next allocation should fail
	_, err = reg.Allocate("overflow", "web", PurposeVite)
	if err == nil {
		t.Error("Allocate() should fail when range is exhausted")
	}
	if err != ErrRangeExhausted {
		t.Errorf("Expected ErrRangeExhausted, got %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Error("DefaultPath() returned empty string")
	}
	// Should be in ~/.orch/
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultPath() should return absolute path, got %s", path)
	}
}

func TestInvalidPurpose(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = reg.Allocate("project", "web", "invalid-purpose")
	if err == nil {
		t.Error("Allocate() should fail for invalid purpose")
	}
	if err != ErrInvalidPurpose {
		t.Errorf("Expected ErrInvalidPurpose, got %v", err)
	}
}

func TestYAMLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ports.yaml")
	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Allocate some ports
	reg.Allocate("project1", "web", PurposeVite)
	reg.Allocate("project1", "api", PurposeAPI)
	if err := reg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Read the file and verify it's valid YAML
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	// Basic sanity checks on format
	content := string(data)
	if len(content) == 0 {
		t.Error("Saved file is empty")
	}
	// Should contain project name
	if !contains(content, "project1") {
		t.Error("YAML should contain project name")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
