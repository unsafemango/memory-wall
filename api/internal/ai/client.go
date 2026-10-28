package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const model = "claude-opus-5-5"

// ErrDeclined means the model declined to write suggestions for this request.
var ErrDeclined = errors.New("model declined the request")

// Client drafts celebration messages with Claude. The API key stays on the
// server, the browser only ever talks to our own endpoint.
type Client struct {
	anthropic anthropic.Client
}

func NewClient(apiKey string) *Client {
	return &Client{anthropic: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

var toneDescriptions = map[string]string{
	"heartfelt": "warm and heartfelt",
	"funny":     "light and funny",
	"short":     "short and sweet",
}

// SuggestMessages returns up to 3 short drafts from sender to recipient.
// detail is an optional thing to work in, like a shared memory or inside joke.
func (c *Client) SuggestMessages(ctx context.Context, recipientName, senderName, tone, detail string) ([]string, error) {
	toneText, ok := toneDescriptions[tone]
	if !ok {
		toneText = toneDescriptions["heartfelt"]
	}
	if recipientName == "" {
		recipientName = "the person being celebrated"
	}
	if senderName == "" {
		senderName = "a friend"
	}

	prompt := fmt.Sprintf(
		"Write 3 different short messages (1-2 sentences each, %s) from %s to %s, to be left on a memory wall for a celebration.",
		toneText, senderName, recipientName,
	)
	if detail != "" {
		prompt += fmt.Sprintf(" Work in this detail if it fits naturally: %s.", detail)
	}
	prompt += " Return only the 3 messages, separated by a line containing only ---, with no numbering or other text."

	resp, err := c.anthropic.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     model,
		MaxTokens: 4000,
		// Drafting a few sentences doesn't need deep reasoning.
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		// If the request is declined, let the API retry it on a fallback model.
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return nil, ErrDeclined
	}

	var text strings.Builder
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(b.Text)
		}
	}

	var suggestions []string
	for _, part := range strings.Split(text.String(), "---") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			suggestions = append(suggestions, trimmed)
		}
	}
	if len(suggestions) == 0 {
		return nil, errors.New("empty response from model")
	}
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions, nil
}
