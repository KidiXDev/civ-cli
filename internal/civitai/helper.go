package civitai

// Helper to inject late API key setting
func (c *Client) SetAuthToken(token string) {
	c.http.SetAuthToken(token)
}

// Ensure the real resty client handles it natively.
