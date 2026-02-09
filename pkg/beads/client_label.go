package beads

import (
	"fmt"
)

// AddLabel adds a label to an issue.
func (c *Client) AddLabel(id, label string) error {
	return c.AddLabels(id, label)
}

// AddLabels adds one or more labels to an issue.
func (c *Client) AddLabels(id string, labels ...string) error {
	if len(labels) == 0 {
		return nil
	}

	_, err := c.Update(&UpdateArgs{ID: id, AddLabels: labels})
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
	return FallbackAddLabels(id, label)
}

// FallbackAddLabels adds one or more labels to an issue via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackAddLabels(id string, labels ...string) error {
	if len(labels) == 0 {
		return nil
	}

	args := []string{"update", id}
	for _, label := range labels {
		args = append(args, "--add-label", label)
	}

	output, err := runBDCombinedOutput(DefaultDir, args...)
	if err != nil {
		if IsCLITimeout(err) {
			return fmt.Errorf("bd add-label(s) timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd add-label(s) failed: %w: %s", err, string(output))
	}
	return nil
}
