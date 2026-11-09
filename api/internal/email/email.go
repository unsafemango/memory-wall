// Package email sends email through Resend's HTTP API.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.resend.com"

// Message is one email to one person.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
	Headers map[string]string
}

// Client sends messages. A nil *Client is valid and means email isn't set up,
// so callers can use Enabled to check.
type Client struct {
	apiKey  string
	from    string
	baseURL string
	http    *http.Client
}

// New returns a client, or nil when no API key is configured. baseURL is
// optional and only overridden for testing.
func New(apiKey, from, baseURL string) *Client {
	if apiKey == "" || from == "" {
		return nil
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{apiKey: apiKey, from: from, baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// Enabled reports whether email can be sent.
func (c *Client) Enabled() bool {
	return c != nil
}

// Send delivers a message, returning an error if Resend doesn't accept it.
func (c *Client) Send(ctx context.Context, m Message) error {
	body, err := json.Marshal(map[string]any{
		"from":    c.from,
		"to":      []string{m.To},
		"subject": strings.Join(strings.Fields(m.Subject), " "),
		"html":    m.HTML,
		"text":    m.Text,
		"headers": m.Headers,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, detail)
	}
	return nil
}
