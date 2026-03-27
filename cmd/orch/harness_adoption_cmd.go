package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	adoptionJSON bool
)

var harnessAdoptionCmd = &cobra.Command{
	Use:   "adoption",
	Short: "Measure compositional signal adoption rates across artifact types",
	Long: `Measures adoption rates for compositional signals that enable knowledge
composition. Surfaces drift from targets so silent degradation is caught
before signals go effectively dead.

Signals measured:
  - Investigation model link rate (target: 80%)
  - Brief tension rate (target: 100%)
  - Probe claim/verdict rate (target: 80%)
  - Thread resolved_to rate (target: 80%)
  - Beads enrichment label rate (target: 80%)
  - Decision Extends rate (target: 50%)

Based on compositional-accretion model: signals below 80% adoption
behave identically to no signal. Only opt-out signals achieve >80%.

Examples:
  orch harness adoption         # Table with current vs target
  orch harness adoption --json  # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("harness adoption", flagsFromCmd(cmd)...)
		return runHarnessAdoption()
	},
}

func init() {
	harnessAdoptionCmd.Flags().BoolVar(&adoptionJSON, "json", false, "Output as JSON")
	harnessCmd.AddCommand(harnessAdoptionCmd)
}

// AdoptionResult is the top-level result for adoption measurement.
type AdoptionResult struct {
	GeneratedAt string           `json:"generated_at"`
	Signals     []AdoptionSignal `json:"signals"`
	Alerts      []AdoptionAlert  `json:"alerts,omitempty"`
}

// AdoptionSignal holds the measured adoption rate for one compositional signal.
type AdoptionSignal struct {
	Name      string  `json:"name"`
	Surface   string  `json:"surface"`    // artifact type being measured
	Total     int     `json:"total"`      // total artifacts
	Adopted   int     `json:"adopted"`    // artifacts with signal filled
	RatePct   float64 `json:"rate_pct"`   // adopted/total * 100
	TargetPct float64 `json:"target_pct"` // expected adoption rate
	Status    string  `json:"status"`     // "ok", "drift", "critical"
}

// AdoptionAlert flags a signal that has drifted below target.
type AdoptionAlert struct {
	Signal  string `json:"signal"`
	Level   string `json:"level"` // "drift" (<target) or "critical" (<50% of target)
	Message string `json:"message"`
}

func runHarnessAdoption() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	result := measureAdoption(projectDir)

	if adoptionJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Print(formatAdoptionText(result))
	return nil
}

func measureAdoption(projectDir string) *AdoptionResult {
	result := &AdoptionResult{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	kbDir := filepath.Join(projectDir, ".kb")
	beadsDir := filepath.Join(projectDir, ".beads")

	result.Signals = append(result.Signals, measureInvestigationModelLink(kbDir))
	result.Signals = append(result.Signals, measureBriefTension(kbDir))
	result.Signals = append(result.Signals, measureProbeClaimRate(kbDir))
	result.Signals = append(result.Signals, measureProbeVerdictRate(kbDir))
	result.Signals = append(result.Signals, measureThreadResolvedTo(kbDir))
	result.Signals = append(result.Signals, measureBeadsEnrichment(beadsDir))
	result.Signals = append(result.Signals, measureDecisionExtends(kbDir))

	// Generate alerts for signals below target
	for _, sig := range result.Signals {
		if sig.Status == "critical" {
			result.Alerts = append(result.Alerts, AdoptionAlert{
				Signal:  sig.Name,
				Level:   "critical",
				Message: fmt.Sprintf("%.0f%% adoption (target %.0f%%) — signal is effectively dead", sig.RatePct, sig.TargetPct),
			})
		} else if sig.Status == "drift" {
			result.Alerts = append(result.Alerts, AdoptionAlert{
				Signal:  sig.Name,
				Level:   "drift",
				Message: fmt.Sprintf("%.0f%% adoption (target %.0f%%)", sig.RatePct, sig.TargetPct),
			})
		}
	}

	return result
}

func makeSignal(name, surface string, total, adopted int, target float64) AdoptionSignal {
	rate := 0.0
	if total > 0 {
		rate = float64(adopted) / float64(total) * 100
	}
	status := "ok"
	if total > 0 && rate < target/2 {
		status = "critical"
	} else if total > 0 && rate < target {
		status = "drift"
	}
	return AdoptionSignal{
		Name:      name,
		Surface:   surface,
		Total:     total,
		Adopted:   adopted,
		RatePct:   rate,
		TargetPct: target,
		Status:    status,
	}
}

// measureInvestigationModelLink counts active investigations with **Model:** field.
func measureInvestigationModelLink(kbDir string) AdoptionSignal {
	invDir := filepath.Join(kbDir, "investigations")
	files := listMDFiles(invDir)
	total := len(files)
	adopted := 0
	for _, f := range files {
		if fileContainsLine(f, "**Model:**") {
			adopted++
		}
	}
	return makeSignal("Investigation model link", "investigations", total, adopted, 80)
}

// measureBriefTension counts briefs with ## Tension section.
func measureBriefTension(kbDir string) AdoptionSignal {
	briefDir := filepath.Join(kbDir, "briefs")
	files := listMDFiles(briefDir)
	total := len(files)
	adopted := 0
	for _, f := range files {
		if fileContainsLine(f, "## Tension") {
			adopted++
		}
	}
	return makeSignal("Brief tension", "briefs", total, adopted, 100)
}

// measureProbeClaimRate counts probes with claim: in frontmatter.
// Handles both **claim:** (bold) and claim: (plain) formats.
func measureProbeClaimRate(kbDir string) AdoptionSignal {
	total, adopted := 0, 0
	for _, f := range listAllProbeFiles(kbDir) {
		total++
		if fileHasFrontmatterField(f, "claim:") {
			adopted++
		}
	}
	return makeSignal("Probe claim", "probes", total, adopted, 80)
}

// measureProbeVerdictRate counts probes with verdict: in frontmatter.
func measureProbeVerdictRate(kbDir string) AdoptionSignal {
	total, adopted := 0, 0
	for _, f := range listAllProbeFiles(kbDir) {
		total++
		if fileHasFrontmatterField(f, "verdict:") {
			adopted++
		}
	}
	return makeSignal("Probe verdict", "probes", total, adopted, 80)
}

// listAllProbeFiles returns all .md files under .kb/models/*/probes/.
func listAllProbeFiles(kbDir string) []string {
	modelsDir := filepath.Join(kbDir, "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		probesDir := filepath.Join(modelsDir, entry.Name(), "probes")
		files = append(files, listMDFiles(probesDir)...)
	}
	return files
}

// fileHasFrontmatterField checks if a file has a given field in its frontmatter
// (first 15 lines). Matches both **field:** (bold) and field: (plain) formats.
func fileHasFrontmatterField(path, field string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	boldField := "**" + field
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() && lineNum < 15 {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, field) || strings.HasPrefix(line, boldField) {
			return true
		}
	}
	return false
}

// measureThreadResolvedTo counts threads with non-empty resolved_to field.
func measureThreadResolvedTo(kbDir string) AdoptionSignal {
	threadsDir := filepath.Join(kbDir, "threads")
	files := listMDFiles(threadsDir)
	total := len(files)
	adopted := 0
	for _, f := range files {
		if fileHasNonEmptyField(f, "resolved_to:") {
			adopted++
		}
	}
	return makeSignal("Thread resolved_to", "threads", total, adopted, 80)
}

// measureBeadsEnrichment counts issues with at least one routing label (skill:, area:, effort:).
func measureBeadsEnrichment(beadsDir string) AdoptionSignal {
	issuesPath := filepath.Join(beadsDir, "issues.jsonl")
	file, err := os.Open(issuesPath)
	if err != nil {
		return makeSignal("Beads enrichment", "issues", 0, 0, 80)
	}
	defer file.Close()

	total, adopted := 0, 0
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var issue struct {
			Labels []string `json:"labels"`
		}
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			continue
		}
		total++
		for _, label := range issue.Labels {
			if strings.HasPrefix(label, "skill:") ||
				strings.HasPrefix(label, "area:") ||
				strings.HasPrefix(label, "effort:") {
				adopted++
				break
			}
		}
	}
	return makeSignal("Beads enrichment", "issues", total, adopted, 80)
}

// measureDecisionExtends counts decisions with **Extends: field.
func measureDecisionExtends(kbDir string) AdoptionSignal {
	decisionsDir := filepath.Join(kbDir, "decisions")
	files := listMDFiles(decisionsDir)
	total := len(files)
	adopted := 0
	for _, f := range files {
		if fileContainsLine(f, "**Extends:") {
			adopted++
		}
	}
	return makeSignal("Decision Extends", "decisions", total, adopted, 50)
}

// listMDFiles returns all .md file paths in a directory (non-recursive).
func listMDFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths
}

// fileContainsLine checks if any line in a file starts with the given prefix.
func fileContainsLine(path, prefix string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), prefix) {
			return true
		}
	}
	return false
}

// fileHasNonEmptyField checks if a file has a field line where the value after the prefix is non-empty.
func fileHasNonEmptyField(path, prefix string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if value != "" && value != `""` && value != `''` {
				return true
			}
		}
	}
	return false
}

func formatAdoptionText(result *AdoptionResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ ADOPTION RATES ═══\n\n")

	fmt.Fprintf(&b, "  %-28s %6s %6s %7s %7s  %s\n",
		"SIGNAL", "TOTAL", "ADOPT", "RATE", "TARGET", "STATUS")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 72))

	for _, sig := range result.Signals {
		statusStr := "ok"
		switch sig.Status {
		case "drift":
			statusStr = "DRIFT"
		case "critical":
			statusStr = "CRITICAL"
		}
		fmt.Fprintf(&b, "  %-28s %6d %6d %6.0f%% %6.0f%%  %s\n",
			sig.Name, sig.Total, sig.Adopted, sig.RatePct, sig.TargetPct, statusStr)
	}
	fmt.Fprintln(&b)

	if len(result.Alerts) > 0 {
		fmt.Fprintln(&b, "ALERTS")
		for _, a := range result.Alerts {
			icon := "!"
			if a.Level == "critical" {
				icon = "!!!"
			}
			fmt.Fprintf(&b, "  [%s] %s: %s\n", icon, a.Signal, a.Message)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}
