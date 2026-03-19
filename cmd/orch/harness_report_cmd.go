package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	harnessReportDays    int
	harnessReportJSON    bool
	harnessReportVerbose bool
)

var harnessReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show harness measurement report with falsification verdicts",
	Long: `Produce a harness engineering measurement report.

Shows gate deflection rates, accretion velocity, completion field coverage,
and 4 falsification verdicts (ceremony, irrelevant, inert, anecdotal).

Examples:
  orch harness report              # Last 7 days
  orch harness report --days 30    # Last 30 days
  orch harness report --json       # Machine-readable output
  orch harness report --verbose    # Include measurement coverage details`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("harness report", flagsFromCmd(cmd)...)
		return runHarnessReport()
	},
}

func init() {
	harnessReportCmd.Flags().IntVar(&harnessReportDays, "days", 7, "Number of days to analyze")
	harnessReportCmd.Flags().BoolVar(&harnessReportJSON, "json", false, "Output as JSON")
	harnessReportCmd.Flags().BoolVar(&harnessReportVerbose, "verbose", false, "Show measurement coverage details")
	harnessCmd.AddCommand(harnessReportCmd)
}

func runHarnessReport() error {
	eventsPath := getEventsPath()
	events, err := parseEvents(eventsPath)
	if err != nil {
		// No events — show empty report
		resp := buildEmptyHarnessResponse(harnessReportDays)
		return outputHarnessReport(resp)
	}

	resp := buildHarnessResponse(events, harnessReportDays)
	return outputHarnessReport(resp)
}

func outputHarnessReport(resp *HarnessResponse) error {
	if harnessReportJSON {
		output, err := formatHarnessJSON(resp)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}
	fmt.Print(formatHarnessText(resp, harnessReportVerbose))
	return nil
}

func formatHarnessJSON(resp *HarnessResponse) (string, error) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling harness report: %w", err)
	}
	return string(data) + "\n", nil
}

func formatHarnessText(resp *HarnessResponse, verbose bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ HARNESS REPORT (%s, %d spawns) ═══\n\n",
		resp.AnalysisPeriod, resp.TotalSpawns)

	// Gate Deflection
	fmt.Fprintf(&b, "GATE DEFLECTION")
	if resp.TotalSpawns > 0 {
		fmt.Fprintf(&b, " (from %d spawns)", resp.TotalSpawns)
	}
	fmt.Fprintln(&b)

	if len(resp.Pipeline) > 0 {
		for _, stage := range resp.Pipeline {
			for _, comp := range stage.Components {
				if comp.Type != "hard" && comp.Type != "hard+soft" {
					continue
				}
				if comp.FireRate == nil && comp.Bypassed == 0 && comp.Blocked == 0 {
					continue
				}
				fmt.Fprintf(&b, "  %-18s", comp.Name)
				if comp.FireRate != nil {
					fmt.Fprintf(&b, "fire: %5.1f%%", *comp.FireRate*100)
				}
				if comp.BlockRate != nil {
					fmt.Fprintf(&b, "  block: %5.1f%%", *comp.BlockRate*100)
				}
				if comp.BypassRate != nil {
					fmt.Fprintf(&b, "  bypass: %5.1f%%", *comp.BypassRate*100)
				}
				counts := []string{}
				if comp.Bypassed > 0 {
					counts = append(counts, fmt.Sprintf("%d bypasses", comp.Bypassed))
				}
				if comp.Blocked > 0 {
					counts = append(counts, fmt.Sprintf("%d blocks", comp.Blocked))
				}
				if len(counts) > 0 {
					fmt.Fprintf(&b, "  [%s]", strings.Join(counts, ", "))
				}
				fmt.Fprintln(&b)
			}
		}
	}
	if resp.TotalSpawns == 0 {
		fmt.Fprintln(&b, "  No spawn data available.")
	}
	fmt.Fprintln(&b)

	// Accretion Velocity
	fmt.Fprintln(&b, "ACCRETION VELOCITY")
	if resp.AccretionVelocity != nil {
		av := resp.AccretionVelocity
		fmt.Fprintf(&b, "  Pre-gate baseline:  %d lines/week\n", av.BaselineWeeklyLines)
		fmt.Fprintf(&b, "  Post-gate current:  %d lines/week\n", av.CurrentWeeklyLines)
		fmt.Fprintf(&b, "  Change:             %.1f%%\n", av.VelocityChangePct)
		fmt.Fprintf(&b, "  Trend:              %s\n", av.Trend)
	} else {
		fmt.Fprintln(&b, "  No accretion snapshot data available.")
		fmt.Fprintln(&b, "  Checkpoint: Mar 24 (need 2+ weeks post-gate data)")
	}
	fmt.Fprintln(&b)

	// Completion Coverage
	cc := resp.CompletionCoverage
	fmt.Fprintln(&b, "COMPLETION COVERAGE")
	if cc.TotalCompletions > 0 {
		fmt.Fprintf(&b, "  With skill field:   %5.1f%% (%d/%d)\n",
			pct(cc.WithSkill, cc.TotalCompletions), cc.WithSkill, cc.TotalCompletions)
		fmt.Fprintf(&b, "  With outcome field: %5.1f%% (%d/%d)\n",
			pct(cc.WithOutcome, cc.TotalCompletions), cc.WithOutcome, cc.TotalCompletions)
		fmt.Fprintf(&b, "  With duration:      %5.1f%% (%d/%d)\n",
			pct(cc.WithDuration, cc.TotalCompletions), cc.WithDuration, cc.TotalCompletions)
		fmt.Fprintf(&b, "  Overall:            %5.1f%%\n", cc.CoveragePct)
	} else {
		fmt.Fprintln(&b, "  No completion data available.")
	}
	fmt.Fprintln(&b)

	// Falsification Verdicts
	fmt.Fprintln(&b, "FALSIFICATION VERDICTS")
	verdictOrder := []struct {
		key   string
		label string
	}{
		{"gates_are_ceremony", "Gates are ceremony"},
		{"gates_are_irrelevant", "Gates are irrelevant"},
		{"soft_harness_is_inert", "Soft harness is inert"},
		{"framework_is_anecdotal", "Framework is anecdotal"},
	}
	for _, v := range verdictOrder {
		verdict, ok := resp.FalsificationVerdicts[v.key]
		if !ok {
			continue
		}
		sym := verdictSymbol(verdict.Status)
		label := verdictLabel(verdict.Status)
		fmt.Fprintf(&b, "  %s %-24s %s\n", sym, v.label+":", label)
		if verbose {
			fmt.Fprintf(&b, "    Evidence:  %s\n", verdict.Evidence)
			fmt.Fprintf(&b, "    Threshold: %s\n", verdict.Threshold)
		}
	}
	fmt.Fprintln(&b)

	// Verbose: measurement coverage
	if verbose {
		mc := resp.MeasurementCoverage
		fmt.Fprintln(&b, "MEASUREMENT COVERAGE")
		fmt.Fprintf(&b, "  Total components:   %d\n", mc.TotalComponents)
		fmt.Fprintf(&b, "  With measurement:   %d\n", mc.WithMeasurement)
		fmt.Fprintf(&b, "  Proxy only:         %d\n", mc.ProxyOnly)
		fmt.Fprintf(&b, "  Unmeasured:         %d\n", mc.Unmeasured)
		fmt.Fprintln(&b)
	}

	return b.String()
}

func verdictSymbol(status string) string {
	switch status {
	case "falsified":
		return "✓"
	case "confirmed":
		return "✗"
	case "insufficient_data":
		return "…"
	default:
		return "?"
	}
}

func verdictLabel(status string) string {
	switch status {
	case "falsified":
		return "FALSIFIED"
	case "confirmed":
		return "CONFIRMED"
	case "insufficient_data":
		return "INSUFFICIENT DATA"
	case "not_measurable":
		return "NOT MEASURABLE"
	default:
		return strings.ToUpper(status)
	}
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
