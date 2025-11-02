package mail

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Message mewakili email yang akan dikirim.
type Message struct {
	// From adalah alamat email pengirim.
	From string
	// To adalah daftar alamat email penerima.
	To []string
	// Subject adalah subjek email.
	Subject string
	// HTML adalah isi email dalam format HTML.
	HTML string
	// PlainText adalah isi email dalam format plaintext.
	PlainText string
	// Cc adalah daftar alamat email yang di-copy.
	Cc []string
	// Bcc adalah daftar alamat email yang di-hidden copy.
	Bcc []string
	// ReplyTo adalah daftar alamat email untuk header Reply-To.
	ReplyTo []string
	// MessageID adalah nilai untuk header Message-ID (opsional).
	// Jika kosong, transport dapat membiarkan server/pustaka menetapkan nilainya.
	MessageID string
	// Headers adalah header tambahan untuk tracking/analytic, dsb.
	Headers map[string]string
	// Tags adalah pasangan kunci-nilai untuk email tags (misalnya SES).
	// Implementasi transport dapat memetakan ke fitur tagging masing-masing provider.
	Tags map[string]string
	// Attachments adalah daftar lampiran yang akan dikirim bersama email.
	Attachments []Attachment
}

// Attachment mewakili file lampiran email.
type Attachment struct {
	// Filename adalah nama file yang akan terlihat oleh penerima.
	Filename string
	// ContentType adalah MIME type dari lampiran (mis. "application/pdf").
	ContentType string
	// Data adalah isi file lampiran dalam bentuk byte slice.
	Data []byte
}

// Mailer adalah antarmuka untuk mengirim email.
// Implementasi bertanggung jawab untuk menangani retry, timeout, dan error handling.
type Mailer interface {
	// Send mengirim email dengan informasi yang diberikan dalam pesan.
	// Mengembalikan error jika pengiriman gagal.
	Send(ctx context.Context, msg *Message) error
}

// Config menyimpan konfigurasi mail. Gunakan FromEnv() untuk memuat dari variabel environment.
//
// Transport:
// - "null"  : Pengiriman in-memory/log (untuk pengujian); alias: "log", "mock"
// - "smtp"  : Pengiriman melalui SMTP
// - "ses"   : Pengiriman melalui AWS SES
type Config struct {
	Transport string
	// FromAddress adalah alamat email default pengirim global (meng-override SMTP.From / SES.From).
	FromAddress string
	// FromName adalah nama pengirim default global. Jika diisi, header From akan berbentuk 'Name <address>'.
	FromName string

	SMTP struct {
		// Host adalah host SMTP server.
		Host string
		// Port adalah port SMTP server.
		Port int
		// Username adalah username untuk autentikasi SMTP.
		Username string
		// Password adalah password untuk autentikasi SMTP.
		Password string
		// From adalah alamat email default pengirim.
		From string
	}

	SES struct {
		// Region AWS (mis. ap-southeast-1)
		Region string
		// AccessKeyID untuk autentikasi AWS (opsional jika pakai default provider chain)
		AccessKeyID string
		// SecretAccessKey untuk autentikasi AWS (opsional jika pakai default provider chain)
		SecretAccessKey string
		// From adalah alamat email default pengirim.
		From string
		// ConfigurationSet opsional untuk SES v2
		ConfigurationSet string
	}
}

// FromEnv memuat nilai Config mail dari variabel environment.
//
// Variabel:
// - MAIL_TRANSPORT: null | log | mock | smtp | ses (default: null)
// - MAIL_FROM_ADDRESS, MAIL_FROM_NAME
// - SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM (fallback bila MAIL_FROM_* kosong)
// - AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
// - SES_FROM, SES_CONFIGURATION_SET (SES_FROM fallback bila MAIL_FROM_* kosong)
func FromEnv() *Config {
	cfg := &Config{
		Transport: strings.ToLower(strings.TrimSpace(os.Getenv("MAIL_TRANSPORT"))),
	}

	// Global From (MAIL_FROM_ADDRESS/MAIL_FROM_NAME)
	cfg.FromAddress = strings.TrimSpace(os.Getenv("MAIL_FROM_ADDRESS"))
	cfg.FromName = strings.TrimSpace(os.Getenv("MAIL_FROM_NAME"))
	if cfg.FromAddress == "" {
		// Backward-compatible fallback ke SMTP_FROM/SES_FROM
		if v := strings.TrimSpace(os.Getenv("SMTP_FROM")); v != "" {
			cfg.FromAddress = v
		} else if v := strings.TrimSpace(os.Getenv("SES_FROM")); v != "" {
			cfg.FromAddress = v
		}
	}

	// SMTP
	cfg.SMTP.Host = strings.TrimSpace(os.Getenv("SMTP_HOST"))
	cfg.SMTP.Username = strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	cfg.SMTP.Password = strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	cfg.SMTP.From = strings.TrimSpace(os.Getenv("SMTP_FROM"))

	// AWS/SES
	cfg.SES.Region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	cfg.SES.AccessKeyID = strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))
	cfg.SES.SecretAccessKey = strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	// Legacy fallback untuk env SES_*
	if cfg.SES.Region == "" {
		cfg.SES.Region = strings.TrimSpace(os.Getenv("SES_REGION"))
	}
	if cfg.SES.AccessKeyID == "" {
		cfg.SES.AccessKeyID = strings.TrimSpace(os.Getenv("SES_ACCESS_KEY_ID"))
	}
	if cfg.SES.SecretAccessKey == "" {
		cfg.SES.SecretAccessKey = strings.TrimSpace(os.Getenv("SES_SECRET_ACCESS_KEY"))
	}
	// From khusus SES (legacy); global MAIL_FROM_* diutamakan di factory
	cfg.SES.From = strings.TrimSpace(os.Getenv("SES_FROM"))
	cfg.SES.ConfigurationSet = strings.TrimSpace(os.Getenv("SES_CONFIGURATION_SET"))

	if v := strings.TrimSpace(os.Getenv("SMTP_PORT")); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.SMTP.Port = port
		}
	}

	if cfg.Transport == "" {
		cfg.Transport = "null"
	}

	return cfg
}

// NewTransportFromEnv membuat Mailer dari variabel environment.
func NewTransportFromEnv(ctx context.Context) (Mailer, error) {
	return NewTransport(ctx, FromEnv())
}

// NewTransport membuat Mailer berdasarkan Config yang diberikan.
func NewTransport(ctx context.Context, cfg *Config) (Mailer, error) {
	if cfg == nil {
		return nil, errors.New("mail: missing config")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case "null", "log", "mock", "":
		return NewNullMailer(), nil
	case "smtp":
		// Validasi minimal konfigurasi SMTP
		if strings.TrimSpace(cfg.SMTP.Host) == "" || cfg.SMTP.Port == 0 {
			return nil, errors.New("mail: invalid smtp config: host and port are required")
		}
		// Tentukan default From dengan prioritas: MAIL_FROM_ADDRESS/MAIL_FROM_NAME -> SMTP_FROM
		defaultFrom := ""
		if cfg.FromAddress != "" {
			if cfg.FromName != "" {
				defaultFrom = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromAddress)
			} else {
				defaultFrom = cfg.FromAddress
			}
		} else {
			defaultFrom = cfg.SMTP.From
		}
		return NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, defaultFrom)
	case "ses":
		// Validasi minimal konfigurasi SES
		if strings.TrimSpace(cfg.SES.Region) == "" {
			return nil, errors.New("mail: invalid ses config: region is required")
		}
		// Tentukan default From dengan prioritas: MAIL_FROM_ADDRESS/MAIL_FROM_NAME -> SES_FROM
		defaultFrom := ""
		if cfg.FromAddress != "" {
			if cfg.FromName != "" {
				defaultFrom = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromAddress)
			} else {
				defaultFrom = cfg.FromAddress
			}
		} else {
			defaultFrom = cfg.SES.From
		}
		if strings.TrimSpace(defaultFrom) == "" {
			return nil, errors.New("mail: invalid ses config: from is required")
		}
		return NewSESMailer(cfg.SES.Region, cfg.SES.AccessKeyID, cfg.SES.SecretAccessKey, defaultFrom, cfg.SES.ConfigurationSet)
	default:
		return nil, fmt.Errorf("mail: unsupported transport %q", cfg.Transport)
	}
}
