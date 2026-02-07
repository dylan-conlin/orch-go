package beads

import (
	"fmt"
)

// AddLabel adds a label to an issue.
func (c *Client) AddLabel(id, label string) error {
	args := LabelAddArgs{
		ID:    id,
		Label: label,
	}

	_, err := c.execute(OpLabelAdd, args)
	return err
}

// RemoveLabel removes a label from an issue.
func (c *Client) RemoveLabel(id, label string) error {
	args := LabelRemoveArgs{
		ID:    id,
		Label: label,
	}

	_, err := c.execute(OpLabelRemove, args)
	return err
}

// FallbackRemoveLabel removes a label from an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackRemoveLabel(id, label string) error {
	output, err := runBDCombinedOutput(DefaultDir, "update", id, "--remove-label", label)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd remove-label timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd remove-label failed: %w: %s", err, string(output))
	}
	return nil
}

// FallbackAddLabel adds a label to an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackAddLabel(id, label string) error {
	output, err := runBDCombinedOutput(DefaultDir, "update", id, "--add-label", label)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd add-label timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd add-label failed: %w: %s", err, string(output))
	}
	return nil
}
