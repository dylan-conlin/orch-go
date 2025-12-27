package urltomd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.WaitTime != 2*time.Second {
		t.Errorf("expected WaitTime 2s, got %v", opts.WaitTime)
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", opts.Timeout)
	}
	if opts.Headless == nil || !*opts.Headless {
		t.Error("expected Headless to be true by default")
	}
	if opts.Selector != "" {
		t.Errorf("expected empty Selector, got %q", opts.Selector)
	}
	if opts.UserAgent != "" {
		t.Errorf("expected empty UserAgent, got %q", opts.UserAgent)
	}
}

func TestCleanMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim whitespace",
			input:    "  hello world  \n\n",
			expected: "hello world",
		},
		{
			name:     "collapse multiple blank lines",
			input:    "hello\n\n\n\nworld",
			expected: "hello\n\nworld",
		},
		{
			name:     "collapse many blank lines",
			input:    "a\n\n\n\n\n\nb",
			expected: "a\n\nb",
		},
		{
			name:     "preserve double newlines",
			input:    "hello\n\nworld",
			expected: "hello\n\nworld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("cleanMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHtmlToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		pageURL  string
		contains []string // substrings that should be in the result
	}{
		{
			name:     "simple heading",
			html:     "<html><body><h1>Hello World</h1></body></html>",
			pageURL:  "https://example.com",
			contains: []string{"# Hello World"},
		},
		{
			name:     "paragraph",
			html:     "<html><body><p>This is a paragraph.</p></body></html>",
			pageURL:  "https://example.com",
			contains: []string{"This is a paragraph."},
		},
		{
			name:     "link",
			html:     `<html><body><a href="https://example.com">Link</a></body></html>`,
			pageURL:  "https://example.com",
			contains: []string{"[Link](https://example.com)"},
		},
		{
			name:     "relative link converted to absolute",
			html:     `<html><body><a href="/page">Link</a></body></html>`,
			pageURL:  "https://example.com",
			contains: []string{"[Link](https://example.com/page)"},
		},
		{
			name:     "list",
			html:     "<html><body><ul><li>Item 1</li><li>Item 2</li></ul></body></html>",
			pageURL:  "https://example.com",
			contains: []string{"- Item 1", "- Item 2"},
		},
		{
			name:     "bold and italic",
			html:     "<html><body><p><strong>Bold</strong> and <em>italic</em></p></body></html>",
			pageURL:  "https://example.com",
			contains: []string{"**Bold**", "*italic*"},
		},
		{
			name:     "code block",
			html:     "<html><body><pre><code>console.log('hello');</code></pre></body></html>",
			pageURL:  "https://example.com",
			contains: []string{"console.log('hello');"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := htmlToMarkdown(tt.html, tt.pageURL)
			if err != nil {
				t.Fatalf("htmlToMarkdown error: %v", err)
			}

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("result missing %q\nGot: %s", substr, result)
				}
			}
		})
	}
}

func TestHtmlToMarkdown_InvalidURL(t *testing.T) {
	_, err := htmlToMarkdown("<html></html>", "://invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		url  string
	}{
		{"empty URL", ""},
		{"no scheme", "example.com"},
		{"file scheme", "file:///etc/passwd"},
		{"ftp scheme", "ftp://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Fetch(ctx, tt.url, nil)
			if err == nil {
				t.Errorf("expected error for URL %q", tt.url)
			}
		})
	}
}

// TestFetch_Integration tests the full fetch flow with a real HTTP server.
// This test requires Chrome to be installed, so it's skipped in CI.
func TestFetch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Hello from Test Server</h1>
				<p>This is a test paragraph.</p>
				<a href="/other">Link to other page</a>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	ctx := context.Background()
	opts := DefaultOptions()
	opts.WaitTime = 100 * time.Millisecond // Short wait for test
	opts.Timeout = 10 * time.Second

	result, err := Fetch(ctx, server.URL, opts)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	// Check result
	if result.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", result.Title)
	}

	if !strings.Contains(result.Markdown, "# Hello from Test Server") {
		t.Errorf("markdown missing heading\nGot: %s", result.Markdown)
	}

	if !strings.Contains(result.Markdown, "This is a test paragraph.") {
		t.Errorf("markdown missing paragraph\nGot: %s", result.Markdown)
	}

	// Check that relative link is converted to absolute
	if !strings.Contains(result.Markdown, server.URL+"/other") {
		t.Errorf("markdown missing absolute link\nGot: %s", result.Markdown)
	}

	// Check timing
	if result.FetchDuration == 0 {
		t.Error("FetchDuration should not be zero")
	}
	if result.ConvertDuration == 0 {
		t.Error("ConvertDuration should not be zero")
	}
}

// TestFetch_WithSelector tests content extraction with a CSS selector.
func TestFetch_WithSelector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a test server with a specific element to target
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Test Page</title></head>
			<body>
				<header><nav>Navigation</nav></header>
				<main id="content">
					<h1>Main Content</h1>
					<p>This is the main content area.</p>
				</main>
				<footer>Footer stuff</footer>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	ctx := context.Background()
	opts := DefaultOptions()
	opts.WaitTime = 100 * time.Millisecond
	opts.Timeout = 10 * time.Second
	opts.Selector = "#content"

	result, err := Fetch(ctx, server.URL, opts)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	// Should contain main content
	if !strings.Contains(result.Markdown, "Main Content") {
		t.Errorf("markdown missing main content\nGot: %s", result.Markdown)
	}

	// Should NOT contain header or footer (since we targeted #content)
	if strings.Contains(result.Markdown, "Navigation") {
		t.Errorf("markdown should not contain navigation\nGot: %s", result.Markdown)
	}
	if strings.Contains(result.Markdown, "Footer stuff") {
		t.Errorf("markdown should not contain footer\nGot: %s", result.Markdown)
	}
}

// TestFetchWithFrontmatter tests the frontmatter generation.
func TestFetchWithFrontmatter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Frontmatter Test</title></head>
			<body><h1>Content</h1></body>
			</html>
		`))
	}))
	defer server.Close()

	ctx := context.Background()
	opts := DefaultOptions()
	opts.WaitTime = 100 * time.Millisecond
	opts.Timeout = 10 * time.Second

	result, err := FetchWithFrontmatter(ctx, server.URL, opts)
	if err != nil {
		t.Fatalf("FetchWithFrontmatter error: %v", err)
	}

	// Check frontmatter
	if !strings.HasPrefix(result.Markdown, "---\n") {
		t.Errorf("markdown should start with YAML frontmatter\nGot: %s", result.Markdown[:min(100, len(result.Markdown))])
	}

	if !strings.Contains(result.Markdown, "source: "+server.URL) {
		t.Errorf("frontmatter missing source URL\nGot: %s", result.Markdown)
	}

	if !strings.Contains(result.Markdown, "fetched:") {
		t.Errorf("frontmatter missing fetched timestamp\nGot: %s", result.Markdown)
	}

	if !strings.Contains(result.Markdown, "title: Frontmatter Test") {
		t.Errorf("frontmatter missing title\nGot: %s", result.Markdown)
	}
}

// TestFetch_Timeout tests that timeouts are respected.
func TestFetch_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Delay longer than timeout
		w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer server.Close()

	ctx := context.Background()
	opts := DefaultOptions()
	opts.Timeout = 1 * time.Second // Very short timeout

	_, err := Fetch(ctx, server.URL, opts)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
