package mail

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
)

// WriteHeader menuliskan header email single-line dengan CRLF.
// Abaikan jika key atau value kosong.
func WriteHeader(w io.Writer, key, value string) {
	if key == "" || value == "" {
		return
	}
	_, _ = io.WriteString(w, fmt.Sprintf("%s: %s\r\n", key, value))
}

// WriteQuotedPrintable menuliskan string menggunakan encoding quoted-printable
// ke writer yang diberikan, lalu menutup encoder.
func WriteQuotedPrintable(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := io.WriteString(qp, s); err != nil {
		_ = qp.Close()
		return err
	}
	return qp.Close()
}

// WriteBase64 menuliskan data dalam encoding base64 dengan line wrap
// 76 karakter per baris sesuai rekomendasi MIME.
func WriteBase64(w io.Writer, data []byte) {
	if len(data) == 0 {
		return
	}
	enc := base64.StdEncoding.EncodeToString(data)
	for len(enc) > 0 {
		n := 76
		if len(enc) < n {
			n = len(enc)
		}
		_, _ = io.WriteString(w, enc[:n])
		_, _ = io.WriteString(w, "\r\n")
		enc = enc[n:]
	}
}

// GenerateBoundary membuat boundary acak untuk multipart MIME.
func GenerateBoundary() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("----ZISWAPP-%x", b[:])
}

// SanitizeHeaderKey membersihkan key header dari karakter terlarang dan spasi.
func SanitizeHeaderKey(k string) string {
	k = strings.TrimSpace(k)
	k = strings.ReplaceAll(k, "\r", "")
	k = strings.ReplaceAll(k, "\n", "")
	k = strings.TrimSpace(k)
	return k
}

// SanitizeHeaderValue membersihkan value header dari karakter terlarang dan spasi.
func SanitizeHeaderValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.TrimSpace(v)
	return v
}

// SanitizeFilename membersihkan nama file lampiran dari karakter terlarang.
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		return "attachment"
	}
	return name
}

// NormalizeEmails menormalkan daftar email dengan memangkas spasi
// dan menghapus entri kosong.
func NormalizeEmails(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// WriteIndented menuliskan teks dengan indentasi ke builder (untuk logging/presentation).
func WriteIndented(b *strings.Builder, s string, indent int) {
	prefix := strings.Repeat(" ", indent)
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		fmt.Fprintf(b, "%s%s\n", prefix, line)
	}
}
