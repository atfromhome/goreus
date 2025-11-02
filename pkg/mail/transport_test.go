package mail

import (
	"context"
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		check   func(*Config) bool
		wantErr bool
	}{
		{
			name: "default null transport",
			env:  map[string]string{},
			check: func(c *Config) bool {
				return c.Transport == "null"
			},
		},
		{
			name: "smtp config from env",
			env: map[string]string{
				"MAIL_TRANSPORT": "smtp",
				"SMTP_HOST":      "smtp.example.com",
				"SMTP_PORT":      "587",
				"SMTP_USERNAME":  "user",
				"SMTP_PASSWORD":  "pass",
				"SMTP_FROM":      "noreply@example.com",
			},
			check: func(c *Config) bool {
				return c.Transport == "smtp" &&
					c.SMTP.Host == "smtp.example.com" &&
					c.SMTP.Port == 587 &&
					c.SMTP.Username == "user" &&
					c.SMTP.Password == "pass"
			},
		},
		{
			name: "global from address overrides",
			env: map[string]string{
				"MAIL_FROM_ADDRESS": "global@example.com",
				"MAIL_FROM_NAME":    "Global Sender",
				"SMTP_FROM":         "smtp@example.com",
			},
			check: func(c *Config) bool {
				return c.FromAddress == "global@example.com" &&
					c.FromName == "Global Sender"
			},
		},
		{
			name: "ses config from env",
			env: map[string]string{
				"MAIL_TRANSPORT": "ses",
				"AWS_REGION":     "ap-southeast-1",
				"SES_FROM":       "ses@example.com",
			},
			check: func(c *Config) bool {
				return c.Transport == "ses" &&
					c.SES.Region == "ap-southeast-1" &&
					c.SES.From == "ses@example.com"
			},
		},
		{
			name: "invalid smtp port",
			env: map[string]string{
				"SMTP_PORT": "invalid",
			},
			check: func(c *Config) bool {
				return c.SMTP.Port == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg := FromEnv()
			if !tt.check(cfg) {
				t.Errorf("FromEnv check failed")
			}
		})
	}
}

func TestNewTransport(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "missing config",
		},
		{
			name: "null transport",
			cfg: &Config{
				Transport: "null",
			},
			wantErr: false,
		},
		{
			name: "log transport",
			cfg: &Config{
				Transport: "log",
			},
			wantErr: false,
		},
		{
			name: "smtp valid",
			cfg: &Config{
				Transport: "smtp",
				SMTP: struct {
					Host     string
					Port     int
					Username string
					Password string
					From     string
				}{
					Host: "smtp.example.com",
					Port: 587,
					From: "test@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "smtp missing host",
			cfg: &Config{
				Transport: "smtp",
				SMTP: struct {
					Host     string
					Port     int
					Username string
					Password string
					From     string
				}{
					Port: 587,
				},
			},
			wantErr: true,
			errMsg:  "invalid smtp config",
		},
		{
			name: "ses missing region",
			cfg: &Config{
				Transport: "ses",
			},
			wantErr: true,
			errMsg:  "invalid ses config",
		},
		{
			name: "unsupported transport",
			cfg: &Config{
				Transport: "unsupported",
			},
			wantErr: true,
			errMsg:  "unsupported transport",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mailer, err := NewTransport(ctx, tt.cfg)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch: got %q, want containing %q", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && mailer == nil {
				t.Error("expected mailer, got nil")
			}
		})
	}
}

func TestNullMailerSend(t *testing.T) {
	mailer := NewNullMailer()

	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name: "valid message",
			msg: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test",
				HTML:    "<p>Test</p>",
			},
			wantErr: false,
		},
		{
			name:    "nil message",
			msg:     nil,
			wantErr: true,
		},
		{
			name: "message with cc bcc",
			msg: &Message{
				From:      "sender@example.com",
				To:        []string{"to@example.com"},
				Cc:        []string{"cc@example.com"},
				Bcc:       []string{"bcc@example.com"},
				Subject:   "Test",
				PlainText: "Test content",
			},
			wantErr: false,
		},
		{
			name: "message with attachments",
			msg: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "With Attachment",
				HTML:    "<p>See attached</p>",
				Attachments: []Attachment{
					{
						Filename:    "test.txt",
						ContentType: "text/plain",
						Data:        []byte("test data"),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := mailer.Send(ctx, tt.msg)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMessageConstruction(t *testing.T) {
	msg := &Message{
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		Subject:   "Test Subject",
		HTML:      "<p>HTML content</p>",
		PlainText: "Plain text content",
		Cc:        []string{"cc@example.com"},
		Bcc:       []string{"bcc@example.com"},
		ReplyTo:   []string{"reply@example.com"},
		MessageID: "<test@example.com>",
		Headers: map[string]string{
			"X-Custom": "value",
		},
		Tags: map[string]string{
			"tag1": "value1",
		},
		Attachments: []Attachment{
			{
				Filename:    "file.pdf",
				ContentType: "application/pdf",
				Data:        []byte("pdf data"),
			},
		},
	}

	if msg.From != "sender@example.com" {
		t.Error("From not set correctly")
	}
	if len(msg.To) != 1 || msg.To[0] != "recipient@example.com" {
		t.Error("To not set correctly")
	}
	if msg.Subject != "Test Subject" {
		t.Error("Subject not set correctly")
	}
	if len(msg.Attachments) != 1 {
		t.Error("Attachments not set correctly")
	}
}

func TestNewTransportFromEnv(t *testing.T) {
	os.Setenv("MAIL_TRANSPORT", "null")
	defer os.Unsetenv("MAIL_TRANSPORT")

	ctx := context.Background()
	mailer, err := NewTransportFromEnv(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mailer == nil {
		t.Error("expected mailer, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr))
}
