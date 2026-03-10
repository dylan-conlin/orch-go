package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// --- dry-run helper tests ---

func TestValueOrNone_Empty(t *testing.T) {
	if got := valueOrNone(""); got != "(none)" {
		t.Errorf("valueOrNone(\"\") = %q, want \"(none)\"", got)
	}
}

func TestValueOrNone_NonEmpty(t *testing.T) {
	if got := valueOrNone("hello"); got != "hello" {
		t.Errorf("valueOrNone(\"hello\") = %q, want \"hello\"", got)
	}
}

func TestDryRunFlagRegistered(t *testing.T) {
	flag := spawnCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("--dry-run flag not registered on spawn command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want \"false\"", flag.DefValue)
	}
}

func TestPrintSetting(t *testing.T) {
	// Just verify printSetting doesn't panic with various inputs
	tests := []struct {
		name    string
		label   string
		setting spawn.ResolvedSetting
	}{
		{
			name:  "simple setting",
			label: "Backend",
			setting: spawn.ResolvedSetting{
				Value:  "claude",
				Source: spawn.SourceProjectConfig,
			},
		},
		{
			name:  "setting with detail",
			label: "Model",
			setting: spawn.ResolvedSetting{
				Value:  "anthropic/claude-opus-4-5-20251101",
				Source: spawn.SourceHeuristic,
				Detail: "infra-escape-hatch",
			},
		},
		{
			name:  "empty value",
			label: "MCP",
			setting: spawn.ResolvedSetting{
				Value:  "",
				Source: spawn.SourceDefault,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// printSetting writes to stdout; just verify it doesn't panic
			printSetting(tt.label, tt.setting)
		})
	}
}
