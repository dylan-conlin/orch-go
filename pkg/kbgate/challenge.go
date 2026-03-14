package kbgate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Fixed severity codes from the adversarial gate design.
// These are the only valid codes for challenge artifacts.
var validSeverityCodes = map[string]bool{
	"ENDOGENOUS_EVIDENCE":    true, // Claim cites only model+probe descendants
	"VOCABULARY_INFLATION":   true, // Coined term collapses to existing concepts
	"EXTERNAL_NOVELTY_DELTA": true, // No predictive content beyond known concepts
	"PUBLICATION_LANGUAGE":   true, // Uses banned novelty language
}

// ValidSeverityCode returns true if the code is in the fixed severity code set.
func ValidSeverityCode(code string) bool {
	return validSeverityCodes[code]
}

// SeverityEntry is a single row in the challenge severity codes table.
type SeverityEntry struct {
	Code      string `yaml:"code"       json:"code"`
	Status    string `yaml:"status"     json:"status"`     // "pass" or "fail"
	AppliesTo string `yaml:"applies_to" json:"applies_to"` // claim ID or "publication"
	Note      string `yaml:"note"       json:"note"`
}

// Validate checks that a severity entry has valid fields.
func (e SeverityEntry) Validate() error {
	if !ValidSeverityCode(e.Code) {
		return fmt.Errorf("invalid severity code: %q (valid: %s)", e.Code, severityCodeList())
	}
	if e.Status != "pass" && e.Status != "fail" {
		return fmt.Errorf("invalid status %q: must be pass or fail", e.Status)
	}
	if e.AppliesTo == "" {
		return fmt.Errorf("applies_to is required")
	}
	if e.Note == "" {
		return fmt.Errorf("note is required")
	}
	return nil
}

func severityCodeList() string {
	codes := make([]string, 0, len(validSeverityCodes))
	for c := range validSeverityCodes {
		codes = append(codes, c)
	}
	return strings.Join(codes, ", ")
}

// ReviewerIndependence captures the three axes of reviewer independence
// required by the external challenge gate.
type ReviewerIndependence struct {
	ReviewerType          string `yaml:"reviewer_type"          json:"reviewer_type"`          // "human" or "model"
	ReviewerID            string `yaml:"reviewer_id"            json:"reviewer_id"`            // email or model ID
	OriginModel           string `yaml:"origin_model,omitempty" json:"origin_model,omitempty"` // model used for original artifacts
	ModelIndependence     string `yaml:"model_independence"     json:"model_independence"`     // how model independence is satisfied
	ContextIndependence   string `yaml:"context_independence"   json:"context_independence"`   // how context independence is satisfied
	AuthorityIndependence string `yaml:"authority_independence" json:"authority_independence"` // how authority independence is satisfied
}

// Validate checks reviewer independence metadata.
func (ri ReviewerIndependence) Validate() error {
	if ri.ReviewerType != "human" && ri.ReviewerType != "model" {
		return fmt.Errorf("reviewer_type must be 'human' or 'model', got %q", ri.ReviewerType)
	}
	if ri.ReviewerID == "" {
		return fmt.Errorf("reviewer_id is required")
	}
	if ri.ModelIndependence == "" {
		return fmt.Errorf("model_independence is required")
	}
	if ri.ContextIndependence == "" {
		return fmt.Errorf("context_independence is required")
	}
	if ri.AuthorityIndependence == "" {
		return fmt.Errorf("authority_independence is required")
	}

	// If reviewer is a model, check provider independence
	if ri.ReviewerType == "model" && ri.OriginModel != "" {
		reviewerProvider := extractProvider(ri.ReviewerID)
		originProvider := extractProvider(ri.OriginModel)
		if reviewerProvider != "" && originProvider != "" && reviewerProvider == originProvider {
			return fmt.Errorf("reviewer model %q is from same provider as origin model %q — requires different provider for model independence", ri.ReviewerID, ri.OriginModel)
		}
	}

	return nil
}

// extractProvider returns the provider prefix from a model ID (e.g., "anthropic" from "anthropic/claude-opus-4-5").
func extractProvider(modelID string) string {
	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// ChallengeArtifact is the structured frontmatter of a challenge file.
type ChallengeArtifact struct {
	TargetArtifact     string               `yaml:"target_artifact"     json:"target_artifact"`
	Reviewer           ReviewerIndependence  `yaml:"reviewer"           json:"reviewer"`
	SeverityCodes      []SeverityEntry       `yaml:"severity_codes"     json:"severity_codes"`
	PublicationVerdict string               `yaml:"publication_verdict" json:"publication_verdict"` // "pass" or "fail"
}

// Validate checks the challenge artifact for completeness and correctness.
func (ca ChallengeArtifact) Validate() error {
	if ca.TargetArtifact == "" {
		return fmt.Errorf("target_artifact is required")
	}
	if err := ca.Reviewer.Validate(); err != nil {
		return fmt.Errorf("reviewer: %w", err)
	}
	for i, entry := range ca.SeverityCodes {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("severity_codes[%d]: %w", i, err)
		}
	}
	if ca.PublicationVerdict != "pass" && ca.PublicationVerdict != "fail" {
		return fmt.Errorf("publication_verdict must be 'pass' or 'fail', got %q", ca.PublicationVerdict)
	}
	return nil
}

// ChallengePacket is the data sent to a reviewer for one pass of the challenge.
type ChallengePacket struct {
	PassType     string `json:"pass_type"` // "blind" or "framed"
	ArtifactPath string `json:"artifact_path"`
	Instructions string `json:"instructions"`
	Content      string `json:"content"`
}

// ParseChallengeArtifact reads and parses a challenge file's YAML frontmatter.
func ParseChallengeArtifact(path string) (ChallengeArtifact, error) {
	var ca ChallengeArtifact

	data, err := os.ReadFile(path)
	if err != nil {
		return ca, fmt.Errorf("read challenge: %w", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return ca, fmt.Errorf("challenge artifact must have YAML frontmatter")
	}

	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return ca, fmt.Errorf("no closing --- for frontmatter")
	}

	yamlBlock := rest[:idx]
	if err := yaml.Unmarshal([]byte(yamlBlock), &ca); err != nil {
		return ca, fmt.Errorf("parse frontmatter: %w", err)
	}

	return ca, nil
}

// CreateChallengeArtifact creates a new challenge file from the template.
// Returns the path of the created file.
func CreateChallengeArtifact(projectDir, slug, targetArtifact string) (string, error) {
	challengesDir := filepath.Join(projectDir, ".kb", "challenges")
	if err := os.MkdirAll(challengesDir, 0755); err != nil {
		return "", fmt.Errorf("create challenges dir: %w", err)
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", date, slug)
	path := filepath.Join(challengesDir, filename)

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("challenge already exists: %s", path)
	}

	tmpl := GenerateChallengeTemplate(slug, targetArtifact)
	if err := os.WriteFile(path, []byte(tmpl), 0644); err != nil {
		return "", fmt.Errorf("write challenge: %w", err)
	}

	return path, nil
}

// GenerateChallengeTemplate returns a challenge artifact template with all required sections.
func GenerateChallengeTemplate(slug, targetArtifact string) string {
	return fmt.Sprintf(`---
target_artifact: %s
reviewer:
  reviewer_type: ""       # "human" or "model"
  reviewer_id: ""         # email or model ID (e.g., google/gemini-2.5-pro)
  origin_model: ""        # model used for original artifacts (if reviewer is model)
  model_independence: ""  # how model independence is satisfied
  context_independence: "" # how context independence is satisfied
  authority_independence: "" # how authority independence is satisfied
severity_codes: []
publication_verdict: ""   # "pass" or "fail"
---

# Challenge: %s

## Target Artifact
%s

## Reviewer Independence
<!-- Document how the three independence axes are satisfied:
1. Model independence: different provider/model family, or human external to project
2. Context independence: reviewer received fixed challenge packet, not full thread
3. Authority independence: reviewer can block/downgrade but cannot bless -->

## Blind Canonicalization
<!-- Reviewer's blind-pass findings: What existing concepts explain this?
What is surprising? What would be overclaim? -->

## Prior-Art Mapping
<!-- Map coined/framework terms to nearest existing concepts -->

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
|      |                |                           |               |         |

## Evidence Loop Findings
<!-- Which claims have only endogenous evidence (model/probe self-references)?
Which have exogenous grounding (investigations, external sources)? -->

## Severity Codes

| code | status | applies_to | note |
|------|--------|------------|------|
| ENDOGENOUS_EVIDENCE | | | |
| VOCABULARY_INFLATION | | | |
| EXTERNAL_NOVELTY_DELTA | | | |
| PUBLICATION_LANGUAGE | | | |

## Publication Verdict
<!-- "pass" or "fail" — computed from severity codes above.
Any severity code with status=fail blocks publication. -->
`, targetArtifact, slug, targetArtifact)
}

// GenerateBlindPacket creates a challenge packet for the blind pass.
// The blind pass strips model-specific vocabulary and asks the reviewer
// to identify what existing concepts explain the observations.
func GenerateBlindPacket(artifactPath string) (ChallengePacket, error) {
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		return ChallengePacket{}, fmt.Errorf("read artifact: %w", err)
	}

	instructions := `You are reviewing observations from a software engineering project.
You have NOT seen the model or framework that produced these observations.

Answer these three questions:

1. What existing concepts from software engineering, organizational theory,
   or other established fields explain these observations?
2. What, if anything, is surprising or not explained by known concepts?
3. What claims would be overclaim if published as novel findings?

Respond with structured findings. Do not generate a better theory —
only identify what is familiar, what is surprising, and what is overclaimed.`

	return ChallengePacket{
		PassType:     "blind",
		ArtifactPath: artifactPath,
		Instructions: instructions,
		Content:      string(content),
	}, nil
}

// GenerateFramedPacket creates a challenge packet for the framed pass.
// The framed pass shows the actual model/publication and asks the reviewer
// to identify renamed concepts, endogenous evidence, and banned language.
func GenerateFramedPacket(artifactPath string) (ChallengePacket, error) {
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		return ChallengePacket{}, fmt.Errorf("read artifact: %w", err)
	}

	instructions := `You are reviewing a model or publication from a software engineering project.
You have already seen a blind summary of the observations (without framework language).

Answer these three questions:

1. Which claims are renamed familiar concepts? For each, name the existing
   concept and whether the new name adds predictive value.
2. Which claims depend on endogenous evidence (the model cites probes that
   cite the model)? Flag any circular evidence chains.
3. Which publication terms should be banned? Check against: "physics",
   "new framework", "general law", "substrate-independent", "proves",
   "validated theory".

Respond with structured severity codes. You can block or downgrade claims,
but you cannot approve them with freeform endorsement.`

	return ChallengePacket{
		PassType:     "framed",
		ArtifactPath: artifactPath,
		Instructions: instructions,
		Content:      string(content),
	}, nil
}
