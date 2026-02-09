package verify

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// VerificationSpecFileName is the canonical proof-carrying verification spec filename.
const VerificationSpecFileName = "VERIFICATION_SPEC.yaml"

// VerificationMethod identifies how a verification entry is executed.
type VerificationMethod string

const (
	VerificationMethodCLISmoke    VerificationMethod = "cli_smoke"
	VerificationMethodIntegration VerificationMethod = "integration"
	VerificationMethodBrowser     VerificationMethod = "browser"
	VerificationMethodManual      VerificationMethod = "manual"
	VerificationMethodStatic      VerificationMethod = "static"
)

// VerificationTier identifies the verification strictness tier.
type VerificationTier string

const (
	VerificationTierLight        VerificationTier = "light"
	VerificationTierFull         VerificationTier = "full"
	VerificationTierOrchestrator VerificationTier = "orchestrator"
)

// ProofSpec is the parsed, validated verification contract.
type ProofSpec struct {
	Version      int                 `json:"version"`
	Scope        ProofSpecScope      `json:"scope"`
	Verification []ProofVerification `json:"verification"`
}

// ProofSpecScope scopes a verification contract to a specific spawn.
type ProofSpecScope struct {
	BeadsID   string `json:"beads_id" yaml:"beads_id"`
	Workspace string `json:"workspace" yaml:"workspace"`
	Skill     string `json:"skill" yaml:"skill"`
}

// ProofVerification is one verification step from VERIFICATION_SPEC.yaml.
type ProofVerification struct {
	ID             string             `json:"id" yaml:"id"`
	Method         VerificationMethod `json:"method" yaml:"method"`
	Tier           VerificationTier   `json:"tier" yaml:"tier"`
	Command        string             `json:"command,omitempty" yaml:"command,omitempty"`
	CWD            string             `json:"cwd,omitempty" yaml:"cwd,omitempty"`
	ManualSteps    []string           `json:"manual_steps,omitempty" yaml:"manual_steps,omitempty"`
	TimeoutSeconds int                `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	Expect         ProofExpectations  `json:"expect" yaml:"expect"`
}

// ProofExpectations is the normalized expectation block for a verification step.
type ProofExpectations struct {
	ExitCode              int      `json:"exit_code" yaml:"exit_code"`
	StdoutContains        []string `json:"stdout_contains,omitempty" yaml:"stdout_contains,omitempty"`
	HumanApprovalRequired bool     `json:"human_approval_required,omitempty" yaml:"human_approval_required,omitempty"`
}

type proofSpecYAML struct {
	Version      *int                    `yaml:"version"`
	Scope        *proofSpecScopeYAML     `yaml:"scope"`
	Verification []proofVerificationYAML `yaml:"verification"`
}

type proofSpecScopeYAML struct {
	BeadsID   string `yaml:"beads_id"`
	Workspace string `yaml:"workspace"`
	Skill     string `yaml:"skill"`
}

type proofVerificationYAML struct {
	ID             string           `yaml:"id"`
	Method         string           `yaml:"method"`
	Tier           string           `yaml:"tier"`
	Command        string           `yaml:"command"`
	CWD            string           `yaml:"cwd"`
	ManualSteps    []string         `yaml:"manual_steps"`
	TimeoutSeconds *int             `yaml:"timeout_seconds"`
	Expect         *proofExpectYAML `yaml:"expect"`
}

type proofExpectYAML struct {
	ExitCode              *int     `yaml:"exit_code"`
	StdoutContains        []string `yaml:"stdout_contains"`
	HumanApprovalRequired *bool    `yaml:"human_approval_required"`
}

// LoadProofSpec parses VERIFICATION_SPEC.yaml from a workspace root.
func LoadProofSpec(workspacePath string) (*ProofSpec, error) {
	path := filepath.Join(workspacePath, VerificationSpecFileName)
	return ParseProofSpecFile(path)
}

// ParseProofSpecFile parses and validates a VERIFICATION_SPEC.yaml file.
func ParseProofSpecFile(path string) (*ProofSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read verification spec: %w", err)
	}

	spec, err := ParseProofSpecYAML(data)
	if err != nil {
		return nil, fmt.Errorf("invalid verification spec %s: %w", path, err)
	}

	return spec, nil
}

// ParseProofSpecYAML parses and validates raw VERIFICATION_SPEC.yaml bytes.
func ParseProofSpecYAML(data []byte) (*ProofSpec, error) {
	var raw proofSpecYAML
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	var extra any
	err := dec.Decode(&extra)
	if err == nil {
		return nil, fmt.Errorf("yaml must contain a single document")
	}
	if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return validateProofSpec(raw)
}

func validateProofSpec(raw proofSpecYAML) (*ProofSpec, error) {
	if raw.Version == nil {
		return nil, fmt.Errorf("version is required")
	}
	if *raw.Version != 1 {
		return nil, fmt.Errorf("version must be 1, got %d", *raw.Version)
	}

	if raw.Scope == nil {
		return nil, fmt.Errorf("scope is required")
	}

	scope := ProofSpecScope{
		BeadsID:   strings.TrimSpace(raw.Scope.BeadsID),
		Workspace: strings.TrimSpace(raw.Scope.Workspace),
		Skill:     strings.TrimSpace(raw.Scope.Skill),
	}
	if scope.BeadsID == "" {
		return nil, fmt.Errorf("scope.beads_id is required")
	}
	if scope.Workspace == "" {
		return nil, fmt.Errorf("scope.workspace is required")
	}
	if scope.Skill == "" {
		return nil, fmt.Errorf("scope.skill is required")
	}

	if len(raw.Verification) == 0 {
		return nil, fmt.Errorf("verification must contain at least one entry")
	}

	out := &ProofSpec{
		Version: *raw.Version,
		Scope:   scope,
	}

	seenIDs := make(map[string]struct{}, len(raw.Verification))
	for i, entry := range raw.Verification {
		parsed, err := validateProofVerification(entry, i)
		if err != nil {
			return nil, err
		}

		if _, exists := seenIDs[parsed.ID]; exists {
			return nil, fmt.Errorf("verification[%d].id %q is duplicated", i, parsed.ID)
		}
		seenIDs[parsed.ID] = struct{}{}

		out.Verification = append(out.Verification, parsed)
	}

	return out, nil
}

func validateProofVerification(entry proofVerificationYAML, index int) (ProofVerification, error) {
	id := strings.TrimSpace(entry.ID)
	if id == "" {
		return ProofVerification{}, fmt.Errorf("verification[%d].id is required", index)
	}

	method, err := parseVerificationMethod(entry.Method)
	if err != nil {
		return ProofVerification{}, fmt.Errorf("verification[%d].method: %w", index, err)
	}

	tier, err := parseVerificationTier(entry.Tier)
	if err != nil {
		return ProofVerification{}, fmt.Errorf("verification[%d].tier: %w", index, err)
	}

	parsed := ProofVerification{
		ID:     id,
		Method: method,
		Tier:   tier,
		Expect: ProofExpectations{},
	}

	if entry.TimeoutSeconds != nil {
		if *entry.TimeoutSeconds <= 0 {
			return ProofVerification{}, fmt.Errorf("verification[%d].timeout_seconds must be > 0 when provided", index)
		}
		parsed.TimeoutSeconds = *entry.TimeoutSeconds
	}

	command := strings.TrimSpace(entry.Command)
	cwd := strings.TrimSpace(entry.CWD)
	steps, err := normalizeManualSteps(entry.ManualSteps, index)
	if err != nil {
		return ProofVerification{}, err
	}

	expect, err := parseProofExpect(entry.Expect, method, index)
	if err != nil {
		return ProofVerification{}, err
	}
	parsed.Expect = expect

	if method == VerificationMethodManual {
		if command != "" {
			return ProofVerification{}, fmt.Errorf("verification[%d].command is not allowed for manual method", index)
		}
		if len(steps) == 0 {
			return ProofVerification{}, fmt.Errorf("verification[%d].manual_steps is required for manual method", index)
		}
		if !expect.HumanApprovalRequired {
			return ProofVerification{}, fmt.Errorf("verification[%d].expect.human_approval_required must be true for manual method", index)
		}

		parsed.ManualSteps = steps
		parsed.CWD = cwd
		return parsed, nil
	}

	if command == "" {
		return ProofVerification{}, fmt.Errorf("verification[%d].command is required for method %q", index, method)
	}
	if len(steps) > 0 {
		return ProofVerification{}, fmt.Errorf("verification[%d].manual_steps is only allowed for manual method", index)
	}

	parsed.Command = command
	parsed.CWD = cwd
	return parsed, nil
}

func normalizeManualSteps(steps []string, index int) ([]string, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(steps))
	for i, step := range steps {
		clean := strings.TrimSpace(step)
		if clean == "" {
			return nil, fmt.Errorf("verification[%d].manual_steps[%d] cannot be empty", index, i)
		}
		normalized = append(normalized, clean)
	}

	return normalized, nil
}

func parseProofExpect(expect *proofExpectYAML, method VerificationMethod, index int) (ProofExpectations, error) {
	out := ProofExpectations{}

	if expect == nil {
		if method != VerificationMethodManual {
			out.ExitCode = 0
		}
		return out, nil
	}

	for i, token := range expect.StdoutContains {
		clean := strings.TrimSpace(token)
		if clean == "" {
			return ProofExpectations{}, fmt.Errorf("verification[%d].expect.stdout_contains[%d] cannot be empty", index, i)
		}
		out.StdoutContains = append(out.StdoutContains, clean)
	}

	if expect.HumanApprovalRequired != nil {
		out.HumanApprovalRequired = *expect.HumanApprovalRequired
	}

	if method == VerificationMethodManual {
		if expect.ExitCode != nil {
			return ProofExpectations{}, fmt.Errorf("verification[%d].expect.exit_code is not allowed for manual method", index)
		}
		if len(out.StdoutContains) > 0 {
			return ProofExpectations{}, fmt.Errorf("verification[%d].expect.stdout_contains is not allowed for manual method", index)
		}
		return out, nil
	}

	out.ExitCode = 0
	if expect.ExitCode != nil {
		out.ExitCode = *expect.ExitCode
	}

	return out, nil
}

func parseVerificationMethod(raw string) (VerificationMethod, error) {
	method := VerificationMethod(strings.TrimSpace(raw))
	switch method {
	case VerificationMethodCLISmoke,
		VerificationMethodIntegration,
		VerificationMethodBrowser,
		VerificationMethodManual,
		VerificationMethodStatic:
		return method, nil
	default:
		return "", fmt.Errorf("must be one of cli_smoke|integration|browser|manual|static, got %q", raw)
	}
}

func parseVerificationTier(raw string) (VerificationTier, error) {
	tier := VerificationTier(strings.TrimSpace(raw))
	switch tier {
	case VerificationTierLight, VerificationTierFull, VerificationTierOrchestrator:
		return tier, nil
	default:
		return "", fmt.Errorf("must be one of light|full|orchestrator, got %q", raw)
	}
}

// ValidateProofSpecCommandSyntax checks bash syntax for executable proof-spec commands.
func ValidateProofSpecCommandSyntax(spec *ProofSpec) error {
	if spec == nil {
		return fmt.Errorf("spec is required")
	}

	errList := make([]string, 0)
	for i, entry := range spec.Verification {
		if entry.Method == VerificationMethodManual {
			continue
		}

		command := strings.TrimSpace(entry.Command)
		if command == "" {
			continue
		}

		if err := validateBashSyntax(command); err != nil {
			errList = append(errList, fmt.Sprintf("verification[%d].command (%s): %v", i, entry.ID, err))
		}
	}

	if len(errList) == 0 {
		return nil
	}

	return errors.New(strings.Join(errList, "; "))
}

func validateBashSyntax(command string) error {
	cmd := exec.Command("bash", "-n", "-c", command)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		text = err.Error()
	}

	return fmt.Errorf("invalid bash syntax: %s", text)
}
