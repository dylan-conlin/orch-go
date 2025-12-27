// Package urltomd provides URL-to-Markdown conversion using headless Chrome.
//
// This package fetches web pages using chromedp (Chrome DevTools Protocol),
// waits for JavaScript to render, and converts the resulting HTML to Markdown.
// It replaces the previous Python-based approach (shot-scraper + markitdown)
// with a pure Go implementation.
//
// Key features:
//   - JavaScript rendering via headless Chrome
//   - Configurable wait time for dynamic content
//   - CSS selector targeting for specific content extraction
//   - Automatic relative-to-absolute URL conversion
//   - Clean Markdown output suitable for LLMs
//
// Example usage:
//
//	result, err := urltomd.Fetch(ctx, "https://example.com", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Markdown)
package urltomd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// Options configures the URL-to-Markdown conversion.
type Options struct {
	// WaitTime is the duration to wait after page load for JavaScript to render.
	// Default: 2 seconds.
	WaitTime time.Duration

	// Selector is a CSS selector to target specific content.
	// If empty, the entire page body is converted.
	Selector string

	// Timeout is the maximum duration for the entire fetch operation.
	// Default: 30 seconds.
	Timeout time.Duration

	// UserAgent sets a custom user agent string.
	// If empty, uses Chrome's default user agent.
	UserAgent string

	// Headless controls whether Chrome runs in headless mode.
	// Default: true.
	Headless *bool
}

// DefaultOptions returns the default options for URL-to-Markdown conversion.
func DefaultOptions() *Options {
	headless := true
	return &Options{
		WaitTime: 2 * time.Second,
		Timeout:  30 * time.Second,
		Headless: &headless,
	}
}

// Result contains the output of a URL-to-Markdown conversion.
type Result struct {
	// Markdown is the converted Markdown content.
	Markdown string

	// Title is the page title, if available.
	Title string

	// URL is the final URL after any redirects.
	URL string

	// FetchDuration is how long the fetch operation took.
	FetchDuration time.Duration

	// ConvertDuration is how long the HTML-to-Markdown conversion took.
	ConvertDuration time.Duration
}

// Fetch retrieves a URL and converts it to Markdown.
//
// The function:
// 1. Launches a headless Chrome browser
// 2. Navigates to the URL and waits for JavaScript to render
// 3. Extracts the HTML (optionally targeting a specific selector)
// 4. Converts the HTML to Markdown
//
// If opts is nil, DefaultOptions() is used.
func Fetch(ctx context.Context, targetURL string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Apply timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create browser context with options
	allocOpts := chromedp.DefaultExecAllocatorOptions[:]
	if opts.Headless != nil && *opts.Headless {
		allocOpts = append(allocOpts, chromedp.Headless)
	}
	if opts.UserAgent != "" {
		allocOpts = append(allocOpts, chromedp.UserAgent(opts.UserAgent))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	result := &Result{
		URL: targetURL,
	}

	// Fetch the page
	fetchStart := time.Now()

	var html string
	var title string
	var finalURL string

	actions := []chromedp.Action{
		chromedp.Navigate(targetURL),
	}

	// Add wait time if specified
	if opts.WaitTime > 0 {
		actions = append(actions, chromedp.Sleep(opts.WaitTime))
	}

	// Add actions to extract content
	if opts.Selector != "" {
		// Target specific element
		actions = append(actions,
			chromedp.OuterHTML(opts.Selector, &html, chromedp.ByQuery),
		)
	} else {
		// Get full page HTML
		actions = append(actions,
			chromedp.ActionFunc(func(ctx context.Context) error {
				node, err := dom.GetDocument().Do(ctx)
				if err != nil {
					return err
				}
				html, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
				return err
			}),
		)
	}

	// Get page title and final URL
	actions = append(actions,
		chromedp.Title(&title),
		chromedp.Location(&finalURL),
	)

	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	result.FetchDuration = time.Since(fetchStart)
	result.Title = title
	if finalURL != "" {
		result.URL = finalURL
	}

	// Convert HTML to Markdown
	convertStart := time.Now()

	markdown, err := htmlToMarkdown(html, result.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	result.Markdown = markdown
	result.ConvertDuration = time.Since(convertStart)

	return result, nil
}

// htmlToMarkdown converts HTML content to Markdown.
// The domain parameter is used to convert relative URLs to absolute.
func htmlToMarkdown(html string, pageURL string) (string, error) {
	// Parse the page URL to get the domain for relative URL conversion
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse page URL: %w", err)
	}
	domain := parsedURL.Scheme + "://" + parsedURL.Host

	// Create converter with commonmark plugin for standard Markdown output
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)

	// Convert HTML to Markdown with domain option for relative URL conversion
	markdown, err := conv.ConvertString(html, converter.WithDomain(domain))
	if err != nil {
		return "", err
	}

	// Clean up the output
	markdown = cleanMarkdown(markdown)

	return markdown, nil
}

// cleanMarkdown performs post-processing on the Markdown output.
func cleanMarkdown(markdown string) string {
	// Trim leading/trailing whitespace
	markdown = strings.TrimSpace(markdown)

	// Collapse multiple consecutive blank lines into two
	for strings.Contains(markdown, "\n\n\n") {
		markdown = strings.ReplaceAll(markdown, "\n\n\n", "\n\n")
	}

	return markdown
}

// FetchWithFrontmatter fetches a URL and returns Markdown with YAML frontmatter.
// The frontmatter includes the source URL and fetch timestamp.
func FetchWithFrontmatter(ctx context.Context, targetURL string, opts *Options) (*Result, error) {
	result, err := Fetch(ctx, targetURL, opts)
	if err != nil {
		return nil, err
	}

	// Add YAML frontmatter
	frontmatter := fmt.Sprintf(`---
source: %s
fetched: %s
title: %s
---

`, result.URL, time.Now().Format(time.RFC3339), result.Title)

	result.Markdown = frontmatter + result.Markdown

	return result, nil
}
