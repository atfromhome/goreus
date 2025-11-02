package mail

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// NullMailer adalah implementasi Mailer yang tidak benar-benar mengirim email.
// Ini hanya mencetak isi email ke stdout untuk tujuan testing/logging.
type NullMailer struct{}

// NewNullMailer membuat instance NullMailer.
func NewNullMailer() Mailer {
	return &NullMailer{}
}

// Send "mengirim" email dengan cara mencetak ke stdout.
// Mendukung headers dan attachments untuk verifikasi dalam pengujian.
func (n *NullMailer) Send(ctx context.Context, msg *Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if msg == nil {
		return fmt.Errorf("null mailer: nil message")
	}

	ts := time.Now().Format(time.RFC3339)
	var b strings.Builder
	border := strings.Repeat("-", 72)

	fmt.Fprintf(&b, "%s\n", border)
	fmt.Fprintf(&b, "[NULL MAILER] %s\n", ts)
	fmt.Fprintf(&b, "From: %s\n", strings.TrimSpace(msg.From))
	fmt.Fprintf(&b, "To: %s\n", strings.Join(NormalizeEmails(msg.To), ", "))
	if len(msg.Cc) > 0 {
		fmt.Fprintf(&b, "Cc: %s\n", strings.Join(NormalizeEmails(msg.Cc), ", "))
	}
	if len(msg.Bcc) > 0 {
		fmt.Fprintf(&b, "Bcc: %s\n", strings.Join(NormalizeEmails(msg.Bcc), ", "))
	}
	if strings.TrimSpace(msg.Subject) != "" {
		fmt.Fprintf(&b, "Subject: %s\n", msg.Subject)
	}
	if len(msg.ReplyTo) > 0 {
		fmt.Fprintf(&b, "Reply-To: %s\n", strings.Join(NormalizeEmails(msg.ReplyTo), ", "))
	}
	if strings.TrimSpace(msg.MessageID) != "" {
		fmt.Fprintf(&b, "Message-ID: %s\n", strings.TrimSpace(msg.MessageID))
	}

	// Tags
	if len(msg.Tags) > 0 {
		keys := make([]string, 0, len(msg.Tags))
		for k := range msg.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "Tags:\n")
		for _, k := range keys {
			v := msg.Tags[k]
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
	}

	// Headers
	if len(msg.Headers) > 0 {
		keys := make([]string, 0, len(msg.Headers))
		for k := range msg.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "Headers:\n")
		for _, k := range keys {
			v := msg.Headers[k]
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}

	// Body
	hasText := strings.TrimSpace(msg.PlainText) != ""
	hasHTML := strings.TrimSpace(msg.HTML) != ""
	fmt.Fprintf(&b, "Body:\n")
	if hasText {
		fmt.Fprintf(&b, "  [text/plain]\n")
		WriteIndented(&b, msg.PlainText, 2)
	}
	if hasHTML {
		if hasText {
			fmt.Fprintf(&b, "  ---\n")
		}
		fmt.Fprintf(&b, "  [text/html]\n")
		WriteIndented(&b, msg.HTML, 2)
	}
	if !hasText && !hasHTML {
		fmt.Fprintf(&b, "  <empty>\n")
	}

	// Attachments
	if len(msg.Attachments) > 0 {
		fmt.Fprintf(&b, "Attachments (%d):\n", len(msg.Attachments))
		for i, a := range msg.Attachments {
			name := strings.TrimSpace(a.Filename)
			if name == "" {
				name = "attachment"
			}
			ctype := strings.TrimSpace(a.ContentType)
			if ctype == "" {
				ctype = "application/octet-stream"
			}
			fmt.Fprintf(&b, "  %d) name=%q type=%q size=%d bytes\n", i+1, name, ctype, len(a.Data))
		}
	}

	fmt.Fprintf(&b, "%s\n", border)
	// Cetak ke stdout
	fmt.Print(b.String())

	return nil
}
