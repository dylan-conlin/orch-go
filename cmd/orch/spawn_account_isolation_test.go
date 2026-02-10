package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/model"
)

func withSpawnAccountHelpersStubbed(t *testing.T) {
	t.Helper()

	oldSwitch := switchSpawnAccount
	oldLoad := loadSpawnAccounts
	oldHome := spawnUserHomeDir

	t.Cleanup(func() {
		switchSpawnAccount = oldSwitch
		loadSpawnAccounts = oldLoad
		spawnUserHomeDir = oldHome
	})
}

func TestMaybeSwitchSpawnAccountAnthropicModel(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	called := false
	switchSpawnAccount = func(name string) (string, error) {
		called = true
		if name != "work" {
			t.Fatalf("switch account name = %q, want %q", name, "work")
		}
		return "work@example.com", nil
	}

	err := maybeSwitchSpawnAccount("work", model.Resolve("opus"))
	if err != nil {
		t.Fatalf("maybeSwitchSpawnAccount() error = %v", err)
	}
	if !called {
		t.Fatal("expected switchSpawnAccount to be called for anthropic model")
	}
}

func TestMaybeSwitchSpawnAccountNonAnthropicModelIgnored(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	called := false
	switchSpawnAccount = func(name string) (string, error) {
		called = true
		return "", nil
	}

	err := maybeSwitchSpawnAccount("work", model.Resolve("google/gemini-2.5-pro"))
	if err != nil {
		t.Fatalf("maybeSwitchSpawnAccount() error = %v", err)
	}
	if called {
		t.Fatal("did not expect switchSpawnAccount to be called for non-anthropic model")
	}
}

func TestMaybeSwitchSpawnAccountReturnsSwitchError(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	switchSpawnAccount = func(name string) (string, error) {
		return "", errors.New("boom")
	}

	err := maybeSwitchSpawnAccount("work", model.Resolve("opus"))
	if err == nil {
		t.Fatal("expected error when account switch fails")
	}
}

func TestResolveSpawnClaudeConfigDirExplicitAccount(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	spawnUserHomeDir = func() (string, error) {
		return "/tmp/home", nil
	}

	got := resolveSpawnClaudeConfigDir("work", nil)
	want := filepath.Join("/tmp/home", ".claude-work")
	if got != want {
		t.Fatalf("resolveSpawnClaudeConfigDir() = %q, want %q", got, want)
	}
}

func TestResolveSpawnClaudeConfigDirAutoSwitchNonPrimary(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	loadSpawnAccounts = func() (*account.Config, error) {
		return &account.Config{Default: "personal"}, nil
	}
	spawnUserHomeDir = func() (string, error) {
		return "/tmp/home", nil
	}

	got := resolveSpawnClaudeConfigDir("", &UsageCheckResult{Switched: true, SwitchedToAccount: "work"})
	want := filepath.Join("/tmp/home", ".claude-work")
	if got != want {
		t.Fatalf("resolveSpawnClaudeConfigDir() = %q, want %q", got, want)
	}
}

func TestResolveSpawnClaudeConfigDirAutoSwitchPrimarySkipsIsolation(t *testing.T) {
	withSpawnAccountHelpersStubbed(t)

	loadSpawnAccounts = func() (*account.Config, error) {
		return &account.Config{Default: "personal"}, nil
	}

	got := resolveSpawnClaudeConfigDir("", &UsageCheckResult{Switched: true, SwitchedToAccount: "personal"})
	if got != "" {
		t.Fatalf("resolveSpawnClaudeConfigDir() = %q, want empty", got)
	}
}

func TestSanitizeAccountForConfigDir(t *testing.T) {
	got := sanitizeAccountForConfigDir(" work/team ")
	if got != "work-team" {
		t.Fatalf("sanitizeAccountForConfigDir() = %q, want %q", got, "work-team")
	}
}
