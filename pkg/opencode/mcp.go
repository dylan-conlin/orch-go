package opencode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MCPServerStatus represents the status of a single MCP server.
type MCPServerStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// MCPStatus fetches the status of all MCP servers from the OpenCode API.
func (c *Client) MCPStatus() (map[string]MCPServerStatus, error) {
	resp, err := c.httpClient.Get(c.ServerURL + "/mcp")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MCP status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	var status map[string]MCPServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode MCP status: %w", err)
	}
	return status, nil
}

// MCPConnect connects (or reconnects) an MCP server by name.
func (c *Client) MCPConnect(name string) error {
	req, err := http.NewRequest("POST", c.ServerURL+"/mcp/"+name+"/connect", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect MCP server %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to connect MCP server %s: status %d: %s", name, resp.StatusCode, string(body))
	}
	return nil
}

// MCPDisconnect disconnects an MCP server by name.
func (c *Client) MCPDisconnect(name string) error {
	req, err := http.NewRequest("POST", c.ServerURL+"/mcp/"+name+"/disconnect", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to disconnect MCP server %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to disconnect MCP server %s: status %d: %s", name, resp.StatusCode, string(body))
	}
	return nil
}
