package gates

import (
	"fmt"
	"os"
	"strings"
)

// HotspotResult contains the result of a spawn hotspot check.
// This is a minimal interface used by the gate; the full analysis
// lives in cmd/orch/hotspot.go.
type HotspotResult struct {
	HasHotspots        bool
	HasCriticalHotspot bool     // True when any matched hotspot is CRITICAL (>1500 lines)
	Warning            string
	CriticalFiles      []string // File paths of CRITICAL hotspots
	MatchedFiles       []string // All matched hotspot file/topic paths (for context injection)
}

// HotspotChecker is a function that runs hotspot analysis for a given project directory and task.
// Returns nil if no hotspots were detected.
type HotspotChecker func(projectDir, task string) (*HotspotResult, error)

// ArchitectVerifier validates that an architect issue exists and is closed.
// Returns nil if the issue is a valid closed architect issue, error otherwise.
// The caller constructs this from verify.GetIssue to keep the gates package decoupled.
type ArchitectVerifier func(issueID string) error

// blockingSkills are skills that modify code and should be blocked on CRITICAL hotspots.
// Read-only/strategic skills are exempt because they need to READ hotspot files.
var blockingSkills = map[string]bool{
	"feature-impl":         true,
	"systematic-debugging": true,
}

// IsBlockingSkill returns true if the skill should be blocked on CRITICAL hotspots.
func IsBlockingSkill(skillName string) bool {
	return blockingSkills[skillName]
}

// CheckHotspot runs hotspot analysis and displays warnings if the task targets a high-churn area.
// The checker function performs the actual hotspot analysis (injected from cmd/orch).
// daemonDriven spawns suppress output (triage already happened).
// forceHotspot bypasses the blocking gate but requires architectRef with a verified closed
// architect issue (validated via architectVerifier).
// Returns error if skill is blocked by CRITICAL hotspot and requirements are not met.
func CheckHotspot(projectDir, task, skillName string, daemonDriven, forceHotspot bool, architectRef string, checker HotspotChecker, architectVerifier ArchitectVerifier) (*HotspotResult, error) {
	if projectDir == "" || checker == nil {
		return nil, nil
	}

	result, err := checker(projectDir, task)
	if err != nil || result == nil {
		return nil, nil
	}

	// Daemon-driven spawns stay silent (triage already happened)
	if daemonDriven {
		return result, nil
	}

	// Show hotspot warning (includes recommendation to use architect)
	fmt.Fprint(os.Stderr, result.Warning)

	// Check if this skill should be blocked on CRITICAL hotspots
	if result.HasCriticalHotspot && IsBlockingSkill(skillName) {
		if forceHotspot {
			// --force-hotspot requires --architect-ref to prove architect reviewed the area
			if architectRef == "" {
				return result, fmt.Errorf("--force-hotspot requires --architect-ref <issue-id> to prove architect reviewed the area.\nSpawn architect first, then reference its closed issue.")
			}

			// Verify the architect issue if verifier is available
			if architectVerifier != nil {
				if err := architectVerifier(architectRef); err != nil {
					return result, err
				}
			}

			fmt.Fprintf(os.Stderr, "✓ --force-hotspot: Bypassing CRITICAL hotspot block (architect-ref: %s)\n", architectRef)
			fmt.Fprintln(os.Stderr, "")
			return result, nil
		}
		criticalList := strings.Join(result.CriticalFiles, ", ")
		return result, fmt.Errorf("CRITICAL hotspot: %s exceeds 1500 lines. Spawn architect to design extraction first, or use --force-hotspot --architect-ref <issue-id> to override.\nBlocked files: %s", criticalList, criticalList)
	}

	// Add context based on skill choice
	isExemptSkill := !IsBlockingSkill(skillName)
	if isExemptSkill {
		fmt.Fprintln(os.Stderr, "✓ Strategic/read-only skill: exempt from hotspot blocking")
	} else {
		fmt.Fprintln(os.Stderr, "⚠️  Proceeding with tactical approach in hotspot area")
	}
	fmt.Fprintln(os.Stderr, "")

	return result, nil
}
