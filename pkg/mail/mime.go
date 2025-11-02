package mail

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"time"
)

// BuildMime membangun email MIME mentah (headers + body) dari informasi yang diberikan.
// Fungsi ini reusable untuk berbagai transport (SMTP, API provider, dsb).
//
// Aturan:
// - Mengisi header standar: From, To, Cc, Subject (MIME-encoded), Date, MIME-Version
// - Header kustom dari msg.Headers akan disisipkan setelah disanitasi
// - Body mendukung text/plain dan/atau text/html, memakai multipart/alternative jika keduanya ada
// - Attachment didukung via multipart/mixed dengan base64 encoding
// - Bcc tidak dituliskan ke header (hanya digunakan di envelope oleh transport)
func BuildMime(from string, msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("mime: message is nil")
	}

	var buf bytes.Buffer

	// Header standar
	toHeader := strings.Join(NormalizeEmails(msg.To), ", ")
	if toHeader != "" {
		WriteHeader(&buf, "To", toHeader)
	}
	if len(msg.Cc) > 0 {
		WriteHeader(&buf, "Cc", strings.Join(NormalizeEmails(msg.Cc), ", "))
	}
	// Bcc tidak ditulis ke header
	WriteHeader(&buf, "From", from)

	// Reply-To
	if len(msg.ReplyTo) > 0 {
		WriteHeader(&buf, "Reply-To", strings.Join(NormalizeEmails(msg.ReplyTo), ", "))
	}

	// Message-ID (pastikan terbungkus angle brackets sesuai RFC)
	if mid := strings.TrimSpace(msg.MessageID); mid != "" {
		if !strings.HasPrefix(mid, "<") || !strings.HasSuffix(mid, ">") {
			mid = strings.TrimPrefix(mid, "<")
			mid = strings.TrimSuffix(mid, ">")
			mid = "<" + mid + ">"
		}
		WriteHeader(&buf, "Message-ID", mid)
	}

	if subj := strings.TrimSpace(msg.Subject); subj != "" {
		encSubj := mime.QEncoding.Encode("utf-8", subj)
		WriteHeader(&buf, "Subject", encSubj)
	}
	WriteHeader(&buf, "Date", time.Now().Format(time.RFC1123Z))
	WriteHeader(&buf, "MIME-Version", "1.0")

	// Header kustom
	for k, v := range msg.Headers {
		hk := SanitizeHeaderKey(k)
		hv := SanitizeHeaderValue(v)
		if hk == "" || hv == "" {
			continue
		}
		WriteHeader(&buf, hk, hv)
	}

	// Body
	hasHTML := strings.TrimSpace(msg.HTML) != ""
	hasText := strings.TrimSpace(msg.PlainText) != ""
	hasAttachments := len(msg.Attachments) > 0

	if hasAttachments {
		mixedBoundary := GenerateBoundary()
		WriteHeader(&buf, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", mixedBoundary))
		buf.WriteString("\r\n")

		// Part body (alternative atau single)
		buf.WriteString("--" + mixedBoundary + "\r\n")
		if hasHTML && hasText {
			altBoundary := GenerateBoundary()
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n\r\n", altBoundary))

			// text/plain
			buf.WriteString("--" + altBoundary + "\r\n")
			buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			if err := WriteQuotedPrintable(&buf, msg.PlainText); err != nil {
				return nil, err
			}
			buf.WriteString("\r\n")

			// text/html
			buf.WriteString("--" + altBoundary + "\r\n")
			buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			if err := WriteQuotedPrintable(&buf, msg.HTML); err != nil {
				return nil, err
			}
			buf.WriteString("\r\n")

			// end alternative
			buf.WriteString("--" + altBoundary + "--\r\n")
		} else if hasText {
			buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			if err := WriteQuotedPrintable(&buf, msg.PlainText); err != nil {
				return nil, err
			}
			buf.WriteString("\r\n")
		} else if hasHTML {
			buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			if err := WriteQuotedPrintable(&buf, msg.HTML); err != nil {
				return nil, err
			}
			buf.WriteString("\r\n")
		} else {
			// tanpa body, tetap valid
			buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n\r\n")
		}

		// Attachments
		for _, att := range msg.Attachments {
			ct := strings.TrimSpace(att.ContentType)
			if ct == "" {
				ct = "application/octet-stream"
			}
			filename := SanitizeFilename(att.Filename)
			buf.WriteString("--" + mixedBoundary + "\r\n")
			buf.WriteString(fmt.Sprintf("Content-Type: %s; name=%q\r\n", ct, filename))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n\r\n", filename))
			WriteBase64(&buf, att.Data)
			buf.WriteString("\r\n")
		}

		// end mixed
		buf.WriteString("--" + mixedBoundary + "--\r\n")
	} else if hasHTML && hasText {
		altBoundary := GenerateBoundary()
		WriteHeader(&buf, "Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", altBoundary))
		buf.WriteString("\r\n")

		// text/plain
		buf.WriteString("--" + altBoundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := WriteQuotedPrintable(&buf, msg.PlainText); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")

		// text/html
		buf.WriteString("--" + altBoundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := WriteQuotedPrintable(&buf, msg.HTML); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")

		// end alternative
		buf.WriteString("--" + altBoundary + "--\r\n")
	} else if hasText {
		WriteHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		WriteHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		if err := WriteQuotedPrintable(&buf, msg.PlainText); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")
	} else if hasHTML {
		WriteHeader(&buf, "Content-Type", "text/html; charset=utf-8")
		WriteHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		if err := WriteQuotedPrintable(&buf, msg.HTML); err != nil {
			return nil, err
		}
		buf.WriteString("\r\n")
	} else {
		// Empty body
		WriteHeader(&buf, "Content-Type", "text/plain; charset=utf-8")
		buf.WriteString("\r\n\r\n")
	}

	return buf.Bytes(), nil
}
