package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendCallsResend(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	c := New("re_test", "Memory Wall <n@mail.example>", srv.URL)
	err := c.Send(context.Background(), Message{To: "a@example.com", Subject: "Line\nbreak", HTML: "<p>hi</p>", Text: "hi"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if auth != "Bearer re_test" || got["from"] != "Memory Wall <n@mail.example>" || got["subject"] != "Line break" {
		t.Fatalf("unexpected request: auth=%q body=%v", auth, got)
	}
}

func TestSendReportsRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"domain not verified"}`))
	}))
	defer srv.Close()

	err := New("re_test", "n@mail.example", srv.URL).Send(context.Background(), Message{To: "a@example.com"})
	if err == nil || !strings.Contains(err.Error(), "domain not verified") {
		t.Fatalf("got %v", err)
	}
}

func TestNoKeyMeansDisabled(t *testing.T) {
	var c *Client = New("", "n@mail.example", "")
	if c.Enabled() {
		t.Fatal("a client without a key shouldn't be enabled")
	}
}

func TestNewMemoriesWording(t *testing.T) {
	one, err := NewMemories("o@example.com", "Sam's 30th", []NewMemory{{Sender: "Rosa", Message: "Happy birthday!"}}, 1, true, "https://x/m", "https://x/u")
	if err != nil {
		t.Fatal(err)
	}
	if one.Subject != "Rosa left a memory for Sam's 30th" {
		t.Errorf("subject = %q", one.Subject)
	}
	if !strings.Contains(one.Text, "waiting for you to review it") {
		t.Errorf("should say it needs review: %s", one.Text)
	}

	many := []NewMemory{{Sender: "A", Message: "one"}, {Sender: "B", Message: "two"}, {Sender: "C", Photos: 2, Videos: 1}, {Sender: "D", Message: "four"}}
	batch, err := NewMemories("o@example.com", "Sam's 30th", many, 5, false, "https://x/m", "https://x/u")
	if err != nil {
		t.Fatal(err)
	}
	if batch.Subject != "5 new memories for Sam's 30th" {
		t.Errorf("subject = %q", batch.Subject)
	}
	if !strings.Contains(batch.Text, "And 2 more") || strings.Contains(batch.Text, "four") {
		t.Errorf("should preview 3 and sum up the rest: %s", batch.Text)
	}
	if !strings.Contains(batch.Text, "Shared 2 photos and a video") {
		t.Errorf("uncaptioned files needs a description: %s", batch.Text)
	}
}

func TestGuestTextIsEscaped(t *testing.T) {
	m, err := NewMemories("o@example.com", "Party", []NewMemory{{Sender: "<b>x</b>", Message: "<script>alert(1)</script>"}}, 1, false, "https://x/m", "https://x/u")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, "<b>x</b>") {
		t.Fatalf("guest text must be escaped:\n%s", m.HTML)
	}
}
