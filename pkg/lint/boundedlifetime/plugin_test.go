package boundedlifetime

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPluginRegistration(t *testing.T) {
	newPlugin, err := register.GetPlugin(linterName)
	if err != nil {
		t.Fatalf("register.GetPlugin(%q): %v", linterName, err)
	}

	pluginInstance, err := newPlugin(nil)
	if err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	analyzers, err := pluginInstance.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	if len(analyzers) != 1 {
		t.Fatalf("expected exactly one analyzer, got %d", len(analyzers))
	}

	analysistest.Run(t, testdataDir(t), analyzers[0], "boundedlifetimetest")
}

func testdataDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Join(filepath.Dir(filename), "testdata")
}
