package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/sessions"
	"github.com/spf13/cobra"
)

var (
	// Sessions command flags
	sessionsLimit     int
	sessionsDate      string
	sessionsAfter     string
	sessionsBefore    string
	sessionsDirectory string
	sessionsRegex     bool
	sessionsCase      bool
)

var sessionHistoryCmd = &cobra.Command{
	Use:   "session-history",
	Short: "Search and list OpenCode session history",
	Long: `Search and list OpenCode session history.

Sessions are persisted by OpenCode at ~/.local/share/opencode/storage/.
This command allows searching through session titles and message content
to find past work, insights, and decisions.

Subcommands:
  list    - List recent sessions
  search  - Full-text search of session content
  show    - View a specific session

Examples:
  orch session-history list                    # List recent sessions
  orch session-history list --limit 20         # List last 20 sessions
  orch session-history list --date 2025-12-25  # Sessions from specific date
  orch session-history search "teeth check"    # Search for text in sessions
  orch session-history search --regex "auth.*token"  # Regex search
  orch session-history show ses_abc123         # Show specific session`,
}

var sessionHistoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent sessions",
	Long: `List recent OpenCode sessions with summaries.

Sessions are sorted by most recently updated. Use --limit to control
how many sessions are shown.

Examples:
  orch session-history list                    # List recent sessions (default: 10)
  orch session-history list --limit 50         # List last 50 sessions
  orch session-history list --date 2025-12-25  # Sessions from specific date
  orch session-history list --after 2025-12-20 # Sessions after date
  orch session-history list --directory /path/to/project  # Filter by project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionsList()
	},
}

var sessionHistorySearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search session message content",
	Long: `Search through session message content for matching text.

This command searches the actual message content (not just titles) by
fetching messages from the OpenCode API. Results show matching sessions
with context snippets.

Requires OpenCode to be running (uses API to fetch message content).

Examples:
  orch session-history search "error handling"        # Search for text
  orch session-history search "teeth check"           # Find specific discussion
  orch session-history search --regex "auth.*token"   # Regex search
  orch session-history search -i "ERROR"              # Case-insensitive (default)
  orch session-history search --case "Error"          # Case-sensitive
  orch session-history search --limit 5 "pattern"     # Limit results`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		return runSessionsSearch(query)
	},
}

var sessionHistoryShowCmd = &cobra.Command{
	Use:   "show [session-id]",
	Short: "Show session details and messages",
	Long: `Show detailed information about a specific session.

Displays session metadata and message content. Requires OpenCode
to be running to fetch message content.

Examples:
  orch session-history show ses_abc123   # Show specific session
  orch session-history show ses_xyz789   # View session messages`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		return runSessionsShow(sessionID)
	},
}

func init() {
	// List flags
	sessionHistoryListCmd.Flags().IntVar(&sessionsLimit, "limit", 10, "Maximum number of sessions to show")
	sessionHistoryListCmd.Flags().StringVar(&sessionsDate, "date", "", "Filter by specific date (YYYY-MM-DD)")
	sessionHistoryListCmd.Flags().StringVar(&sessionsAfter, "after", "", "Sessions created after date (YYYY-MM-DD)")
	sessionHistoryListCmd.Flags().StringVar(&sessionsBefore, "before", "", "Sessions created before date (YYYY-MM-DD)")
	sessionHistoryListCmd.Flags().StringVarP(&sessionsDirectory, "directory", "d", "", "Filter by project directory")

	// Search flags
	sessionHistorySearchCmd.Flags().IntVar(&sessionsLimit, "limit", 10, "Maximum number of results")
	sessionHistorySearchCmd.Flags().BoolVar(&sessionsRegex, "regex", false, "Treat query as regular expression")
	sessionHistorySearchCmd.Flags().BoolVar(&sessionsCase, "case", false, "Case-sensitive search (default: case-insensitive)")
	sessionHistorySearchCmd.Flags().StringVar(&sessionsDate, "date", "", "Filter by specific date (YYYY-MM-DD)")
	sessionHistorySearchCmd.Flags().StringVar(&sessionsAfter, "after", "", "Sessions created after date (YYYY-MM-DD)")
	sessionHistorySearchCmd.Flags().StringVar(&sessionsBefore, "before", "", "Sessions created before date (YYYY-MM-DD)")
	sessionHistorySearchCmd.Flags().StringVarP(&sessionsDirectory, "directory", "d", "", "Filter by project directory")

	sessionHistoryCmd.AddCommand(sessionHistoryListCmd)
	sessionHistoryCmd.AddCommand(sessionHistorySearchCmd)
	sessionHistoryCmd.AddCommand(sessionHistoryShowCmd)

	rootCmd.AddCommand(sessionHistoryCmd)
}

func runSessionsList() error {
	client := opencode.NewClient(serverURL)
	store := sessions.NewStore("", client)

	opts := sessions.ListOptions{
		Limit: sessionsLimit,
	}

	if sessionsDirectory != "" {
		opts.Directory = sessionsDirectory
	}

	// Parse date filters
	if sessionsDate != "" {
		date, err := time.Parse("2006-01-02", sessionsDate)
		if err != nil {
			return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
		}
		after := date
		before := date.Add(24 * time.Hour)
		opts.After = &after
		opts.Before = &before
	}
	if sessionsAfter != "" {
		after, err := time.Parse("2006-01-02", sessionsAfter)
		if err != nil {
			return fmt.Errorf("invalid after date format (use YYYY-MM-DD): %w", err)
		}
		opts.After = &after
	}
	if sessionsBefore != "" {
		before, err := time.Parse("2006-01-02", sessionsBefore)
		if err != nil {
			return fmt.Errorf("invalid before date format (use YYYY-MM-DD): %w", err)
		}
		opts.Before = &before
	}

	sessionsList, err := store.List(opts)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessionsList) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	fmt.Printf("Found %d sessions:\n\n", len(sessionsList))

	for _, s := range sessionsList {
		// Format time relative to now
		age := formatSessionAge(s.Updated)

		// Truncate title if too long
		title := s.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}

		// Format summary
		summary := ""
		if s.Summary.Files > 0 {
			summary = fmt.Sprintf("+%d/-%d in %d files", s.Summary.Additions, s.Summary.Deletions, s.Summary.Files)
		}

		fmt.Printf("%-22s  %s\n", s.ID, title)
		fmt.Printf("  Updated: %s", age)
		if summary != "" {
			fmt.Printf(" | %s", summary)
		}
		fmt.Printf("\n")
		if s.Directory != "" {
			fmt.Printf("  Project: %s\n", s.Directory)
		}
		fmt.Println()
	}

	return nil
}

func runSessionsSearch(query string) error {
	client := opencode.NewClient(serverURL)
	store := sessions.NewStore("", client)

	opts := sessions.SearchOptions{
		Query:         query,
		UseRegex:      sessionsRegex,
		CaseSensitive: sessionsCase,
		Limit:         sessionsLimit,
		Client:        client,
	}

	if sessionsDirectory != "" {
		opts.Directory = sessionsDirectory
	}

	// Parse date filters
	if sessionsDate != "" {
		date, err := time.Parse("2006-01-02", sessionsDate)
		if err != nil {
			return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
		}
		after := date
		before := date.Add(24 * time.Hour)
		opts.After = &after
		opts.Before = &before
	}
	if sessionsAfter != "" {
		after, err := time.Parse("2006-01-02", sessionsAfter)
		if err != nil {
			return fmt.Errorf("invalid after date format (use YYYY-MM-DD): %w", err)
		}
		opts.After = &after
	}
	if sessionsBefore != "" {
		before, err := time.Parse("2006-01-02", sessionsBefore)
		if err != nil {
			return fmt.Errorf("invalid before date format (use YYYY-MM-DD): %w", err)
		}
		opts.Before = &before
	}

	results, err := store.Search(opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No sessions found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Found %d sessions matching \"%s\":\n\n", len(results), query)

	for _, r := range results {
		// Format time relative to now
		age := formatSessionAge(r.Session.Updated)

		// Truncate title if too long
		title := r.Session.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		fmt.Printf("%-22s  %s (%d matches)\n", r.Session.ID, title, r.MatchCount)
		fmt.Printf("  Updated: %s\n", age)

		// Show snippets
		for _, snippet := range r.Snippets {
			// Truncate snippet if too long
			if len(snippet) > 200 {
				snippet = snippet[:197] + "..."
			}
			fmt.Printf("  > %s\n", snippet)
		}
		fmt.Println()
	}

	return nil
}

func runSessionsShow(sessionID string) error {
	client := opencode.NewClient(serverURL)
	store := sessions.NewStore("", client)

	session, messages, err := store.Show(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	fmt.Printf("Session: %s\n", session.ID)
	fmt.Printf("Title:   %s\n", session.Title)
	fmt.Printf("Project: %s\n", session.Directory)
	fmt.Printf("Created: %s\n", session.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", session.Updated.Format("2006-01-02 15:04:05"))
	if session.Summary.Files > 0 {
		fmt.Printf("Changes: +%d/-%d in %d files\n", session.Summary.Additions, session.Summary.Deletions, session.Summary.Files)
	}
	fmt.Println()

	if len(messages) == 0 {
		fmt.Println("No messages found")
		return nil
	}

	fmt.Printf("--- Messages (%d) ---\n\n", len(messages))

	for _, msg := range messages {
		role := msg.Info.Role
		roleLabel := strings.ToUpper(role[:1]) + role[1:]

		created := time.Unix(msg.Info.Time.Created/1000, 0)

		fmt.Printf("[%s] %s\n", roleLabel, created.Format("15:04:05"))

		for _, part := range msg.Parts {
			if part.Type == "text" && part.Text != "" {
				// Indent message content
				lines := strings.Split(part.Text, "\n")
				for _, line := range lines {
					fmt.Printf("  %s\n", line)
				}
			}
		}
		fmt.Println()
	}

	return nil
}

// formatSessionAge formats a time as a human-readable age string.
func formatSessionAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	return t.Format("2006-01-02")
}
