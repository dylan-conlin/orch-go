package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfig_MissingFile(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{
					Type:    "command",
					Command: "/nonexistent/path/hook.py",
					Timeout: 10,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundError := false
	for _, issue := range issues {
		if issue.Severity == SeverityError && issue.Message != "" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected error for missing file")
	}
}

func TestValidateConfig_NotExecutable(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nprint('hi')"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{
					Type:    "command",
					Command: script,
					Timeout: 10,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundNotExec := false
	for _, issue := range issues {
		if issue.Severity == SeverityError && containsStr(issue.Message, "not executable") {
			foundNotExec = true
		}
	}
	if !foundNotExec {
		t.Error("expected 'not executable' error")
	}
}

func TestValidateConfig_InvalidMatcher(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "[invalid", Hooks: []HookConfig{{
					Type:    "command",
					Command: "some-hook",
					Timeout: 10,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundInvalid := false
	for _, issue := range issues {
		if issue.Severity == SeverityError && containsStr(issue.Message, "invalid matcher regex") {
			foundInvalid = true
		}
	}
	if !foundInvalid {
		t.Error("expected 'invalid matcher regex' error")
	}
}

func TestValidateConfig_NoTimeout(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{
					Type:    "command",
					Command: "some-hook",
					Timeout: 0,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundTimeoutWarn := false
	for _, issue := range issues {
		if issue.Severity == SeverityWarning && containsStr(issue.Message, "no timeout") {
			foundTimeoutWarn = true
		}
	}
	if !foundTimeoutWarn {
		t.Error("expected warning about missing timeout")
	}
}

func TestValidateConfig_SessionStartHighTimeout(t *testing.T) {
	s := &Settings{
		Hooks: map[string][]HookGroup{
			"SessionStart": {
				{Hooks: []HookConfig{{
					Type:    "command",
					Command: "slow-hook.py",
					Timeout: 30,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundSlowStartup := false
	for _, issue := range issues {
		if issue.Severity == SeverityWarning && containsStr(issue.Message, "slow startup") {
			foundSlowStartup = true
		}
	}
	if !foundSlowStartup {
		t.Error("expected warning about slow startup for high SessionStart timeout")
	}
}

func TestValidateConfig_ValidHook(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{
					Type:    "command",
					Command: script,
					Timeout: 10,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	errors := 0
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errors++
		}
	}
	if errors > 0 {
		t.Errorf("expected 0 errors for valid hook, got %d", errors)
	}
}

func TestValidateConfig_MissingShebang(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.py")
	if err := os.WriteFile(script, []byte("print('no shebang')"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{Matcher: "Bash", Hooks: []HookConfig{{
					Type:    "command",
					Command: script,
					Timeout: 10,
				}}},
			},
		},
	}

	issues := ValidateConfig(s)

	foundShebang := false
	for _, issue := range issues {
		if issue.Severity == SeverityWarning && containsStr(issue.Message, "shebang") {
			foundShebang = true
		}
	}
	if !foundShebang {
		t.Error("expected warning about missing shebang for .py file")
	}
}

func TestFormatIssues_Empty(t *testing.T) {
	result := FormatIssues(nil)
	if result != "No issues found" {
		t.Errorf("expected 'No issues found', got '%s'", result)
	}
}

func TestFormatIssues_WithIssues(t *testing.T) {
	issues := []ValidationIssue{
		{Event: "PreToolUse", Matcher: "Bash", Command: "hook.py", Severity: SeverityError, Message: "not found"},
		{Event: "SessionStart", Matcher: "", Command: "start.sh", Severity: SeverityWarning, Message: "high timeout"},
	}

	result := FormatIssues(issues)
	if !containsStr(result, "1 error(s), 1 warning(s)") {
		t.Errorf("expected summary, got: %s", result)
	}
}
