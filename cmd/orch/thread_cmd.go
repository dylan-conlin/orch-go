// Package main provides the thread command for living threads management.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/claims"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/thread"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var threadWorkdir string

func threadsDir() (string, error) {
	projectDir, _, err := identity.ResolveProjectDirectory(threadWorkdir)
	if err != nil {
		return "", err
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

var threadNewFrom string

var threadNewCmd = &cobra.Command{
	Use:   `new "title" ["initial entry"]`,
	Short: "Create a new thread",
	Long: `Create a new living thread with the given title.

Optionally provide an initial entry as the second argument.
If no entry is provided, creates the thread with an empty first entry.

Use --from to spawn a child thread from an existing parent thread.
The child gets a spawned_from reference and the parent's spawned list is updated.

Examples:
  orch thread new "How enforcement and comprehension relate"
  orch thread new "Daemon capacity" "First thought about this..."
  orch thread new --from coordination-primitives "Route vs sequence" "Exploring the distinction..."`,
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

		var result *thread.Result
		if threadNewFrom != "" {
			result, err = thread.CreateWithParent(dir, title, entry, threadNewFrom)
		} else {
			result, err = thread.CreateOrAppend(dir, title, entry)
		}
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(mustGetwd(), result.FilePath)
		if result.Created {
			if threadNewFrom != "" {
				fmt.Printf("Thread created: %s (from %s)\n", relPath, threadNewFrom)
			} else {
				fmt.Printf("Thread created: %s\n", relPath)
			}
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
			if thread.IsActive(t.Status) {
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
	threadResolveTo    string
	threadUpdateStatus string
	threadUpdateTo     string
)

var threadInputIsTerminal = func(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func promptResolvedTo(reader *bufio.Reader, writer io.Writer) (string, error) {
	for {
		if _, err := fmt.Fprint(writer, "Resolved to (model, decision, or brief): "); err != nil {
			return "", err
		}

		response, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		response = strings.TrimSpace(response)
		if response != "" {
			return response, nil
		}

		if _, err := fmt.Fprint(writer, "Resolved without artifact - confirm? [y/N]: "); err != nil {
			return "", err
		}

		confirm, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(confirm)) {
		case "y", "yes":
			return "", nil
		}
	}
}

func resolveToForStatus(cmd *cobra.Command, status, resolvedTo string) (string, error) {
	if status != thread.StatusResolved {
		return resolvedTo, nil
	}

	if strings.TrimSpace(resolvedTo) != "" {
		return strings.TrimSpace(resolvedTo), nil
	}

	stdin := cmd.InOrStdin()
	if !threadInputIsTerminal(stdin) {
		return "", fmt.Errorf("--status resolved requires interactive input or --to")
	}

	return promptResolvedTo(bufio.NewReader(stdin), cmd.ErrOrStderr())
}

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

		resolvedTo, err := resolveToForStatus(cmd, thread.StatusResolved, threadResolveTo)
		if err != nil {
			return err
		}

		if err := thread.Resolve(dir, slug, resolvedTo); err != nil {
			return err
		}

		if resolvedTo != "" {
			fmt.Printf("Thread resolved: %s -> %s\n", slug, resolvedTo)
		} else {
			fmt.Printf("Thread resolved: %s\n", slug)
		}
		return nil
	},
}

var threadUpdateCmd = &cobra.Command{
	Use:   "update <slug>",
	Short: "Update a thread's lifecycle status",
	Long: `Update a thread's lifecycle status.

Examples:
  orch thread update enforcement-comprehension --status active
  orch thread update enforcement-comprehension --status resolved
  orch thread update enforcement-comprehension --status resolved --to ".kb/models/enforcement.md"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		if strings.TrimSpace(threadUpdateStatus) == "" {
			return fmt.Errorf("--status is required")
		}

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		status := thread.NormalizeStatus(strings.TrimSpace(threadUpdateStatus))
		switch status {
		case thread.StatusForming, thread.StatusActive, thread.StatusConverged, thread.StatusSubsumed, thread.StatusResolved, thread.StatusPromoted:
		default:
			return fmt.Errorf("invalid thread status %q", threadUpdateStatus)
		}

		resolvedTo, err := resolveToForStatus(cmd, status, threadUpdateTo)
		if err != nil {
			return err
		}

		if err := thread.UpdateStatus(dir, slug, status, resolvedTo); err != nil {
			return err
		}

		if resolvedTo != "" {
			fmt.Printf("Thread updated: %s (%s -> %s)\n", slug, status, resolvedTo)
		} else {
			fmt.Printf("Thread updated: %s (%s)\n", slug, status)
		}
		return nil
	},
}

var threadLinkCmd = &cobra.Command{
	Use:   "link <thread-slug> <beads-id>",
	Short: "Link a thread to active work (beads issue)",
	Long: `Add a beads issue ID to a thread's active_work list.

This creates a bidirectional connection between a thread and spawned work,
so completion can back-propagate to the thread.

Examples:
  orch thread link coordination-primitives orch-go-abc12
  orch thread link daemon-capacity orch-go-def34`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		beadsID := args[1]

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		if err := thread.LinkWork(dir, slug, beadsID); err != nil {
			return err
		}

		fmt.Printf("Linked %s -> %s\n", slug, beadsID)
		return nil
	},
}

var (
	threadPromoteAs     string
	threadPromoteDryRun bool
	threadPromoteName   string
)

var threadPromoteCmd = &cobra.Command{
	Use:   "promote <slug>",
	Short: "Promote a converged thread into a durable artifact",
	Long: `Promote a converged thread into a model or decision.

Creates the target artifact with provenance from the thread, updates the
thread status to promoted, and propagates the promotion to ancestor threads.

Use --name to specify the artifact directory name when the thread slug
(derived from the title) is too long or doesn't capture the concept.

Examples:
  orch thread promote generative-systems-organized-around --as model --name named-incompleteness
  orch thread promote product-surface-five-elements-not --as decision --name product-surface
  orch thread promote converged-idea --as model
  orch thread promote converged-idea --as model --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]

		dir, err := threadsDir()
		if err != nil {
			return err
		}

		// Validate --as flag
		switch threadPromoteAs {
		case "model", "decision":
		default:
			return fmt.Errorf("--as must be 'model' or 'decision', got %q", threadPromoteAs)
		}

		// Load thread to get content for scaffold
		shown, err := thread.Show(dir, slug)
		if err != nil {
			return err
		}

		if shown.Status != thread.StatusConverged {
			return fmt.Errorf("thread %q has status %q, must be converged to promote", slug, shown.Status)
		}

		// Determine target path — use --name if provided, otherwise slug
		projectDir, _, err := identity.ResolveProjectDirectory(threadWorkdir)
		if err != nil {
			return err
		}

		artifactName := slug
		if threadPromoteName != "" {
			artifactName = threadPromoteName
		}

		var targetPath string
		switch threadPromoteAs {
		case "model":
			targetPath = filepath.Join(".kb", "models", artifactName, "model.md")
		case "decision":
			today := time.Now().Format("2006-01-02")
			targetPath = filepath.Join(".kb", "decisions", today+"-"+artifactName+".md")
		}

		if threadPromoteDryRun {
			fmt.Printf("Would promote: %s -> %s (%s)\n", slug, targetPath, threadPromoteAs)
			fmt.Printf("Thread title: %s\n", shown.Title)
			fmt.Printf("Thread entries: %d\n", len(shown.Entries))
			return nil
		}

		// Scaffold target artifact
		absTargetPath := filepath.Join(projectDir, targetPath)
		if err := scaffoldPromotionArtifact(absTargetPath, threadPromoteAs, shown); err != nil {
			return err
		}

		// Promote the thread (updates status, promoted_to, propagates ancestors)
		if err := thread.Promote(dir, slug, threadPromoteAs, targetPath); err != nil {
			return err
		}

		relTarget, _ := filepath.Rel(mustGetwd(), absTargetPath)
		fmt.Printf("Promoted: %s -> %s (%s)\n", slug, relTarget, threadPromoteAs)
		return nil
	},
}

// scaffoldPromotionArtifact creates the target artifact file with provenance.
func scaffoldPromotionArtifact(absPath, artifactType string, t *thread.Thread) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("creating artifact directory: %w", err)
	}

	// For models, also create the probes/ subdirectory and seed claims.yaml
	if artifactType == "model" {
		modelDir := filepath.Dir(absPath)
		probesDir := filepath.Join(modelDir, "probes")
		if err := os.MkdirAll(probesDir, 0755); err != nil {
			return fmt.Errorf("creating probes directory: %w", err)
		}

		// Create seed claims.yaml so downstream consumers (completion pipeline,
		// daemon probe generation) don't fail with 'no such file or directory'.
		today := time.Now().Format("2006-01-02")
		seedClaims := &claims.File{
			Model:     filepath.Base(modelDir),
			Version:   1,
			LastAudit: today,
			Claims:    []claims.Claim{},
		}
		claimsPath := filepath.Join(modelDir, "claims.yaml")
		if err := claims.SaveFile(claimsPath, seedClaims); err != nil {
			return fmt.Errorf("creating seed claims.yaml: %w", err)
		}
	}

	var content string
	switch artifactType {
	case "model":
		content = scaffoldModel(t)
	case "decision":
		content = scaffoldDecision(t)
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

func scaffoldModel(t *thread.Thread) string {
	// Collect entry text for the initial thesis
	var entryText strings.Builder
	for _, e := range t.Entries {
		entryText.WriteString(fmt.Sprintf("**%s:** %s\n\n", e.Date, e.Text))
	}

	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`# Model: %s

**Domain:** {System area this model describes}
**Last Updated:** %s
**Promoted From:** Thread "%s" (%s)

---

## Summary (30 seconds)

{Synthesize from thread entries below}

---

## Core Mechanism

{How does this component/system actually work?}

---

## Thread Lineage

%s
---

## References

**Thread:**
- .kb/threads/%s — Promoted thread
`, t.Title, today, t.Title, t.Filename, entryText.String(), t.Filename)
}

func scaffoldDecision(t *thread.Thread) string {
	var entryText strings.Builder
	for _, e := range t.Entries {
		entryText.WriteString(fmt.Sprintf("**%s:** %s\n\n", e.Date, e.Text))
	}

	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`# Decision: %s

**Date:** %s
**Status:** Accepted
**Promoted From:** Thread "%s" (%s)

## Context

{Why was this decision needed?}

## Thread Lineage

%s
## Decision

{What was decided?}

## Consequences

{What follows from this decision?}
`, t.Title, today, t.Title, t.Filename, entryText.String())
}

func init() {
	threadCmd.AddCommand(threadNewCmd)
	threadCmd.AddCommand(threadAppendCmd)
	threadCmd.AddCommand(threadListCmd)
	threadCmd.AddCommand(threadShowCmd)
	threadCmd.AddCommand(threadResolveCmd)
	threadCmd.AddCommand(threadUpdateCmd)
	threadCmd.AddCommand(threadLinkCmd)
	threadCmd.AddCommand(threadPromoteCmd)

	threadCmd.PersistentFlags().StringVar(&threadWorkdir, "workdir", "", "Target project directory (for cross-project thread operations)")
	threadNewCmd.Flags().StringVar(&threadNewFrom, "from", "", "Parent thread slug (creates child thread with spawned_from reference)")
	threadResolveCmd.Flags().StringVar(&threadResolveTo, "to", "", "Target artifact path (e.g., .kb/models/enforcement.md)")
	threadUpdateCmd.Flags().StringVar(&threadUpdateStatus, "status", "", "New lifecycle status (forming, active, converged, subsumed, resolved, promoted)")
	threadUpdateCmd.Flags().StringVar(&threadUpdateTo, "to", "", "Target artifact path or brief when resolving")
	threadPromoteCmd.Flags().StringVar(&threadPromoteAs, "as", "model", "Target artifact type: model or decision")
	threadPromoteCmd.Flags().BoolVar(&threadPromoteDryRun, "dry-run", false, "Preview promotion without making changes")
	threadPromoteCmd.Flags().StringVar(&threadPromoteName, "name", "", "Override artifact directory name (use when thread slug is too long or unclear)")
}

func threadStatusIcon(status string) string {
	switch status {
	case thread.StatusForming:
		return "[~]"
	case thread.StatusActive:
		return "[*]"
	case thread.StatusConverged:
		return "[x]"
	case thread.StatusSubsumed:
		return "[>]"
	case thread.StatusResolved:
		return "[x]"
	case thread.StatusPromoted:
		return "[^]"
	default:
		return "[?]"
	}
}

func mustGetwd() string {
	dir, _ := os.Getwd()
	return dir
}
