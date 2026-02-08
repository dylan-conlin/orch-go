package beads

// WithConnected initializes a client for projectDir, connects to the daemon,
// and closes the connection after fn returns.
func WithConnected(projectDir string, fn func(*Client) error, opts ...Option) error {
	if fn == nil {
		return nil
	}

	return Do(projectDir, func(client *Client) error {
		if err := client.Connect(); err != nil {
			return err
		}
		defer client.Close()
		return fn(client)
	}, opts...)
}

// WithFallback runs fn against the RPC client and falls back when RPC is unavailable.
// Fallback runs for any RPC path error (socket discovery, connect, or operation error).
func WithFallback(projectDir string, fn func(*Client) error, fallback func() error, opts ...Option) error {
	err := WithConnected(projectDir, fn, opts...)
	if err == nil {
		return nil
	}
	if fallback == nil {
		return err
	}
	return fallback()
}
