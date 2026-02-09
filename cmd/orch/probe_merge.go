package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

var probeVerdictRe = regexp.MustCompile(`(?i)\*\*Verdict:\*\*\s*(confirms|extends|contradicts)\b`)

type probeMergeResult struct {
	merged    int
	confirms  int
	extends   int
	skipped   int
	changelog []string
	review    []string
	warnings  []string
}

func findProbeMergeCandidates(target *CompletionTarget) []spawn.ProjectProbe {
	return spawn.FindProjectProbes(resolveProbeMergeRoot(target))
}

func resolveProbeMergeRoot(target *CompletionTarget) string {
	if strings.TrimSpace(target.WorkspacePath) != "" {
		return target.WorkspacePath
	}
	return target.BeadsProjectDir
}

func mergeProbeIntoModel(probe spawn.ProjectProbe) error {
	return spawn.MergeProbeIntoModel(probe.ModelPath, probe)
}

func mergeProbesNonInteractive(probes []spawn.ProjectProbe) probeMergeResult {
	result := probeMergeResult{}
	for _, probe := range probes {
		if strings.TrimSpace(probe.Impact) == "" {
			result.skipped++
			continue
		}

		verdict := probeVerdict(probe.Impact)
		if verdict == "contradicts" || verdict == "" {
			result.review = append(result.review, probeLabel(probe))
			continue
		}

		if err := mergeProbeIntoModel(probe); err != nil {
			result.warnings = append(result.warnings, fmt.Sprintf("failed to merge %s: %v", probeLabel(probe), err))
			continue
		}

		result.merged++
		if verdict == "confirms" {
			result.confirms++
			continue
		}

		result.extends++
		result.changelog = append(result.changelog, fmt.Sprintf("%s extends model %s", probe.Probe.Name, probe.ModelName))
	}
	return result
}

func printProbeMergeNonInteractive(result probeMergeResult) {
	if result.merged > 0 {
		fmt.Printf("Auto-merged %d probe(s) in non-interactive mode.\n", result.merged)
	}

	for _, note := range result.changelog {
		fmt.Printf("  Changelog note: %s\n", note)
	}

	if len(result.review) > 0 {
		fmt.Println("  Review required (not auto-merged):")
		for _, label := range result.review {
			fmt.Printf("    - %s\n", label)
		}
	}

	if result.skipped > 0 {
		fmt.Printf("  Skipped %d probe(s) with no Model Impact section.\n", result.skipped)
	}

	for _, warning := range result.warnings {
		fmt.Fprintf(os.Stderr, "  Warning: %s\n", warning)
	}
}

func commitProbeMergeArtifacts(target *CompletionTarget, probes []spawn.ProjectProbe) (bool, error) {
	root := resolveProbeMergeRoot(target)
	files := probeCommitFiles(root, probes)
	if len(files) == 0 {
		return false, nil
	}

	name := strings.TrimSpace(target.AgentName)
	if name == "" {
		name = filepath.Base(strings.TrimSpace(target.WorkspacePath))
	}
	if name == "" || name == "." {
		name = "workspace"
	}

	return commitProbeFiles(root, name, files)
}

func probeCommitFiles(root string, probes []spawn.ProjectProbe) []string {
	files := []string{}
	seen := map[string]struct{}{}

	add := func(path string) {
		if path == "" {
			return
		}

		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return
		}

		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		files = append(files, rel)
	}

	for _, probe := range probes {
		add(probe.Probe.Path)
		add(probe.ModelPath)
	}

	return files
}

func commitProbeFiles(root, name string, files []string) (bool, error) {
	if root == "" {
		return false, fmt.Errorf("missing merge root")
	}
	if len(files) == 0 {
		return false, nil
	}

	addArgs := append([]string{"add", "--"}, files...)
	out, err := runGit(root, addArgs...)
	if err != nil {
		return false, fmt.Errorf("git add failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	diffArgs := append([]string{"diff", "--cached", "--name-only", "--"}, files...)
	out, err = runGit(root, diffArgs...)
	if err != nil {
		return false, fmt.Errorf("git diff --cached failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	msg := fmt.Sprintf("chore: merge probes from %s", name)
	commitArgs := append([]string{"commit", "-m", msg, "--"}, files...)
	out, err = runGit(root, commitArgs...)
	if err != nil {
		return false, fmt.Errorf("git commit failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	return true, nil
}

func runGit(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

func probeVerdict(impact string) string {
	m := probeVerdictRe.FindStringSubmatch(impact)
	if len(m) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(m[1]))
}

func probeLabel(probe spawn.ProjectProbe) string {
	return fmt.Sprintf("%s → %s", probe.Probe.Name, probe.ModelName)
}
