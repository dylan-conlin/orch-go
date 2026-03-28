package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownClaims(t *testing.T) {
	content := `# Model: Test Model

## Summary
Some summary text.

## Claims (Testable)

| ID | Claim | How to Verify |
|----|-------|---------------|
| NI-01 | Named gaps compose; unnamed completeness doesn't. | Compare clustering effectiveness |
| NI-02 | Every success preserves named incompleteness. | Classify features. |
| NI-03 | The mechanism is substrate-independent. | Test in new domain. |

## References
Some refs.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "model.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	claims, err := ParseMarkdownClaims(path)
	if err != nil {
		t.Fatalf("ParseMarkdownClaims error: %v", err)
	}

	if len(claims) != 3 {
		t.Fatalf("expected 3 claims, got %d", len(claims))
	}

	if claims[0].ID != "NI-01" {
		t.Errorf("claim 0 ID = %q, want NI-01", claims[0].ID)
	}
	if claims[0].Text != "Named gaps compose; unnamed completeness doesn't." {
		t.Errorf("claim 0 Text = %q", claims[0].Text)
	}
	if claims[0].HowToVerify != "Compare clustering effectiveness" {
		t.Errorf("claim 0 HowToVerify = %q", claims[0].HowToVerify)
	}

	if claims[2].ID != "NI-03" {
		t.Errorf("claim 2 ID = %q, want NI-03", claims[2].ID)
	}
}

func TestParseMarkdownClaims_NoClaims(t *testing.T) {
	content := `# Model: No Claims

## Summary
No claims section here.

## References
`
	dir := t.TempDir()
	path := filepath.Join(dir, "model.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	claims, err := ParseMarkdownClaims(path)
	if err != nil {
		t.Fatalf("ParseMarkdownClaims error: %v", err)
	}

	if len(claims) != 0 {
		t.Errorf("expected 0 claims, got %d", len(claims))
	}
}

func TestParseMarkdownClaims_FileNotFound(t *testing.T) {
	_, err := ParseMarkdownClaims("/nonexistent/model.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		name string
		line string
		want *MarkdownClaim
	}{
		{
			name: "standard row",
			line: "| NI-01 | Named gaps compose. | Test clustering. |",
			want: &MarkdownClaim{ID: "NI-01", Text: "Named gaps compose.", HowToVerify: "Test clustering."},
		},
		{
			name: "no verify column",
			line: "| CA-01 | Atoms with outward signals. |",
			want: &MarkdownClaim{ID: "CA-01", Text: "Atoms with outward signals."},
		},
		{
			name: "not a claim ID",
			line: "| # | Class Name | Definition |",
			want: nil,
		},
		{
			name: "separator row",
			line: "|---|---|---|",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTableRow(tt.line)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseTableRow(%q) = %+v, want nil", tt.line, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseTableRow(%q) = nil, want %+v", tt.line, tt.want)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.want.ID)
			}
			if got.Text != tt.want.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.want.Text)
			}
			if got.HowToVerify != tt.want.HowToVerify {
				t.Errorf("HowToVerify = %q, want %q", got.HowToVerify, tt.want.HowToVerify)
			}
		})
	}
}
