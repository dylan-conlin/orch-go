// Package main provides the thread command for living threads management.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/thread"
	"github.com/spf13/cobra"
)

var threadWorkdir string

func threadsDir() (string, error) {
	projectDir := ""
	if threadWorkdir != "" {
		abs, err := filepath.Abs(threadWorkdir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("workdir does not exist: %s", abs)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("workdir is not a directory: %s", abs)
		}
		projectDir = abs
	} else {
		projectDir, _ = os.Getwd()
	}
	return filepath.Join(projectDir, ".kb", "threads"), nil
}

var threadCmd = &cobra.Command{
	Use:   "thread",
	Short: "Living threads — mid-session comprehension capture",
	Long: `Manage living threads in .kb/threads/.

Threads capture forming insight mid-session and accumulate entries across sessions.
They fill the gap between ephemeral conversation and formalized knowledge.

Examples:
  orch thread new "How enforcement and comprehension relate"
  orch thread append enforcement-comprehension "New insight..."
  orch thread list
  orch thread show enforcement-comprehension
  orch thread resolve enforcement-comprehension --to ".kb/models/enforcement.md"`,
}

var threadNewCmd = &cobra.Command{
	Use:   `new "title" ["initial entry"]`,
	Short: "Create a new thread",
	Long: `Create a new living thread with the given title.

Optionally provide an initial entry as the second argument.
If no entry is provided, creates the thread with an empty first entry.

Examples:
  orch thread new "How enforcement and comprehension relate"
  orch thread new "Daemon capacity" "First thought about this..."`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		entry := ""
		if len(args) > 1 {
			entry = args[1]
		}

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		result, err := thread.CreateOrAppend(dir, title, entry)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(mustGetwd(), result.FilePath)
		if result.Created {
			fmt.Printf("Thread created: %s\n", relPath)
		} else {
			fmt.Printf("Thread already exists, appended: %s (%d entries)\n", relPath, result.EntryCount)
		}
		return nil
	},
}

var threadAppendCmd = &cobra.Command{
	Use:   `append <slug> "entry text"`,
	Short: "Append an entry to an existing thread",
	Long: `Append a new entry to an existing thread's today section.

If today's date section already exists, appends to it.
Otherwise creates a new dated section.

Examples:
  orch thread append enforcement-comprehension "The distinction is clearer now..."
  orch thread append daemon-capacity "After seeing the metrics..."`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		entry := args[1]

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		result, err := thread.Append(dir, slug, entry)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(mustGetwd(), result.FilePath)
		fmt.Printf("Thread updated: %s (%d entries)\n", relPath, result.EntryCount)
		return nil
	},
}

var threadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all threads",
	Long: `List all threads with status, last update date, and latest entry preview.

Threads are sorted by most recently updated first.
Stale threads (open but not updated in 7+ days) are flagged.

Examples:
  orch thread list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := threadsDir()
		if err != nil {
			return err
		}

		threads, err := thread.List(dir)
		if err != nil {
			return err
		}

		if len(threads) == 0 {
			fmt.Println("No threads found in .kb/threads/")
			fmt.Println("Create one: orch thread new \"Thread title\"")
			return nil
		}

		today := time.Now()
		for _, t := range threads {
			statusIcon := threadStatusIcon(t.Status)
			staleFlag := ""
			if t.Status == "open" {
				if updated, err := time.Parse("2006-01-02", t.Updated); err == nil {
					age := int(today.Sub(updated).Hours() / 24)
					if age > 7 {
						staleFlag = fmt.Sprintf(" [stale: %dd]", age)
					}
				}
			}

			preview := t.LatestEntry
			if preview == "" {
				preview = "(empty)"
			}

			fmt.Printf("%s %s (%s, updated %s)%s\n", statusIcon, t.Name, t.Status, t.Updated, staleFlag)
			fmt.Printf("    %s\n", t.Title)
			fmt.Printf("    > %s\n", preview)
		}

		return nil
	},
}

var threadShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Display thread content",
	Long: `Display the full content of a thread.

Examples:
  orch thread show enforcement-comprehension`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		t, err := thread.Show(dir, slug)
		if err != nil {
			return err
		}

		fmt.Print(t.Content)
		return nil
	},
}

var (
	threadResolveTo string
)

var threadResolveCmd = &cobra.Command{
	Use:   "resolve <slug>",
	Short: "Mark a thread as resolved",
	Long: `Mark a thread as resolved, optionally linking to the target artifact.

Examples:
  orch thread resolve enforcement-comprehension --to ".kb/models/enforcement.md"
  orch thread resolve daemon-capacity`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		if err := thread.Resolve(dir, slug, threadResolveTo); err != nil {
			return err
		}

		if threadResolveTo != "" {
			fmt.Printf("Thread resolved: %s -> %s\n", slug, threadResolveTo)
		} else {
			fmt.Printf("Thread resolved: %s\n", slug)
		}
		return nil
	},
}

func init() {
	threadCmd.AddCommand(threadNewCmd)
	threadCmd.AddCommand(threadAppendCmd)
	threadCmd.AddCommand(threadListCmd)
	threadCmd.AddCommand(threadShowCmd)
	threadCmd.AddCommand(threadResolveCmd)

	threadCmd.PersistentFlags().StringVar(&threadWorkdir, "workdir", "", "Target project directory (for cross-project thread operations)")
	threadResolveCmd.Flags().StringVar(&threadResolveTo, "to", "", "Target artifact path (e.g., .kb/models/enforcement.md)")
}

func threadStatusIcon(status string) string {
	switch status {
	case "open":
		return "[~]"
	case "resolved":
		return "[x]"
	default:
		return "[?]"
	}
}

func mustGetwd() string {
	dir, _ := os.Getwd()
	return dir
}
