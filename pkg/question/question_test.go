// Package question provides extraction of pending questions from agent output.
package question

import (
	"testing"
)

func TestExtractFromAskUserQuestion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic question extraction",
			input:    `<parameter name="questions">[{"question": "Should I proceed with the refactoring?", "choices": ["yes", "no"]}]</parameter>`,
			expected: "Should I proceed with the refactoring?",
		},
		{
			name: "question with surrounding text",
			input: `Some text before
<parameter name="questions">[{"question": "Do you want to continue?"}]</parameter>
Some text after`,
			expected: "Do you want to continue?",
		},
		{
			name:     "no question found",
			input:    "This is just regular text without any question pattern",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractFromAskUserQuestion(tc.input)
			if result != tc.expected {
				t.Errorf("extractFromAskUserQuestion() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestExtractFromQuestionMarks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple question",
			input:    "Should I proceed?",
			expected: "Should I proceed?",
		},
		{
			name: "multi-line question",
			input: `Given the complexity of this change
and the potential impact on existing code,
should I proceed with the refactoring?`,
			expected: "Given the complexity of this change and the potential impact on existing code, should I proceed with the refactoring?",
		},
		{
			name: "question after option markers",
			input: `What would you like to do?
❯ Option 1
  Option 2
  Option 3`,
			expected: "What would you like to do?",
		},
		{
			name: "question after numbered options",
			input: `What would you like to do?
1. Option 1
2. Option 2
3. Option 3`,
			expected: "What would you like to do?",
		},
		{
			name: "most recent question after blank line",
			input: `First question?

Second question?`,
			expected: "Second question?",
		},
		{
			name: "continuous text returns last question with context",
			input: `First question?
Some other text
Second question?`,
			expected: "First question? Some other text Second question?",
		},
		{
			name:     "no question",
			input:    "This is a statement.",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name: "question with blank line before it",
			input: `Some context here

Should I continue?`,
			expected: "Should I continue?",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractFromQuestionMarks(tc.input)
			if result != tc.expected {
				t.Errorf("extractFromQuestionMarks() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "AskUserQuestion pattern takes priority",
			input: `<parameter name="questions">[{"question": "Structured question?"}]</parameter>
Is this also a question?`,
			expected: "Structured question?",
		},
		{
			name:     "falls back to question marks",
			input:    "Should I proceed with the changes?",
			expected: "Should I proceed with the changes?",
		},
		{
			name:     "no question found",
			input:    "This is just a statement about things.",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Extract(tc.input)
			if result != tc.expected {
				t.Errorf("Extract() = %q, want %q", result, tc.expected)
			}
		})
	}
}
