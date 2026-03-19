package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// detectCallerContext determines who is invoking the command based on env vars.
// Returns one of: "human", "orchestrator", "worker".
func detectCallerContext() string {
	claudeCtx := os.Getenv("CLAUDE_CONTEXT")
	switch claudeCtx {
	case "orchestrator", "meta-orchestrator":
		return "orchestrator"
	case "worker":
		return "worker"
	}

	if os.Getenv("ORCH_SPAWNED") == "1" {
		return "worker"
	}

	return "human"
}

// emitCommandInvoked logs a command.invoked event for usage tracking.
// Errors are silently ignored — telemetry must never break the command.
func emitCommandInvoked(command string, flags ...string) {
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.LogCommandInvoked(events.CommandInvokedData{
		Command: command,
		Caller:  detectCallerContext(),
		Flags:   strings.Join(flags, " "),
	})
}

// flagsFromCmd extracts changed (non-default) flags from a cobra command.
func flagsFromCmd(cmd *cobra.Command) []string {
	var flags []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		flags = append(flags, fmt.Sprintf("--%s=%s", f.Name, f.Value.String()))
	})
	return flags
}
