package userconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DocDebt tracks documentation debt for CLI commands.
// Stored in ~/.orch/doc-debt.json.
type DocDebt struct {
	// Commands maps command file name to its debt entry.
	Commands map[string]DocDebtEntry `json:"commands"`
	// LastUpdated is when the doc debt file was last modified.
	LastUpdated string `json:"last_updated"`
}

// DocDebtEntry represents a single CLI command's documentation status.
type DocDebtEntry struct {
	// CommandFile is the file name (e.g., "reconcile.go").
	CommandFile string `json:"command_file"`
	// DateAdded is when the command was first detected (YYYY-MM-DD).
	DateAdded string `json:"date_added"`
	// Documented indicates if the command has been documented.
	Documented bool `json:"documented"`
	// DateDocumented is when the command was marked as documented (YYYY-MM-DD).
	DateDocumented string `json:"date_documented,omitempty"`
	// DocLocations lists where documentation should exist.
	DocLocations []string `json:"doc_locations,omitempty"`
}

// DocDebtPath returns the path to the doc debt file.
func DocDebtPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "doc-debt.json")
}

// LoadDocDebt loads the doc debt file from ~/.orch/doc-debt.json.
// Returns an empty DocDebt if the file doesn't exist.
func LoadDocDebt() (*DocDebt, error) {
	data, err := os.ReadFile(DocDebtPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &DocDebt{
				Commands: make(map[string]DocDebtEntry),
			}, nil
		}
		return nil, err
	}

	var debt DocDebt
	if err := json.Unmarshal(data, &debt); err != nil {
		return nil, err
	}

	if debt.Commands == nil {
		debt.Commands = make(map[string]DocDebtEntry)
	}

	return &debt, nil
}

// SaveDocDebt saves the doc debt to ~/.orch/doc-debt.json.
func SaveDocDebt(debt *DocDebt) error {
	// Update timestamp
	debt.LastUpdated = time.Now().Format("2006-01-02T15:04:05")

	data, err := json.MarshalIndent(debt, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(DocDebtPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(DocDebtPath(), data, 0644)
}

// AddCommand adds a new command to the doc debt tracker.
// Returns true if the command was newly added, false if it already exists.
func (d *DocDebt) AddCommand(fileName string) bool {
	if _, exists := d.Commands[fileName]; exists {
		return false
	}

	d.Commands[fileName] = DocDebtEntry{
		CommandFile: fileName,
		DateAdded:   time.Now().Format("2006-01-02"),
		Documented:  false,
		DocLocations: []string{
			"~/.claude/skills/meta/orchestrator/SKILL.md",
			"docs/orch-commands-reference.md",
		},
	}
	return true
}

// MarkDocumented marks a command as documented.
func (d *DocDebt) MarkDocumented(fileName string) bool {
	entry, exists := d.Commands[fileName]
	if !exists {
		return false
	}

	entry.Documented = true
	entry.DateDocumented = time.Now().Format("2006-01-02")
	d.Commands[fileName] = entry
	return true
}

// UndocumentedCommands returns all commands that are not yet documented.
func (d *DocDebt) UndocumentedCommands() []DocDebtEntry {
	var result []DocDebtEntry
	for _, entry := range d.Commands {
		if !entry.Documented {
			result = append(result, entry)
		}
	}
	return result
}
