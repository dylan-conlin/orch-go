package beads

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Comments retrieves comments for an issue.
func (c *Client) Comments(id string) ([]Comment, error) {
	args := CommentListArgs{ID: id}

	resp, err := c.execute(OpCommentList, args)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	if err := json.Unmarshal(resp.Data, &comments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal comments: %w", err)
	}

	return comments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(id, author, text string) error {
	args := CommentAddArgs{
		ID:     id,
		Author: author,
		Text:   text,
	}

	_, err := c.execute(OpCommentAdd, args)
	return err
}

// FallbackComments retrieves comments via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackComments(id string) ([]Comment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "comments", id, "--json")
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bd comments timed out after %v", DefaultCLITimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd comments failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// FallbackAddComment adds a comment via bd CLI.
// Uses DefaultDir if set to ensure cross-project operations work correctly.
// Uses getBdPath() to resolve the bd executable location.
func FallbackAddComment(id, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, getBdPath(), "comments", "add", id, text)
	setupFallbackEnv(cmd)
	if DefaultDir != "" {
		cmd.Dir = DefaultDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("bd comments add timed out after %v", DefaultCLITimeout)
		}
		return fmt.Errorf("bd comments add failed: %w: %s", err, string(output))
	}
	return nil
}
