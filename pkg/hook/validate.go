package hook

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ValidationSeverity indicates the severity of a validation issue.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// ValidationIssue represents a problem found during config validation.
type ValidationIssue struct {
	Event    string
	Matcher  string
	Command  string
	Severity ValidationSeverity
	Message  string
}

// ValidateConfig performs static validation on the hook configuration.
func ValidateConfig(settings *Settings) []ValidationIssue {
	var issues []ValidationIssue

	for event, groups := range settings.Hooks {
		for _, group := range groups {
			for _, h := range group.Hooks {
				issues = append(issues, validateHookConfig(event, group.Matcher, h)...)
			}
		}
	}

	return issues
}

func validateHookConfig(event, matcher string, h HookConfig) []ValidationIssue {
	var issues []ValidationIssue

	// Check matcher is valid regex
	if matcher != "" {
		if _, err := regexp.Compile("^(" + matcher + ")$"); err != nil {
			issues = append(issues, ValidationIssue{
				Event:    event,
				Matcher:  matcher,
				Command:  h.Command,
				Severity: SeverityError,
				Message:  fmt.Sprintf("invalid matcher regex: %v", err),
			})
		}
	}

	// Check command file exists (only for path-based commands, not bare commands like "bd prime")
	expanded := expandCommand(h.Command)
	if strings.Contains(h.Command, "/") || strings.Contains(h.Command, "$HOME") || strings.Contains(h.Command, "${HOME}") {
		info, err := os.Stat(expanded)
		if os.IsNotExist(err) {
			issues = append(issues, ValidationIssue{
				Event:    event,
				Matcher:  matcher,
				Command:  h.Command,
				Severity: SeverityError,
				Message:  fmt.Sprintf("command not found: %s", expanded),
			})
		} else if err == nil {
			// Check executable
			if info.Mode()&0111 == 0 {
				issues = append(issues, ValidationIssue{
					Event:    event,
					Matcher:  matcher,
					Command:  h.Command,
					Severity: SeverityError,
					Message:  fmt.Sprintf("command not executable: %s", expanded),
				})
			}

			// Check shebang for script files
			if !info.IsDir() {
				issues = append(issues, checkShebang(event, matcher, h.Command, expanded)...)
			}
		}
	}

	// Check timeout
	if h.Timeout == 0 {
		issues = append(issues, ValidationIssue{
			Event:    event,
			Matcher:  matcher,
			Command:  h.Command,
			Severity: SeverityWarning,
			Message:  "no timeout set (default: 600s — likely unintentional for hooks)",
		})
	} else if h.Timeout > 60 {
		issues = append(issues, ValidationIssue{
			Event:    event,
			Matcher:  matcher,
			Command:  h.Command,
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("timeout %ds is high (>60s) — may slow session operations", h.Timeout),
		})
	}

	// Check for SessionStart-specific warnings
	if event == "SessionStart" && h.Timeout > 15 {
		issues = append(issues, ValidationIssue{
			Event:    event,
			Matcher:  matcher,
			Command:  h.Command,
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("timeout %ds for SessionStart hook — may noticeably slow startup", h.Timeout),
		})
	}

	return issues
}

func checkShebang(event, matcher, command, expanded string) []ValidationIssue {
	var issues []ValidationIssue

	f, err := os.Open(expanded)
	if err != nil {
		return issues
	}
	defer f.Close()

	// Read first 256 bytes for shebang
	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil || n < 2 {
		return issues
	}

	content := string(buf[:n])
	if !strings.HasPrefix(content, "#!") {
		// Only warn for script-like extensions
		if strings.HasSuffix(expanded, ".py") || strings.HasSuffix(expanded, ".sh") {
			issues = append(issues, ValidationIssue{
				Event:    event,
				Matcher:  matcher,
				Command:  command,
				Severity: SeverityWarning,
				Message:  "script file missing shebang line (#!/...)",
			})
		}
	}

	return issues
}

// FormatIssues formats validation issues for display.
func FormatIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return "No issues found"
	}

	var b strings.Builder
	errors := 0
	warnings := 0

	for _, issue := range issues {
		var prefix string
		switch issue.Severity {
		case SeverityError:
			prefix = "  ERROR"
			errors++
		case SeverityWarning:
			prefix = "  WARN "
			warnings++
		case SeverityInfo:
			prefix = "  INFO "
		}

		basename := CommandBasename(issue.Command)
		fmt.Fprintf(&b, "%s [%s] %s (matcher: %s): %s\n",
			prefix, issue.Event, basename, issue.Matcher, issue.Message)
	}

	fmt.Fprintf(&b, "\n%d error(s), %d warning(s)", errors, warnings)
	return b.String()
}
