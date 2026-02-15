package tree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// DetectHealthSmells detects health smells in a cluster
func DetectHealthSmells(cluster *Cluster, kbDir string) []HealthSmell {
	var smells []HealthSmell

	// Count node types in cluster
	var investigations []*KnowledgeNode
	var decisions []*KnowledgeNode
	var models []*KnowledgeNode

	for _, node := range cluster.Nodes {
		switch node.Type {
		case NodeTypeInvestigation:
			investigations = append(investigations, node)
		case NodeTypeDecision:
			decisions = append(decisions, node)
		case NodeTypeModel:
			models = append(models, node)
		}
	}

	// Smell 1: 15+ investigations without a decision or model
	if len(investigations) >= 15 && len(decisions) == 0 && len(models) == 0 {
		smells = append(smells, HealthSmell{
			Type:        SmellNeedsSynthesis,
			Description: "needs synthesis",
			Count:       len(investigations),
		})
	}

	// Smell 2: Decision without any spawned issues
	for _, decision := range decisions {
		if !hasSpawnedIssues(decision) {
			smells = append(smells, HealthSmell{
				Type:        SmellNotActedOn,
				Description: "not acted on",
				Count:       1,
			})
			// Only report once per cluster, not per decision
			break
		}
	}

	// Smell 3: Model without probes
	for _, model := range models {
		if !hasProbes(model, kbDir) {
			smells = append(smells, HealthSmell{
				Type:        SmellUntestedModel,
				Description: "untested model",
				Count:       1,
			})
			// Only report once per cluster, not per model
			break
		}
	}

	return smells
}

// hasSpawnedIssues checks if a decision has spawned any beads issues
func hasSpawnedIssues(decision *KnowledgeNode) bool {
	// Initialize beads client
	var client beads.BeadsClient
	client = beads.NewCLIClient()

	// Get all issues
	issues, err := client.List(&beads.ListArgs{})
	if err != nil {
		// If we can't check, assume no issues
		return false
	}

	// Extract decision filename from path
	decisionFile := filepath.Base(decision.Path)

	// Check if any issue references this decision
	for _, issue := range issues {
		// Check description for references to this decision file
		if strings.Contains(issue.Description, decisionFile) {
			return true
		}
	}

	return false
}

// hasProbes checks if a model has probe files
func hasProbes(model *KnowledgeNode, kbDir string) bool {
	// Model ID is the directory path
	// Probes should be in {model-dir}/probes/
	modelDir := model.ID

	// Handle both absolute and relative paths
	if !filepath.IsAbs(modelDir) {
		modelDir = filepath.Join(kbDir, modelDir)
	}

	// Ensure modelDir doesn't end with .md (it should be the directory)
	modelDir = strings.TrimSuffix(modelDir, filepath.Base(model.Path))

	probesDir := filepath.Join(modelDir, "probes")

	// Check if probes directory exists and has any .md files
	entries, err := os.ReadDir(probesDir)
	if err != nil {
		// No probes directory or can't read it
		return false
	}

	// Check for any .md files in probes/
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true
		}
	}

	return false
}

// GetSmellBadge returns the badge string for a health smell type
func GetSmellBadge(smellType HealthSmellType) string {
	return "⚠"
}

// FormatSmellDescription formats a smell for display
func FormatSmellDescription(smell HealthSmell) string {
	switch smell.Type {
	case SmellNeedsSynthesis:
		return fmt.Sprintf("⚠ %s", smell.Description)
	case SmellNotActedOn:
		return fmt.Sprintf("⚠ %s", smell.Description)
	case SmellUntestedModel:
		return fmt.Sprintf("⚠ %s", smell.Description)
	default:
		return fmt.Sprintf("⚠ %s", smell.Description)
	}
}
