package orch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestAllSpawnModesIncludeModelInEventData is a structural regression test.
// It scans spawn_modes.go to ensure every session.spawned event data map
// includes a "model" key. This prevents regressions like the one where
// runSpawnClaude omitted the model field (orch-go-eipy2).
func TestAllSpawnModesIncludeModelInEventData(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "spawn_modes.go", nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse spawn_modes.go: %v", err)
	}

	// Find all composite literals that are map[string]interface{} and contain
	// "session.spawned" nearby (i.e., the event data maps for spawned events).
	// Strategy: find all functions that contain "session.spawned" string literals,
	// then check that they also contain a "model" key in their event data maps.
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Check if this function contains a "session.spawned" string literal
		hasSpawnedEvent := false
		ast.Inspect(fn, func(inner ast.Node) bool {
			lit, ok := inner.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING && lit.Value == `"session.spawned"` {
				hasSpawnedEvent = true
			}
			return true
		})

		if !hasSpawnedEvent {
			return true
		}

		// This function emits a session.spawned event.
		// Check that some map literal in it contains a "model" key.
		hasModelKey := false
		ast.Inspect(fn, func(inner ast.Node) bool {
			comp, ok := inner.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				keyLit, ok := kv.Key.(*ast.BasicLit)
				if ok && keyLit.Kind == token.STRING && keyLit.Value == `"model"` {
					hasModelKey = true
				}
			}
			return true
		})

		if !hasModelKey {
			fnName := fn.Name.Name
			// Only flag runSpawn* functions (the actual spawn mode handlers)
			if strings.HasPrefix(fnName, "runSpawn") {
				t.Errorf("%s emits session.spawned but does not include \"model\" in event data", fnName)
			}
		}

		return true
	})
}
