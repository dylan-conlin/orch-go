package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/advisor"
	"github.com/spf13/cobra"
)

var (
	modelRefresh bool
	modelTask    string
	modelBudget  float64
	modelLimit   int
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Model advisor with live pricing data",
	Long: `Model advisor provides recommendations based on live API data.

Examples:
  orch model list                           # List all models with pricing
  orch model list --refresh                 # Force refresh from API
  orch model recommend --task coding        # Recommend models for coding
  orch model recommend --budget 0.001       # Models under $0.001 per 1K tokens
  orch model cache                          # Show cache info`,
}

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models with live pricing",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := advisor.NewClient()

		var models []advisor.OpenRouterModel
		var err error
		if modelRefresh {
			fmt.Fprintln(os.Stderr, "Fetching fresh data from OpenRouter API...")
			models, err = client.FetchModelsForce()
		} else {
			models, err = client.FetchModels()
			age := client.CacheAge()
			if age >= 0 {
				fmt.Fprintf(os.Stderr, "Using cached data (age: %s). Use --refresh to fetch fresh data.\n", formatDuration(age))
			}
		}

		if err != nil {
			return fmt.Errorf("failed to fetch models: %w", err)
		}

		// Apply limit if specified
		if modelLimit > 0 && modelLimit < len(models) {
			models = models[:modelLimit]
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tINPUT ($/MTok)\tOUTPUT ($/MTok)\tCONTEXT")

		for _, m := range models {
			inputCost := parsePrice(m.Pricing.Prompt)
			outputCost := parsePrice(m.Pricing.Completion)

			// Convert to $/MTok (API returns per-token)
			inputMTok := inputCost * 1_000_000
			outputMTok := outputCost * 1_000_000

			// Format context length
			contextStr := formatContext(m.ContextLength)

			fmt.Fprintf(w, "%s\t%s\t$%.2f\t$%.2f\t%s\n",
				m.ID, m.Name, inputMTok, outputMTok, contextStr)
		}
		w.Flush()

		fmt.Fprintf(os.Stderr, "\nShowing %d models. Run 'orch model recommend' to get task-specific recommendations.\n", len(models))
		return nil
	},
}

var modelRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Recommend models for a task and budget",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := advisor.NewClient()

		models, err := client.FetchModels()
		if err != nil {
			return fmt.Errorf("failed to fetch models: %w", err)
		}

		// Filter and score models
		var scored []scoredModel
		for _, m := range models {
			inputCost := parsePrice(m.Pricing.Prompt) * 1_000_000      // $/MTok
			outputCost := parsePrice(m.Pricing.Completion) * 1_000_000 // $/MTok

			// Average cost (rough estimate for mixed workloads)
			avgCost := (inputCost + outputCost) / 2

			// Apply budget filter if specified
			if modelBudget > 0 && avgCost > modelBudget {
				continue
			}

			// Calculate score based on task
			score := scoreModelForTask(m, modelTask)

			scored = append(scored, scoredModel{
				model:      m,
				score:      score,
				inputCost:  inputCost,
				outputCost: outputCost,
			})
		}

		// Sort by score (descending)
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		// Limit results
		limit := 10
		if modelLimit > 0 {
			limit = modelLimit
		}
		if len(scored) > limit {
			scored = scored[:limit]
		}

		// Print recommendations
		fmt.Printf("Recommendations for task: %s", modelTask)
		if modelBudget > 0 {
			fmt.Printf(" (budget: $%.4f/MTok)", modelBudget)
		}
		fmt.Println()
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RANK\tMODEL\tINPUT\tOUTPUT\tCONTEXT\tSCORE")

		for i, sm := range scored {
			fmt.Fprintf(w, "%d\t%s\t$%.2f\t$%.2f\t%s\t%.1f\n",
				i+1,
				sm.model.Name,
				sm.inputCost,
				sm.outputCost,
				formatContext(sm.model.ContextLength),
				sm.score,
			)
		}
		w.Flush()

		fmt.Fprintf(os.Stderr, "\nShowing top %d models. Use --limit to adjust.\n", len(scored))
		return nil
	},
}

var modelCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Show cache info",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := advisor.NewClient()
		age := client.CacheAge()

		if age < 0 {
			fmt.Println("Cache: not found")
			fmt.Println("\nRun 'orch model list' to fetch and cache model data.")
			return nil
		}

		fmt.Printf("Cache age: %s\n", formatDuration(age))
		fmt.Printf("TTL: 24 hours\n")
		if age > 24*time.Hour {
			fmt.Println("Status: stale (will refresh on next fetch)")
		} else {
			remaining := 24*time.Hour - age
			fmt.Printf("Status: fresh (expires in %s)\n", formatDuration(remaining))
		}

		return nil
	},
}

type scoredModel struct {
	model      advisor.OpenRouterModel
	score      float64
	inputCost  float64
	outputCost float64
}

// scoreModelForTask assigns a score to a model based on task type.
// Higher score = better fit.
func scoreModelForTask(m advisor.OpenRouterModel, task string) float64 {
	task = strings.ToLower(task)

	// Base score
	score := 50.0

	// Task-specific scoring
	switch task {
	case "coding", "code", "programming":
		// Prefer models with "code", "codex", "devstral" in name
		nameLower := strings.ToLower(m.Name)
		if strings.Contains(nameLower, "codex") {
			score += 30
		} else if strings.Contains(nameLower, "code") || strings.Contains(nameLower, "devstral") {
			score += 20
		}
		// Prefer large context for codebase work
		if m.ContextLength >= 200000 {
			score += 10
		}

	case "reasoning", "think", "analysis":
		// Prefer reasoning models
		nameLower := strings.ToLower(m.Name)
		if strings.Contains(nameLower, "think") || strings.Contains(nameLower, "reasoning") || strings.Contains(nameLower, "deepseek-r") {
			score += 30
		}
		if strings.Contains(nameLower, "pro") || strings.Contains(nameLower, "opus") {
			score += 15
		}

	case "chat", "conversation", "general":
		// Prefer conversational models
		nameLower := strings.ToLower(m.Name)
		if strings.Contains(nameLower, "chat") || strings.Contains(nameLower, "instruct") {
			score += 20
		}

	case "vision", "image", "multimodal":
		// Prefer multimodal models
		if len(m.Architecture.InputModalities) > 1 {
			score += 25
		}
		if modelSliceContains(m.Architecture.InputModalities, "image") {
			score += 15
		}
	}

	// Cost efficiency bonus (cheaper = better for equal capability)
	inputCost := parsePrice(m.Pricing.Prompt) * 1_000_000
	if inputCost < 0.5 {
		score += 10 // Very cheap
	} else if inputCost > 10 {
		score -= 5 // Expensive
	}

	// Context length bonus
	if m.ContextLength >= 1000000 {
		score += 5 // 1M+ context
	}

	return score
}

// parsePrice converts string price to float64.
func parsePrice(s string) float64 {
	if s == "" || s == "0" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// formatContext formats context length in human-readable form.
func formatContext(length int) string {
	if length >= 1000000 {
		return fmt.Sprintf("%dM", length/1000000)
	}
	if length >= 1000 {
		return fmt.Sprintf("%dK", length/1000)
	}
	return fmt.Sprintf("%d", length)
}

// modelSliceContains checks if a slice contains a string.
func modelSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(modelCmd)
	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelRecommendCmd)
	modelCmd.AddCommand(modelCacheCmd)

	// Flags for list command
	modelListCmd.Flags().BoolVar(&modelRefresh, "refresh", false, "Force refresh from API (bypass cache)")
	modelListCmd.Flags().IntVar(&modelLimit, "limit", 50, "Maximum number of models to show")

	// Flags for recommend command
	modelRecommendCmd.Flags().StringVar(&modelTask, "task", "general", "Task type (coding, reasoning, chat, vision, general)")
	modelRecommendCmd.Flags().Float64Var(&modelBudget, "budget", 0, "Maximum cost in $/MTok (0 = no limit)")
	modelRecommendCmd.Flags().IntVar(&modelLimit, "limit", 10, "Maximum number of recommendations")
}
