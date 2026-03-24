package thread

// Lifecycle state constants for thread status.
const (
	StatusForming   = "forming"   // Initial state — question is crystallizing
	StatusActive    = "active"    // Thread has evidence, actively being worked
	StatusConverged = "converged" // Thread reached a conclusion, formalized into artifact
	StatusSubsumed  = "subsumed"  // Thread absorbed into another thread
	StatusResolved  = "resolved"  // Legacy terminal state (backward compat)
)

// IsResolved returns true if the status is any terminal state
// (resolved, converged, or subsumed).
func IsResolved(status string) bool {
	switch status {
	case StatusResolved, StatusConverged, StatusSubsumed:
		return true
	}
	return false
}

// IsActive returns true if the status is a non-terminal state
// (forming or active).
func IsActive(status string) bool {
	return !IsResolved(status)
}

// NormalizeStatus maps legacy status values to lifecycle states.
// "open" → "forming" for backward compatibility.
func NormalizeStatus(status string) string {
	if status == "open" {
		return StatusForming
	}
	return status
}
