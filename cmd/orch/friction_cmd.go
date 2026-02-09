package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/friction"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	frictionSymptom  string
	frictionImpact   string
	frictionEvidence string
	frictionIssue    string

	frictionListIssue string
	frictionLimit     int
	frictionJSON      bool
)

var frictionCmd = &cobra.Command{
	Use:   "friction",
	Short: "Capture and review orchestration friction incidents",
	Long: `Capture orchestration friction in a lightweight ledger so repeated failures
are visible and synthesizable.

Each ledger entry captures:
- symptom (what failed)
- impact (why it matters)
- evidence path (where proof lives)
- linked issue (where follow-up happens)

Examples:
  orch friction log --symptom "Duplicate spawn" --impact "Wasted daemon slot" --evidence .kb/investigations/2026-02-08-dup.md --issue orch-go-21409
  orch friction list
  orch friction summary --issue orch-go-21409`,
}

var frictionLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Log one friction incident",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFrictionLog()
	},
}

var frictionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent friction incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFrictionList()
	},
}

var frictionSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show repeated friction patterns by symptom",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFrictionSummary()
	},
}

func init() {
	frictionLogCmd.Flags().StringVar(&frictionSymptom, "symptom", "", "Symptom observed (required)")
	frictionLogCmd.Flags().StringVar(&frictionImpact, "impact", "", "Impact of the friction (required)")
	frictionLogCmd.Flags().StringVar(&frictionEvidence, "evidence", "", "Path to evidence artifact (required)")
	frictionLogCmd.Flags().StringVar(&frictionIssue, "issue", "", "Linked beads issue (required)")
	frictionLogCmd.MarkFlagRequired("symptom")
	frictionLogCmd.MarkFlagRequired("impact")
	frictionLogCmd.MarkFlagRequired("evidence")
	frictionLogCmd.MarkFlagRequired("issue")

	frictionListCmd.Flags().StringVar(&frictionListIssue, "issue", "", "Filter by linked issue")
	frictionListCmd.Flags().IntVar(&frictionLimit, "limit", 20, "Maximum entries to show")
	frictionListCmd.Flags().BoolVar(&frictionJSON, "json", false, "Output as JSON")

	frictionSummaryCmd.Flags().StringVar(&frictionListIssue, "issue", "", "Filter by linked issue")
	frictionSummaryCmd.Flags().BoolVar(&frictionJSON, "json", false, "Output as JSON")

	frictionCmd.AddCommand(frictionLogCmd)
	frictionCmd.AddCommand(frictionListCmd)
	frictionCmd.AddCommand(frictionSummaryCmd)
	rootCmd.AddCommand(frictionCmd)
}

func runFrictionLog() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}

	resolvedIssue, err := resolveShortBeadsID(strings.TrimSpace(frictionIssue))
	if err != nil {
		return fmt.Errorf("resolve linked issue %s: %w", frictionIssue, err)
	}

	if _, err := verify.GetIssue(resolvedIssue); err != nil {
		return fmt.Errorf("validate linked issue %s: %w", resolvedIssue, err)
	}

	evidencePath, err := normalizeEvidencePath(projectDir, frictionEvidence)
	if err != nil {
		return err
	}

	ledgerPath := frictionLedgerPath(projectDir)
	written, err := friction.Append(ledgerPath, friction.Entry{
		Symptom:      strings.TrimSpace(frictionSymptom),
		Impact:       strings.TrimSpace(frictionImpact),
		EvidencePath: evidencePath,
		LinkedIssue:  resolvedIssue,
	})
	if err != nil {
		return fmt.Errorf("append friction entry: %w", err)
	}

	logFrictionEvent(written)
	reportFrictionToIssue(written)

	fmt.Printf("Logged friction incident %s\n", written.ID)
	fmt.Printf("  Symptom:  %s\n", written.Symptom)
	fmt.Printf("  Impact:   %s\n", written.Impact)
	fmt.Printf("  Evidence: %s\n", written.EvidencePath)
	fmt.Printf("  Issue:    %s\n", written.LinkedIssue)
	fmt.Printf("  Ledger:   %s\n", ledgerPath)

	return nil
}

func runFrictionList() error {
	entries, err := loadFilteredFrictionEntries(frictionListIssue)
	if err != nil {
		return err
	}

	if frictionLimit > 0 && len(entries) > frictionLimit {
		entries = entries[:frictionLimit]
	}

	if frictionJSON {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal entries: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No friction incidents logged.")
		fmt.Println("Use 'orch friction log --symptom ... --impact ... --evidence ... --issue ...' to record one.")
		return nil
	}

	fmt.Printf("Friction incidents: %d\n\n", len(entries))
	for _, e := range entries {
		fmt.Printf("[%s] %s\n", e.Timestamp.Format(time.RFC3339), e.ID)
		fmt.Printf("  Symptom:  %s\n", e.Symptom)
		fmt.Printf("  Impact:   %s\n", e.Impact)
		fmt.Printf("  Evidence: %s\n", e.EvidencePath)
		fmt.Printf("  Issue:    %s\n\n", e.LinkedIssue)
	}

	return nil
}

func runFrictionSummary() error {
	entries, err := loadFilteredFrictionEntries(frictionListIssue)
	if err != nil {
		return err
	}

	summary := friction.Summarize(entries)

	if frictionJSON {
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal summary: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(summary) == 0 {
		fmt.Println("No friction incidents logged.")
		return nil
	}

	fmt.Printf("Repeated friction patterns: %d\n\n", len(summary))
	for _, s := range summary {
		issues := "(none)"
		if len(s.LinkedIssues) > 0 {
			issues = strings.Join(s.LinkedIssues, ", ")
		}
		fmt.Printf("%dx %s\n", s.Count, s.Symptom)
		fmt.Printf("  Last seen: %s\n", s.LastSeen.Format(time.RFC3339))
		fmt.Printf("  Latest impact: %s\n", s.LatestImpact)
		fmt.Printf("  Latest evidence: %s\n", s.LatestEvidence)
		fmt.Printf("  Linked issues: %s\n\n", issues)
	}

	return nil
}

func loadFilteredFrictionEntries(issueFilter string) ([]friction.Entry, error) {
	projectDir, err := currentProjectDir()
	if err != nil {
		return nil, fmt.Errorf("resolve project directory: %w", err)
	}

	entries, err := friction.Load(frictionLedgerPath(projectDir))
	if err != nil {
		return nil, fmt.Errorf("load friction ledger: %w", err)
	}

	if strings.TrimSpace(issueFilter) != "" {
		resolvedIssue, err := resolveShortBeadsID(strings.TrimSpace(issueFilter))
		if err != nil {
			return nil, fmt.Errorf("resolve issue filter %s: %w", issueFilter, err)
		}
		filtered := make([]friction.Entry, 0, len(entries))
		for _, e := range entries {
			if e.LinkedIssue == resolvedIssue {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

func normalizeEvidencePath(projectDir, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("evidence path is required")
	}

	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(projectDir, path)
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("evidence path not found: %s", path)
	}

	rel, err := filepath.Rel(projectDir, abs)
	if err != nil {
		return filepath.Clean(path), nil
	}
	if strings.HasPrefix(rel, "..") {
		return abs, nil
	}

	return filepath.Clean(rel), nil
}

func frictionLedgerPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", "friction-ledger.jsonl")
}

func logFrictionEvent(entry friction.Entry) {
	logger := events.NewDefaultLogger()
	_ = logger.Log(events.Event{
		Type:      events.EventTypeFrictionLogged,
		SessionID: entry.LinkedIssue,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"entry_id":      entry.ID,
			"symptom":       entry.Symptom,
			"impact":        entry.Impact,
			"evidence_path": entry.EvidencePath,
			"linked_issue":  entry.LinkedIssue,
		},
	})
}

func reportFrictionToIssue(entry friction.Entry) {
	comment := fmt.Sprintf("Friction: %s\nImpact: %s\nEvidence: %s\nLedger: .orch/friction-ledger.jsonl", entry.Symptom, entry.Impact, entry.EvidencePath)
	if err := beads.FallbackAddComment(entry.LinkedIssue, comment); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to post friction comment to %s: %v\n", entry.LinkedIssue, err)
	}
}
