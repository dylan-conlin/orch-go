package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// BatchSynthesis holds the cross-agent synthesis for a batch of completions.
type BatchSynthesis struct {
	Project        string
	AgentCount     int
	SynthesisCount int
	LightTierCount int
	Findings       []AgentFinding
	NextActions    []BatchNextAction
	OpenQuestions  []BatchQuestion
	Connections    []Connection
}

// AgentFinding holds a single agent's contribution to the batch synthesis.
type AgentFinding struct {
	WorkspaceID string
	BeadsID     string
	Skill       string
	TLDR        string
	Outcome     string
	Knowledge   string
}

// BatchNextAction is a deduplicated next action with source agents.
type BatchNextAction struct {
	Action  string
	Sources []string // workspace IDs that recommended this
}

// BatchQuestion is an open question from an agent.
type BatchQuestion struct {
	Question string
	Source   string // workspace ID
}

// Connection represents a cross-agent link (shared work, related findings).
type Connection struct {
	Description string
	Agents      []string
}

// rawNextAction is an intermediate type for deduplication.
type rawNextAction struct {
	action string
	source string
}

var reviewSynthesizeCmd = &cobra.Command{
	Use:     "synthesize [project]",
	Aliases: []string{"synth"},
	Short:   "Produce batch synthesis across completed agents",
	Long: `Collect completed agent artifacts and produce a cross-referenced synthesis.

Instead of reviewing agents one-by-one, this command reads all SYNTHESIS.md
files and investigation artifacts for a project, cross-references findings,
and outputs what changed in understanding.

Sections:
  WHAT WE NOW KNOW    - Agent findings with TLDR and knowledge
  NEXT ACTIONS         - Deduplicated recommendations (multi-source highlighted)
  OPEN QUESTIONS       - Unexplored questions across agents
  CONNECTIONS          - Cross-agent links (shared actions, related findings)

Use this before 'orch review done' to understand what agents produced together.

Examples:
  orch review synthesize orch-go      # Full batch synthesis
  orch review synth orch-go           # Short alias`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewSynthesize(args[0])
	},
}

func init() {
	reviewCmd.AddCommand(reviewSynthesizeCmd)
}

func runReviewSynthesize(project string) error {
	completions, err := getCompletionsForReview()
	if err != nil {
		return err
	}

	// Filter by project
	var projectCompletions []CompletionInfo
	for _, c := range completions {
		if c.Project == project {
			projectCompletions = append(projectCompletions, c)
		}
	}

	if len(projectCompletions) == 0 {
		fmt.Printf("No pending completions for project: %s\n", project)
		return nil
	}

	// Enrich completions that don't have synthesis yet (from workspace paths)
	for i := range projectCompletions {
		c := &projectCompletions[i]
		if c.Synthesis == nil && c.WorkspacePath != "" && !c.IsLightTier {
			s, err := verify.ParseSynthesis(c.WorkspacePath)
			if err == nil {
				c.Synthesis = s
			}
		}
	}

	batch := buildBatchSynthesis(projectCompletions, project)
	fmt.Print(formatBatchSynthesis(batch))

	// Suggest next step
	if batch.AgentCount > 0 {
		fmt.Printf("\nTo complete these agents: orch review done %s\n", project)
	}

	return nil
}

// buildBatchSynthesis aggregates findings across completed agents.
func buildBatchSynthesis(completions []CompletionInfo, project string) BatchSynthesis {
	batch := BatchSynthesis{
		Project:    project,
		AgentCount: len(completions),
	}

	var allNextActions []rawNextAction

	for _, c := range completions {
		if c.IsLightTier {
			batch.LightTierCount++
		}

		if c.Synthesis == nil {
			continue
		}

		batch.SynthesisCount++

		// Extract finding
		finding := AgentFinding{
			WorkspaceID: c.WorkspaceID,
			BeadsID:     c.BeadsID,
			Skill:       c.Skill,
			TLDR:        c.Synthesis.TLDR,
			Outcome:     c.Synthesis.Outcome,
			Knowledge:   c.Synthesis.Knowledge,
		}
		batch.Findings = append(batch.Findings, finding)

		// Collect next actions for deduplication
		for _, action := range c.Synthesis.NextActions {
			allNextActions = append(allNextActions, rawNextAction{
				action: action,
				source: c.WorkspaceID,
			})
		}

		// Collect open questions
		if c.Synthesis.UnexploredQuestions != "" {
			// Split on newlines and extract individual questions
			for _, line := range strings.Split(c.Synthesis.UnexploredQuestions, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "- ")
				line = strings.TrimPrefix(line, "* ")
				line = strings.TrimPrefix(line, "? ")
				line = strings.TrimSpace(line)
				if line != "" {
					batch.OpenQuestions = append(batch.OpenQuestions, BatchQuestion{
						Question: line,
						Source:   c.WorkspaceID,
					})
				}
			}
		}
	}

	// Deduplicate next actions
	batch.NextActions = deduplicateNextActions(allNextActions)

	// Detect connections (actions recommended by multiple agents)
	for _, na := range batch.NextActions {
		if len(na.Sources) > 1 {
			batch.Connections = append(batch.Connections, Connection{
				Description: fmt.Sprintf("Shared next action: %s", na.Action),
				Agents:      na.Sources,
			})
		}
	}

	return batch
}

// deduplicateNextActions merges duplicate actions (case-insensitive) and tracks sources.
func deduplicateNextActions(actions []rawNextAction) []BatchNextAction {
	// Use lowercase key for dedup, preserve original casing from first occurrence
	type entry struct {
		original string
		sources  []string
	}
	seen := make(map[string]*entry)
	var order []string

	for _, a := range actions {
		key := strings.ToLower(strings.TrimSpace(a.action))
		if key == "" {
			continue
		}
		if e, ok := seen[key]; ok {
			// Add source if not already present
			found := false
			for _, s := range e.sources {
				if s == a.source {
					found = true
					break
				}
			}
			if !found {
				e.sources = append(e.sources, a.source)
			}
		} else {
			seen[key] = &entry{
				original: a.action,
				sources:  []string{a.source},
			}
			order = append(order, key)
		}
	}

	var result []BatchNextAction
	for _, key := range order {
		e := seen[key]
		result = append(result, BatchNextAction{
			Action:  e.original,
			Sources: e.sources,
		})
	}
	return result
}

// formatBatchSynthesis renders the batch synthesis for terminal output.
func formatBatchSynthesis(batch BatchSynthesis) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("\n## BATCH SYNTHESIS: %s\n", batch.Project))
	b.WriteString(fmt.Sprintf("Agents: %d total", batch.AgentCount))
	if batch.SynthesisCount > 0 {
		b.WriteString(fmt.Sprintf(", %d with synthesis", batch.SynthesisCount))
	}
	if batch.LightTierCount > 0 {
		b.WriteString(fmt.Sprintf(", %d light-tier", batch.LightTierCount))
	}
	b.WriteString("\n")

	// WHAT WE NOW KNOW
	if len(batch.Findings) > 0 {
		b.WriteString("\n### WHAT WE NOW KNOW\n\n")
		for _, f := range batch.Findings {
			skillBadge := ""
			if f.Skill != "" {
				skillBadge = fmt.Sprintf(" [%s]", f.Skill)
			}
			beadsRef := ""
			if f.BeadsID != "" {
				beadsRef = fmt.Sprintf(" (%s)", f.BeadsID)
			}
			b.WriteString(fmt.Sprintf("**%s**%s%s\n", f.WorkspaceID, skillBadge, beadsRef))

			if f.TLDR != "" {
				b.WriteString(fmt.Sprintf("  TLDR: %s\n", f.TLDR))
			}
			if f.Knowledge != "" {
				// Truncate long knowledge sections
				knowledge := f.Knowledge
				if len(knowledge) > 200 {
					knowledge = knowledge[:197] + "..."
				}
				// Collapse newlines for compact display
				knowledge = strings.ReplaceAll(knowledge, "\n", " ")
				b.WriteString(fmt.Sprintf("  Learned: %s\n", knowledge))
			}
			if f.Outcome != "" {
				b.WriteString(fmt.Sprintf("  Outcome: %s\n", f.Outcome))
			}
			b.WriteString("\n")
		}
	}

	// NEXT ACTIONS (deduplicated, multi-source highlighted)
	if len(batch.NextActions) > 0 {
		b.WriteString("### NEXT ACTIONS\n\n")
		for _, na := range batch.NextActions {
			action := na.Action
			if len(action) > 100 {
				action = action[:97] + "..."
			}
			if len(na.Sources) > 1 {
				b.WriteString(fmt.Sprintf("  * %s  ← %s\n", action, strings.Join(na.Sources, ", ")))
			} else {
				b.WriteString(fmt.Sprintf("  - %s  (%s)\n", action, na.Sources[0]))
			}
		}
		b.WriteString("\n")
	}

	// OPEN QUESTIONS
	if len(batch.OpenQuestions) > 0 {
		b.WriteString("### OPEN QUESTIONS\n\n")
		for _, q := range batch.OpenQuestions {
			question := q.Question
			if len(question) > 120 {
				question = question[:117] + "..."
			}
			b.WriteString(fmt.Sprintf("  ? %s  (%s)\n", question, q.Source))
		}
		b.WriteString("\n")
	}

	// CONNECTIONS
	if len(batch.Connections) > 0 {
		b.WriteString("### CONNECTIONS\n\n")
		for _, conn := range batch.Connections {
			b.WriteString(fmt.Sprintf("  %s\n", conn.Description))
			b.WriteString(fmt.Sprintf("    Agents: %s\n", strings.Join(conn.Agents, ", ")))
		}
		b.WriteString("\n")
	}

	// Empty state
	if len(batch.Findings) == 0 && batch.LightTierCount > 0 {
		b.WriteString("\nAll agents are light-tier (no synthesis artifacts).\n")
		b.WriteString("Use 'orch review' for the standard completion list.\n")
	}

	return b.String()
}
