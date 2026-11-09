package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeAnthropic answers like the Messages API and records the last request.
func fakeAnthropic(t *testing.T, stopReason, text string) *map[string]any {
	t.Helper()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		body, _ := json.Marshal(text)
		fmt.Fprintf(w, `{"id":"msg_1","type":"message","role":"assistant","model":"m","stop_reason":%q,
			"content":[{"type":"text","text":%s}],"usage":{"input_tokens":1,"output_tokens":1}}`, stopReason, body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	return &got
}

func TestSuggestMessagesSendsTheEventAndParsesDrafts(t *testing.T) {
	got := fakeAnthropic(t, "end_turn", "Happy birthday, Sarah!\n---\nThirty looks good on you.\n---\nHere's to you.\n---\nOne too many.")

	drafts, err := NewClient("test-key").SuggestMessages(context.Background(), Request{
		EventTitle: "Sarah's 30th Birthday", Occasion: "birthday", HonoreeName: "Sarah", Tone: "heartfelt",
	})
	if err != nil {
		t.Fatalf("SuggestMessages: %v", err)
	}
	if len(drafts) != 3 || drafts[0] != "Happy birthday, Sarah!" {
		t.Fatalf("drafts = %q", drafts)
	}

	sent, _ := json.Marshal(*got)
	for _, want := range []string{"Sarah's 30th Birthday", "birthday wishes", "event_is_for"} {
		if !strings.Contains(strings.ReplaceAll(string(sent), `\u0027`, "'"), want) {
			t.Errorf("request is missing %q:\n%s", want, sent)
		}
	}
	if (*got)["system"] == nil {
		t.Error("request should have a system prompt")
	}
}

func TestSuggestMessagesReportsARefusal(t *testing.T) {
	fakeAnthropic(t, "refusal", "")
	_, err := NewClient("test-key").SuggestMessages(context.Background(), Request{EventTitle: "x"})
	if err != ErrDeclined {
		t.Fatalf("got %v, want ErrDeclined", err)
	}
}
