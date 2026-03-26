package daemon

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchedulerRegistrationsHaveRunPeriodicMethods ensures every Task* constant
// registered in NewSchedulerFromConfig has a corresponding RunPeriodic* method
// on *Daemon. This prevents the "consumer-last" pattern where scheduler tasks
// are registered but never consumed.
func TestSchedulerRegistrationsHaveRunPeriodicMethods(t *testing.T) {
	fset := token.NewFileSet()

	// Parse scheduler.go to extract Task* constant names
	schedulerAST, err := parser.ParseFile(fset, "scheduler.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse scheduler.go: %v", err)
	}

	// Collect Task* constants
	taskConstants := map[string]string{} // constant name -> value
	ast.Inspect(schedulerAST, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			return true
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Task") {
					continue
				}
				if i < len(vs.Values) {
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
						taskConstants[name.Name] = strings.Trim(lit.Value, `"`)
					}
				}
			}
		}
		return true
	})

	if len(taskConstants) == 0 {
		t.Fatal("found no Task* constants in scheduler.go — parsing may be broken")
	}

	// Parse all .go files in pkg/daemon/ to find RunPeriodic* methods on *Daemon
	runPeriodicMethods := map[string]bool{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileAST, err := parser.ParseFile(fset, entry.Name(), nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", entry.Name(), err)
		}
		for _, decl := range fileAST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !strings.HasPrefix(fn.Name.Name, "RunPeriodic") {
				continue
			}
			// Verify receiver is *Daemon
			for _, field := range fn.Recv.List {
				if star, ok := field.Type.(*ast.StarExpr); ok {
					if ident, ok := star.X.(*ast.Ident); ok && ident.Name == "Daemon" {
						runPeriodicMethods[fn.Name.Name] = true
					}
				}
			}
		}
	}

	// Map Task constant names to expected RunPeriodic method names.
	// TaskCleanup -> RunPeriodicCleanup, TaskOrphanDetection -> RunPeriodicOrphanDetection, etc.
	for constName, _ := range taskConstants {
		suffix := strings.TrimPrefix(constName, "Task")
		expectedMethod := "RunPeriodic" + suffix
		if !runPeriodicMethods[expectedMethod] {
			t.Errorf("scheduler constant %s is registered but no %s method exists on *Daemon (consumer-last pattern)", constName, expectedMethod)
		}
	}
}

// TestRunPeriodicMethodsAreCalledInLoop ensures every RunPeriodic* method on
// *Daemon is actually called in the daemon's periodic task loop
// (cmd/orch/daemon_periodic.go). A RunPeriodic method that exists but is never
// called is dead code — the scheduler registration and implementation exist
// but the consumer loop never invokes them.
func TestRunPeriodicMethodsAreCalledInLoop(t *testing.T) {
	fset := token.NewFileSet()

	// Find RunPeriodic* methods on *Daemon
	runPeriodicMethods := []string{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileAST, err := parser.ParseFile(fset, entry.Name(), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range fileAST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !strings.HasPrefix(fn.Name.Name, "RunPeriodic") {
				continue
			}
			for _, field := range fn.Recv.List {
				if star, ok := field.Type.(*ast.StarExpr); ok {
					if ident, ok := star.X.(*ast.Ident); ok && ident.Name == "Daemon" {
						runPeriodicMethods = append(runPeriodicMethods, fn.Name.Name)
					}
				}
			}
		}
	}

	if len(runPeriodicMethods) == 0 {
		t.Fatal("found no RunPeriodic* methods on *Daemon — parsing may be broken")
	}

	// Read cmd/orch/daemon_periodic.go to check which methods are called
	periodicLoopPath := filepath.Join("..", "..", "cmd", "orch", "daemon_periodic.go")
	content, err := os.ReadFile(periodicLoopPath)
	if err != nil {
		t.Fatalf("failed to read daemon_periodic.go: %v", err)
	}
	loopSource := string(content)

	for _, method := range runPeriodicMethods {
		// Check for d.RunPeriodic* call pattern
		if !strings.Contains(loopSource, "."+method+"()") {
			t.Errorf("RunPeriodic method %s exists on *Daemon but is never called in daemon_periodic.go (dead consumer)", method)
		}
	}
}

// TestDaemonInterfaceFieldsAreWired ensures every interface-typed field on the
// Daemon struct is assigned either in NewWithConfig (pkg/daemon/daemon.go) or
// in daemonSetup (cmd/orch/daemon_loop.go). An interface field that is declared
// but never assigned will be nil at runtime, causing panics when periodic tasks
// or the spawn pipeline try to use it.
func TestDaemonInterfaceFieldsAreWired(t *testing.T) {
	fset := token.NewFileSet()

	// Parse daemon.go to find the Daemon struct and its fields
	daemonAST, err := parser.ParseFile(fset, "daemon.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse daemon.go: %v", err)
	}

	// Collect all interface types defined in the package
	interfaceTypes := map[string]bool{}
	pkgEntries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range pkgEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileAST, err := parser.ParseFile(fset, entry.Name(), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range fileAST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
					interfaceTypes[ts.Name.Name] = true
				}
			}
		}
	}

	// Find interface-typed fields on Daemon struct
	var interfaceFields []string
	ast.Inspect(daemonAST, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Daemon" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range st.Fields.List {
			typeName := fieldTypeName(field.Type)
			if typeName == "" {
				continue
			}
			if interfaceTypes[typeName] {
				for _, name := range field.Names {
					interfaceFields = append(interfaceFields, name.Name)
				}
			}
		}
		return false
	})

	if len(interfaceFields) == 0 {
		t.Fatal("found no interface-typed fields on Daemon struct — parsing may be broken")
	}

	// Fields that are intentionally nil-safe (checked before use or optional).
	// Each entry documents WHY it's allowed to be nil.
	optionalFields := map[string]string{
		"HotspotChecker":       "optional: extraction system disabled, nil means skip hotspot checks",
		"PriorArchitectFinder": "optional: nil means no prior-architect dedup (escalate every time)",
		"Rejector":             "optional: nil means no automatic rejection processing",
		"AuditLabeler":         "optional: nil means no audit label management",
		"ComprehensionQuerier": "optional: nil means comprehension gate fails open (by design)",
		"AutoCompleter":        "optional: nil means use label-based completion (legacy path)",
		"CapacityPoll":         "optional: nil uses default implementation via resolveCapacityPollService()",
		"AuditSelect":          "optional: nil uses default implementation via resolveAuditSelectService()",
		"EmptyExecutionClassifier": "optional: nil means skip empty-execution classification in orphan detector (falls back to normal reset)",
		"uncachedAgents":           "internal: set dynamically by BeginCycle(), not a setup field",
	}

	// Read NewWithConfig source and daemonSetup source to check for assignments
	daemonGoContent, err := os.ReadFile("daemon.go")
	if err != nil {
		t.Fatalf("failed to read daemon.go: %v", err)
	}

	daemonLoopPath := filepath.Join("..", "..", "cmd", "orch", "daemon_loop.go")
	daemonLoopContent, err := os.ReadFile(daemonLoopPath)
	if err != nil {
		t.Fatalf("failed to read daemon_loop.go: %v", err)
	}

	combinedSource := string(daemonGoContent) + "\n" + string(daemonLoopContent)

	for _, fieldName := range interfaceFields {
		if _, isOptional := optionalFields[fieldName]; isOptional {
			continue
		}

		// Check for assignment pattern: fieldName: or .fieldName =
		assignedInConstructor := strings.Contains(combinedSource, fieldName+":")
		assignedInSetup := strings.Contains(combinedSource, "."+fieldName+" =") || strings.Contains(combinedSource, "."+fieldName+"=")

		if !assignedInConstructor && !assignedInSetup {
			t.Errorf("Daemon interface field %s is declared but never assigned in NewWithConfig or daemonSetup (consumer-last pattern)", fieldName)
		}
	}
}

// fieldTypeName extracts the type name from a field type expression.
// Returns empty string for non-simple types (maps, slices, pointers to non-idents, etc.)
func fieldTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		// Skip pointer types — they're concrete types, not interfaces
		return ""
	default:
		return ""
	}
}
