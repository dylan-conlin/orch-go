// Package kbgate implements adversarial gates for the knowledge pipeline.
// Gates are mechanical blockers that prevent publication of overclaimed theory.
package kbgate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Verdict represents a single gate check result.
type Verdict struct {
	Code      string `json:"code"`
	Status    string `json:"status"` // "pass" or "fail"
	AppliesTo string `json:"applies_to"`
	Note      string `json:"note"`
}

// GateResult is the aggregate result of all gate checks.
type GateResult struct {
	Pass     bool      `json:"pass"`
	Verdicts []Verdict `json:"verdicts"`
}

// Claim represents a structured claim in publication frontmatter.
type Claim struct {
	ClaimID      string   `yaml:"claim_id"`
	ClaimText    string   `yaml:"claim_text"`
	ClaimType    string   `yaml:"claim_type"`    // observation, mechanism, generalization, recommendation
	NoveltyLevel string   `yaml:"novelty_level"` // restatement, synthesis, novel
	EvidenceRefs []string `yaml:"evidence_refs"`
}

// PublicationFrontmatter is the YAML frontmatter parsed from a publication file.
type PublicationFrontmatter struct {
	ChallengeRefs []string `yaml:"challenge_refs"`
	ClaimRefs     []string `yaml:"claim_refs"`
	Claims        []Claim  `yaml:"claims"`
}

// bannedTerms are novelty-bearing phrases that require earned evidence class.
var bannedTerms = []string{
	"physics",
	"new framework",
	"general law",
	"substrate-independent",
	"proves",
	"validated theory",
}

// CheckPublish runs all Phase 1 gate checks on a publication file.
// Returns a GateResult with pass/fail and individual verdicts.
func CheckPublish(pubPath string) GateResult {
	result := GateResult{Pass: true}

	content, err := os.ReadFile(pubPath)
	if err != nil {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:   "FILE_NOT_FOUND",
			Status: "fail",
			Note:   fmt.Sprintf("cannot read publication: %v", err),
		})
		return result
	}

	fm, body, err := parseFrontmatter(string(content))
	if err != nil {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:   "INVALID_FRONTMATTER",
			Status: "fail",
			Note:   fmt.Sprintf("cannot parse frontmatter: %v", err),
		})
		return result
	}

	// Resolve project root from publication path for artifact lookups
	projectDir := resolveProjectDir(pubPath)

	// Gate 1: Publication contract — required fields
	checkContract(fm, &result)

	// Gate 2: Challenge artifact existence
	checkChallengeArtifacts(fm, projectDir, &result)

	// Gate 3: Lineage — endogenous evidence detection
	checkLineage(fm, &result)

	// Gate 4: Banned novelty language
	checkBannedLanguage(body, &result)

	return result
}

// checkContract verifies required publication contract fields.
func checkContract(fm PublicationFrontmatter, result *GateResult) {
	if len(fm.ChallengeRefs) == 0 {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:      "MISSING_CHALLENGE_REFS",
			Status:    "fail",
			AppliesTo: "publication",
			Note:      "publication must include challenge_refs in frontmatter",
		})
	}
	if len(fm.ClaimRefs) == 0 {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:      "MISSING_CLAIM_REFS",
			Status:    "fail",
			AppliesTo: "publication",
			Note:      "publication must include claim_refs in frontmatter",
		})
	}
}

// checkChallengeArtifacts verifies that referenced challenge files exist.
func checkChallengeArtifacts(fm PublicationFrontmatter, projectDir string, result *GateResult) {
	for _, ref := range fm.ChallengeRefs {
		fullPath := filepath.Join(projectDir, ref)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "CHALLENGE_ARTIFACT_MISSING",
				Status:    "fail",
				AppliesTo: ref,
				Note:      fmt.Sprintf("challenge artifact not found: %s", ref),
			})
		}
	}
}

// checkLineage detects self-referential evidence chains.
// Claims of type generalization or novel that reference only models/probes
// (endogenous evidence) fail the gate.
func checkLineage(fm PublicationFrontmatter, result *GateResult) {
	for _, claim := range fm.Claims {
		// Only check generalization/mechanism claims with novel/synthesis novelty
		if claim.ClaimType == "observation" || claim.ClaimType == "recommendation" {
			continue
		}
		if claim.NoveltyLevel == "restatement" {
			continue
		}

		if len(claim.EvidenceRefs) == 0 {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "ENDOGENOUS_EVIDENCE",
				Status:    "fail",
				AppliesTo: claim.ClaimID,
				Note:      "claim has no evidence_refs",
			})
			continue
		}

		hasExogenous := false
		for _, ref := range claim.EvidenceRefs {
			if isExogenousRef(ref) {
				hasExogenous = true
				break
			}
		}

		if !hasExogenous {
			result.Pass = false
			result.Verdicts = append(result.Verdicts, Verdict{
				Code:      "ENDOGENOUS_EVIDENCE",
				Status:    "fail",
				AppliesTo: claim.ClaimID,
				Note:      "claim references only models/probes (endogenous evidence); needs investigation, external source, or challenge artifact",
			})
		}
	}
}

// isExogenousRef returns true if a reference points to evidence outside
// the model/probe loop (investigations, external sources, challenges).
func isExogenousRef(ref string) bool {
	// Endogenous: .kb/models/*/model.md, .kb/models/*/probes/*
	if strings.Contains(ref, "/models/") && (strings.HasSuffix(ref, "/model.md") || strings.Contains(ref, "/probes/")) {
		return false
	}
	// Everything else is considered exogenous:
	// investigations, challenges, external URLs, raw data
	return true
}

// checkBannedLanguage scans publication body for banned novelty terms.
func checkBannedLanguage(body string, result *GateResult) {
	lower := strings.ToLower(body)
	var found []string
	for _, term := range bannedTerms {
		if strings.Contains(lower, term) {
			found = append(found, term)
		}
	}
	if len(found) > 0 {
		result.Pass = false
		result.Verdicts = append(result.Verdicts, Verdict{
			Code:      "BANNED_LANGUAGE",
			Status:    "fail",
			AppliesTo: "publication",
			Note:      fmt.Sprintf("banned novelty terms found: %s", strings.Join(found, ", ")),
		})
	}
}

// parseFrontmatter splits YAML frontmatter from markdown body.
func parseFrontmatter(content string) (PublicationFrontmatter, string, error) {
	var fm PublicationFrontmatter

	if !strings.HasPrefix(content, "---") {
		return fm, content, fmt.Errorf("no YAML frontmatter found (must start with ---)")
	}

	// Find closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, content, fmt.Errorf("no closing --- for frontmatter")
	}

	yamlBlock := rest[:idx]
	body := rest[idx+4:] // skip \n---

	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return fm, body, fmt.Errorf("parse YAML: %w", err)
	}

	return fm, body, nil
}

// resolveProjectDir walks up from pubPath to find a directory containing .kb/
func resolveProjectDir(pubPath string) string {
	absPath, err := filepath.Abs(pubPath)
	if err != nil {
		return filepath.Dir(pubPath)
	}

	dir := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".kb")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback: check if parent of pubPath has .kb
	return filepath.Dir(absPath)
}

// FormatResult produces a human-readable summary of gate results.
func FormatResult(result GateResult) string {
	var sb strings.Builder

	if result.Pass {
		sb.WriteString("✓ Publication gate: PASS\n")
	} else {
		sb.WriteString("✗ Publication gate: FAIL\n")
	}

	sb.WriteString("\nVerdicts:\n")
	for _, v := range result.Verdicts {
		icon := "✓"
		if v.Status == "fail" {
			icon = "✗"
		}
		line := fmt.Sprintf("  %s %-30s %s", icon, v.Code, v.Note)
		if v.AppliesTo != "" {
			line = fmt.Sprintf("  %s %-30s [%s] %s", icon, v.Code, v.AppliesTo, v.Note)
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

