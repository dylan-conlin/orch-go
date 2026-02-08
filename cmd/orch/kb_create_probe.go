package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var kbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create kb artifacts from templates",
}

var kbCreateProbeCmd = &cobra.Command{
	Use:   "probe <model-name> <slug>",
	Short: "Create a probe file for a model",
	Long: `Create a model probe at .kb/models/{model-name}/probes/YYYY-MM-DD-{slug}.md.

The model must already exist in .kb/models/{model-name}.md.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := runKBCreateProbe(args[0], args[1])
		if err != nil {
			return err
		}

		fmt.Println(path)
		return nil
	},
}

func init() {
	kbCreateCmd.AddCommand(kbCreateProbeCmd)
	kbCmd.AddCommand(kbCreateCmd)
}

func runKBCreateProbe(modelName, slug string) (string, error) {
	projectDir, err := currentProjectDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine current project directory: %w", err)
	}

	return createProbeFile(projectDir, modelName, slug, time.Now())
}

func createProbeFile(projectDir, modelName, slug string, now time.Time) (string, error) {
	modelPath, normalizedModelName, err := resolveProbeModelPath(projectDir, modelName)
	if err != nil {
		return "", err
	}

	normalizedSlug, err := normalizeProbeSlug(slug)
	if err != nil {
		return "", err
	}

	probeDir, err := spawn.EnsureProbesDir(modelPath)
	if err != nil {
		return "", err
	}

	dateStamp := now.Format("2006-01-02")
	probePath := filepath.Join(probeDir, fmt.Sprintf("%s-%s.md", dateStamp, normalizedSlug))

	templateContent, err := readProbeTemplate(projectDir)
	if err != nil {
		return "", err
	}

	content := renderProbeTemplate(templateContent, normalizedModelName, dateStamp)
	if err := writeFileExclusive(probePath, content); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("probe already exists: %s", probePath)
		}
		return "", fmt.Errorf("failed to create probe file: %w", err)
	}

	return probePath, nil
}

func resolveProbeModelPath(projectDir, modelName string) (string, string, error) {
	normalized := strings.TrimSpace(modelName)
	normalized = strings.TrimSuffix(normalized, ".md")

	if normalized == "" {
		return "", "", fmt.Errorf("model name is required")
	}

	if strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") {
		return "", "", fmt.Errorf("model name must be a basename under .kb/models: %q", modelName)
	}

	modelPath := filepath.Join(projectDir, ".kb", "models", normalized+".md")
	info, err := os.Stat(modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			available := listAvailableModels(projectDir)
			if len(available) > 0 {
				return "", "", fmt.Errorf("model not found: %s (available: %s)", normalized, strings.Join(available, ", "))
			}
			return "", "", fmt.Errorf("model not found: %s", normalized)
		}
		return "", "", fmt.Errorf("failed to validate model %s: %w", normalized, err)
	}

	if info.IsDir() {
		return "", "", fmt.Errorf("model path is a directory, expected .md file: %s", modelPath)
	}

	return modelPath, normalized, nil
}

func normalizeProbeSlug(slug string) (string, error) {
	normalized := generateSlug(strings.TrimSpace(slug))
	if normalized == "" {
		return "", fmt.Errorf("slug must contain at least one alphanumeric character")
	}

	return normalized, nil
}

func readProbeTemplate(projectDir string) (string, error) {
	templatePath := filepath.Join(projectDir, ".orch", "templates", "PROBE.md")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("probe template not found: %s", templatePath)
		}
		return "", fmt.Errorf("failed to read probe template: %w", err)
	}

	return string(content), nil
}

func renderProbeTemplate(templateContent, modelName, date string) string {
	rendered := strings.ReplaceAll(templateContent, "{{model-path}}", modelName)
	rendered = strings.ReplaceAll(rendered, "{{date}}", date)
	return rendered
}

func writeFileExclusive(path, content string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func listAvailableModels(projectDir string) []string {
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	models := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}

		base := strings.TrimSuffix(name, ".md")
		if base == "README" || strings.HasPrefix(base, "_") {
			continue
		}

		models = append(models, base)
	}

	sort.Strings(models)
	return models
}
