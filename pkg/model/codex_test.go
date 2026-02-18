package model

import "testing"

func TestCodexAliasesExist(t *testing.T) {
	aliases := []string{
		"codex",
		"codex-mini",
		"codex-max",
		"codex-latest",
		"codex-5.1",
		"codex-5.2",
	}

	for _, alias := range aliases {
		alias := alias
		t.Run(alias, func(t *testing.T) {
			spec := Resolve(alias)
			if spec.Provider == "" || spec.ModelID == "" {
				t.Fatalf("alias %q resolved to empty spec: %+v", alias, spec)
			}
		})
	}
}
