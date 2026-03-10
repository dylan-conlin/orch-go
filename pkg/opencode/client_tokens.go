package opencode

// TokenStats represents aggregated token usage for a session.
type TokenStats struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens"` // input + output + reasoning
}

// AggregateTokens calculates total token usage from a slice of messages.
// It sums up input, output, reasoning, and cache tokens across all messages.
func AggregateTokens(messages []Message) TokenStats {
	var stats TokenStats
	for _, msg := range messages {
		if msg.Info.Tokens == nil {
			continue
		}
		stats.InputTokens += msg.Info.Tokens.Input
		stats.OutputTokens += msg.Info.Tokens.Output
		stats.ReasoningTokens += msg.Info.Tokens.Reasoning
		if msg.Info.Tokens.Cache != nil {
			stats.CacheReadTokens += msg.Info.Tokens.Cache.Read
		}
	}
	stats.TotalTokens = stats.InputTokens + stats.OutputTokens + stats.ReasoningTokens
	return stats
}

// GetSessionTokens fetches messages for a session and returns aggregated token stats.
// Returns nil if session doesn't exist or has no messages.
func (c *Client) GetSessionTokens(sessionID string) (*TokenStats, error) {
	messages, err := c.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	stats := AggregateTokens(messages)
	return &stats, nil
}
