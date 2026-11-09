package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// layout wraps every email in the app's look. html/template escapes everything
// passed in, so a guest's message can't inject markup.
var layout = template.Must(template.New("email").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px 12px;background:#fdf1ff;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#3b1d4a">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:20px">
<tr><td style="padding:28px">
<p style="margin:0 0 20px;font-size:15px;font-weight:700;color:#e63888">Memory Wall</p>
<h1 style="margin:0 0 12px;font-size:22px;line-height:1.25;color:#3b1d4a">{{.Heading}}</h1>
{{if .Intro}}<p style="margin:0 0 18px;font-size:15px;line-height:1.55;color:#6b5478">{{.Intro}}</p>{{end}}
{{range .Quotes}}<div style="margin:0 0 12px;padding:12px 14px;background:#f8f0ff;border-radius:12px">
<p style="margin:0 0 4px;font-size:15px;line-height:1.5;white-space:pre-wrap">{{.Body}}</p>
<p style="margin:0;font-size:13px;font-weight:600;color:#e63888">{{.Sender}}</p></div>{{end}}
{{if .More}}<p style="margin:0 0 12px;font-size:14px;color:#6b5478">{{.More}}</p>{{end}}
<p style="margin:22px 0 4px"><a href="{{.ButtonURL}}" style="display:inline-block;padding:12px 22px;border-radius:999px;background:#6d28d9;color:#ffffff;font-weight:600;font-size:15px;text-decoration:none">{{.ButtonText}}</a></p>
{{if .Footer}}<p style="margin:22px 0 0;font-size:12px;line-height:1.5;color:#6b5478">{{.Footer}}{{if .UnsubscribeURL}} <a href="{{.UnsubscribeURL}}" style="color:#6b5478">Turn off these emails</a>{{end}}</p>{{end}}
</td></tr></table>
</td></tr></table>
</body></html>`))

type quote struct{ Body, Sender string }

type emailData struct {
	Heading, Intro, More, ButtonURL, ButtonText, Footer, UnsubscribeURL string
	Quotes                                                              []quote
}

func render(d emailData) (string, error) {
	var b bytes.Buffer
	err := layout.Execute(&b, d)
	return b.String(), err
}

// Confirmation is the email that confirms an address belongs to an organizer.
func Confirmation(to, name, link string) (Message, error) {
	html, err := render(emailData{
		Heading:    "Confirm your email",
		Intro:      fmt.Sprintf("Hi %s, confirm this address so Memory Wall can tell you when a new memory arrives.", name),
		ButtonURL:  link,
		ButtonText: "Confirm my email",
		Footer:     "This link works for 7 days. If you didn't sign up, you can ignore this email.",
	})
	if err != nil {
		return Message{}, err
	}
	return Message{
		To:      to,
		Subject: "Confirm your email for Memory Wall",
		HTML:    html,
		Text: fmt.Sprintf("Hi %s,\n\nConfirm this address so Memory Wall can tell you when a new memory arrives:\n%s\n\n"+
			"This link works for 7 days. If you didn't sign up, you can ignore this email.\n", name, link),
	}, nil
}

// NewMemory describes one memory in a notification.
type NewMemory struct {
	Sender  string
	Message string
	Photos  int
	Videos  int
}

// preview is what a memory looks like in an email. Photos and videos without
// a caption get a short description instead.
func (m NewMemory) preview() string {
	text := strings.TrimSpace(m.Message)
	if text == "" {
		var parts []string
		if m.Photos > 0 {
			parts = append(parts, countOf(m.Photos, "photo"))
		}
		if m.Videos > 0 {
			parts = append(parts, countOf(m.Videos, "video"))
		}
		if len(parts) == 0 {
			return "Left a memory"
		}
		return "Shared " + strings.Join(parts, " and ")
	}
	runes := []rune(text)
	if len(runes) > 200 {
		return strings.TrimSpace(string(runes[:200])) + "…"
	}
	return text
}

// countOf gives "a photo" or "3 photos".
func countOf(n int, word string) string {
	if n == 1 {
		return "a " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// maxPreviews is how many memories one email shows before summing up the rest.
const maxPreviews = 3

// NewMemories tells an organizer about new memories on an event. total can be
// higher than len(memories), which only holds the ones to preview.
func NewMemories(to, eventTitle string, memories []NewMemory, total int, needsReview bool, manageURL, unsubscribeURL string) (Message, error) {
	if len(memories) > maxPreviews {
		memories = memories[:maxPreviews]
	}

	subject := fmt.Sprintf("%d new memories for %s", total, eventTitle)
	heading := fmt.Sprintf("%d new memories", total)
	if total == 1 {
		subject = fmt.Sprintf("%s left a memory for %s", memories[0].Sender, eventTitle)
		heading = fmt.Sprintf("%s left a memory", memories[0].Sender)
	}

	intro := "For " + eventTitle + "."
	button := "See your memories"
	if needsReview {
		intro = "For " + eventTitle + ". It's waiting for you to review it."
		if total > 1 {
			intro = "For " + eventTitle + ". They're waiting for you to review them."
		}
		button = "Review now"
	}

	quotes := make([]quote, len(memories))
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n\n%s\n\n", heading, intro)
	for i, m := range memories {
		quotes[i] = quote{Body: m.preview(), Sender: m.Sender}
		fmt.Fprintf(&text, "\"%s\"\n- %s\n\n", m.preview(), m.Sender)
	}
	more := ""
	if extra := total - len(memories); extra > 0 {
		more = fmt.Sprintf("And %d more.", extra)
		fmt.Fprintf(&text, "%s\n\n", more)
	}
	fmt.Fprintf(&text, "%s: %s\n\nTurn off these emails: %s\n", button, manageURL, unsubscribeURL)

	html, err := render(emailData{
		Heading:        heading,
		Intro:          intro,
		Quotes:         quotes,
		More:           more,
		ButtonURL:      manageURL,
		ButtonText:     button,
		Footer:         "You're getting this because you run this event on Memory Wall.",
		UnsubscribeURL: unsubscribeURL,
	})
	if err != nil {
		return Message{}, err
	}
	return Message{
		To:      to,
		Subject: subject,
		HTML:    html,
		Text:    text.String(),
		Headers: map[string]string{"List-Unsubscribe": "<" + unsubscribeURL + ">"},
	}, nil
}
