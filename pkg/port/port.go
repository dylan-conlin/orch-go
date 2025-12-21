// Package port provides port allocation registry for orch-go projects.
//
// The port registry prevents conflicts by tracking which ports are allocated
// to which project/service combinations. Ports are allocated from predefined
// ranges by purpose (e.g., vite dev servers, API servers).
//
// Configuration file: ~/.orch/ports.yaml
//
// Example config:
//
//	allocations:
//	  - project: snap
//	    service: web
//	    port: 5173
//	    purpose: vite
//	    allocated_at: "2025-12-21T10:30:00Z"
//	  - project: snap
//	    service: api
//	    port: 3333
//	    purpose: api
//	    allocated_at: "2025-12-21T10:30:00Z"
package port

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Purpose constants for port ranges.
const (
	PurposeVite = "vite" // Dev server ports (5173-5199)
	PurposeAPI  = "api"  // API server ports (3333-3399)
)

// Range defines a port range for a purpose.
type Range struct {
	Start int
	End   int
}

// Predefined port ranges by purpose.
var ranges = map[string]Range{
	PurposeVite: {Start: 5173, End: 5199}, // 27 ports
	PurposeAPI:  {Start: 3333, End: 3399}, // 67 ports
}

// Errors.
var (
	ErrRangeExhausted = errors.New("all ports in range are allocated")
	ErrInvalidPurpose = errors.New("invalid port purpose")
	ErrNotFound       = errors.New("allocation not found")
)

// Allocation represents a port allocation for a project/service.
type Allocation struct {
	Project     string `yaml:"project"`
	Service     string `yaml:"service"`
	Port        int    `yaml:"port"`
	Purpose     string `yaml:"purpose"`
	AllocatedAt string `yaml:"allocated_at"`
}

// registryData is the on-disk YAML format.
type registryData struct {
	Allocations []Allocation `yaml:"allocations"`
}

// Registry manages port allocations across projects.
type Registry struct {
	path        string
	allocations []Allocation
	mu          sync.RWMutex
}

// GetRange returns the port range for a purpose.
// Returns an empty Range if purpose is invalid.
func GetRange(purpose string) Range {
	if r, ok := ranges[purpose]; ok {
		return r
	}
	return Range{}
}

// DefaultPath returns the default registry file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "ports.yaml")
}

// New creates a new Registry with the given path.
// If path is empty, uses DefaultPath().
func New(path string) (*Registry, error) {
	if path == "" {
		path = DefaultPath()
	}

	r := &Registry{
		path:        path,
		allocations: make([]Allocation, 0),
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

// load reads the registry from disk.
func (r *Registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		r.allocations = make([]Allocation, 0)
		return nil
	}

	data, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("failed to read ports file: %w", err)
	}

	if len(data) == 0 {
		r.allocations = make([]Allocation, 0)
		return nil
	}

	var rd registryData
	if err := yaml.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("failed to parse ports file: %w", err)
	}

	r.allocations = rd.Allocations
	if r.allocations == nil {
		r.allocations = make([]Allocation, 0)
	}

	return nil
}

// Save persists the registry to disk.
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	rd := registryData{Allocations: r.allocations}
	data, err := yaml.Marshal(rd)
	if err != nil {
		return fmt.Errorf("failed to marshal ports file: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ports file: %w", err)
	}

	return nil
}

// Allocate allocates a port for a project/service.
// If already allocated, returns the existing port.
// Otherwise, finds the next available port in the range for the purpose.
func (r *Registry) Allocate(project, service, purpose string) (int, error) {
	// Validate purpose
	portRange, ok := ranges[purpose]
	if !ok {
		return 0, ErrInvalidPurpose
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already allocated
	for _, a := range r.allocations {
		if a.Project == project && a.Service == service && a.Purpose == purpose {
			return a.Port, nil
		}
	}

	// Find used ports in this range
	usedPorts := make(map[int]bool)
	for _, a := range r.allocations {
		if a.Purpose == purpose {
			usedPorts[a.Port] = true
		}
	}

	// Find next available port
	for port := portRange.Start; port <= portRange.End; port++ {
		if !usedPorts[port] {
			// Allocate this port
			alloc := Allocation{
				Project:     project,
				Service:     service,
				Port:        port,
				Purpose:     purpose,
				AllocatedAt: time.Now().Format(time.RFC3339),
			}
			r.allocations = append(r.allocations, alloc)

			// Auto-save on allocation
			if err := r.saveUnlocked(); err != nil {
				// Remove the allocation if save fails
				r.allocations = r.allocations[:len(r.allocations)-1]
				return 0, fmt.Errorf("failed to save allocation: %w", err)
			}

			return port, nil
		}
	}

	return 0, ErrRangeExhausted
}

// saveUnlocked saves without acquiring the lock (caller must hold lock).
func (r *Registry) saveUnlocked() error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	rd := registryData{Allocations: r.allocations}
	data, err := yaml.Marshal(rd)
	if err != nil {
		return fmt.Errorf("failed to marshal ports file: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ports file: %w", err)
	}

	return nil
}

// List returns all allocations.
func (r *Registry) List() []Allocation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Allocation, len(r.allocations))
	copy(result, r.allocations)
	return result
}

// ListByProject returns allocations for a specific project.
func (r *Registry) ListByProject(project string) []Allocation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Allocation
	for _, a := range r.allocations {
		if a.Project == project {
			result = append(result, a)
		}
	}
	return result
}

// Find returns the allocation for a project/service, or nil if not found.
func (r *Registry) Find(project, service string) *Allocation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.allocations {
		if r.allocations[i].Project == project && r.allocations[i].Service == service {
			alloc := r.allocations[i]
			return &alloc
		}
	}
	return nil
}

// FindByPort returns the allocation for a specific port, or nil if not found.
func (r *Registry) FindByPort(port int) *Allocation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.allocations {
		if r.allocations[i].Port == port {
			alloc := r.allocations[i]
			return &alloc
		}
	}
	return nil
}

// Release releases a port allocation for a project/service.
// Returns true if the allocation was found and released, false otherwise.
func (r *Registry) Release(project, service string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.allocations {
		if r.allocations[i].Project == project && r.allocations[i].Service == service {
			// Remove allocation by swapping with last and truncating
			r.allocations[i] = r.allocations[len(r.allocations)-1]
			r.allocations = r.allocations[:len(r.allocations)-1]

			// Auto-save
			r.saveUnlocked()
			return true
		}
	}
	return false
}

// ReleaseByPort releases a port allocation by port number.
// Returns true if the allocation was found and released, false otherwise.
func (r *Registry) ReleaseByPort(port int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.allocations {
		if r.allocations[i].Port == port {
			// Remove allocation by swapping with last and truncating
			r.allocations[i] = r.allocations[len(r.allocations)-1]
			r.allocations = r.allocations[:len(r.allocations)-1]

			// Auto-save
			r.saveUnlocked()
			return true
		}
	}
	return false
}

// ReleaseProject releases all port allocations for a project.
// Returns the number of allocations released.
func (r *Registry) ReleaseProject(project string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	newAllocs := make([]Allocation, 0, len(r.allocations))
	for _, a := range r.allocations {
		if a.Project == project {
			count++
		} else {
			newAllocs = append(newAllocs, a)
		}
	}

	if count > 0 {
		r.allocations = newAllocs
		r.saveUnlocked()
	}
	return count
}
