package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/model"
)

var (
	switchSpawnAccount = account.SwitchAccount
	loadSpawnAccounts  = account.LoadConfig
	spawnUserHomeDir   = os.UserHomeDir
)

func maybeSwitchSpawnAccount(accountName string, resolvedModel model.ModelSpec) error {
	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		return nil
	}

	if !resolvedModel.IsAnthropic() {
		fmt.Fprintf(os.Stderr, "Warning: --account ignored for non-Anthropic model (%s)\n", resolvedModel.Format())
		return nil
	}

	email, err := switchSpawnAccount(accountName)
	if err != nil {
		return fmt.Errorf("failed to switch account %q: %w", accountName, err)
	}

	if strings.TrimSpace(email) != "" {
		fmt.Fprintf(os.Stderr, "🔐 Account override: %s (%s)\n", accountName, email)
	} else {
		fmt.Fprintf(os.Stderr, "🔐 Account override: %s\n", accountName)
	}

	return nil
}

func resolveSpawnClaudeConfigDir(explicitAccount string, usageResult *UsageCheckResult) string {
	if accountName := strings.TrimSpace(explicitAccount); accountName != "" {
		return claudeConfigDirForAccount(accountName)
	}

	if usageResult == nil || !usageResult.Switched {
		return ""
	}

	autoSwitchedAccount := strings.TrimSpace(usageResult.SwitchedToAccount)
	if autoSwitchedAccount == "" {
		return ""
	}

	accountsCfg, err := loadSpawnAccounts()
	if err == nil {
		primaryAccount := strings.TrimSpace(accountsCfg.Default)
		if primaryAccount != "" && autoSwitchedAccount == primaryAccount {
			return ""
		}
	}

	return claudeConfigDirForAccount(autoSwitchedAccount)
}

func claudeConfigDirForAccount(accountName string) string {
	homeDir, err := spawnUserHomeDir()
	if err != nil {
		return ""
	}

	safeAccount := sanitizeAccountForConfigDir(accountName)
	if safeAccount == "" {
		return ""
	}

	return filepath.Join(homeDir, ".claude-"+safeAccount)
}

func sanitizeAccountForConfigDir(accountName string) string {
	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(accountName))

	for _, r := range accountName {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' ||
			r == '_' ||
			r == '.' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}

	return strings.Trim(builder.String(), "-.")
}
