// Package main provides the CLI entry point for orch-go.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/urltomd"
	"github.com/spf13/cobra"
)

// ============================================================================
// Fetch-MD Command - Fetch URL and convert to Markdown
// ============================================================================

var (
	fetchmdWait        int    // Wait time in milliseconds
	fetchmdSelector    string // CSS selector for content targeting
	fetchmdTimeout     int    // Timeout in seconds
	fetchmdFrontmatter bool   // Include YAML frontmatter
	fetchmdOutput      string // Output file path (default: stdout)
	fetchmdQuiet       bool   // Suppress status messages
)

var fetchmdCmd = &cobra.Command{
	Use:    "fetch-md [url]",
	Short:  "Fetch a URL and convert it to Markdown",
	Hidden: true,
	Long: `Fetch a web page and convert it to Markdown.

Uses headless Chrome to render JavaScript before extracting content,
then converts the HTML to clean Markdown suitable for LLMs.

This is a pure Go replacement for the Python-based url-to-markdown
pipeline (shot-scraper + markitdown).

Examples:
  orch-go fetch-md https://example.com
  orch-go fetch-md https://example.com --wait 3000       # Wait 3s for JS
  orch-go fetch-md https://example.com --selector "main" # Target specific element
  orch-go fetch-md https://example.com --frontmatter     # Add YAML metadata
  orch-go fetch-md https://example.com -o page.md        # Write to file`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFetchMD(args[0])
	},
}

func init() {
	fetchmdCmd.Flags().IntVar(&fetchmdWait, "wait", 2000, "Wait time in milliseconds for JavaScript to render")
	fetchmdCmd.Flags().StringVar(&fetchmdSelector, "selector", "", "CSS selector to target specific content")
	fetchmdCmd.Flags().IntVar(&fetchmdTimeout, "timeout", 30, "Timeout in seconds for the entire operation")
	fetchmdCmd.Flags().BoolVar(&fetchmdFrontmatter, "frontmatter", false, "Include YAML frontmatter with source URL and date")
	fetchmdCmd.Flags().StringVarP(&fetchmdOutput, "output", "o", "", "Output file path (default: stdout)")
	fetchmdCmd.Flags().BoolVarP(&fetchmdQuiet, "quiet", "q", false, "Suppress status messages")

	rootCmd.AddCommand(fetchmdCmd)
}

func runFetchMD(url string) error {
	ctx := context.Background()

	// Build options
	opts := urltomd.DefaultOptions()
	opts.WaitTime = time.Duration(fetchmdWait) * time.Millisecond
	opts.Timeout = time.Duration(fetchmdTimeout) * time.Second
	if fetchmdSelector != "" {
		opts.Selector = fetchmdSelector
	}

	// Status message
	if !fetchmdQuiet {
		fmt.Fprintf(os.Stderr, "Fetching: %s\n", url)
		if fetchmdSelector != "" {
			fmt.Fprintf(os.Stderr, "Selector: %s\n", fetchmdSelector)
		}
		fmt.Fprintf(os.Stderr, "Wait:     %dms\n", fetchmdWait)
	}

	// Fetch and convert
	var result *urltomd.Result
	var err error

	if fetchmdFrontmatter {
		result, err = urltomd.FetchWithFrontmatter(ctx, url, opts)
	} else {
		result, err = urltomd.Fetch(ctx, url, opts)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Status message
	if !fetchmdQuiet {
		fmt.Fprintf(os.Stderr, "Title:    %s\n", result.Title)
		fmt.Fprintf(os.Stderr, "Fetch:    %v\n", result.FetchDuration.Round(time.Millisecond))
		fmt.Fprintf(os.Stderr, "Convert:  %v\n", result.ConvertDuration.Round(time.Millisecond))
		fmt.Fprintf(os.Stderr, "---\n")
	}

	// Output
	if fetchmdOutput != "" {
		// Write to file
		if err := os.WriteFile(fetchmdOutput, []byte(result.Markdown), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if !fetchmdQuiet {
			fmt.Fprintf(os.Stderr, "Written to: %s\n", fetchmdOutput)
		}
	} else {
		// Write to stdout
		fmt.Println(result.Markdown)
	}

	return nil
}
