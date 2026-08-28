package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	netmail "net/mail"
	"net/smtp"
	"strings"
	"time"
)

// SMTPMailer adalah implementasi Mailer menggunakan protokol SMTP.
type SMTPMailer struct {
	host        string
	port        int
	username    string
	password    string
	defaultFrom string

	tlsConfig   *tls.Config
	dialTimeout time.Duration
}

// NewSMTPMailer membuat Mailer SMTP baru.
func NewSMTPMailer(host string, port int, username, password, defaultFrom string) (Mailer, error) {
	if strings.TrimSpace(host) == "" || port <= 0 {
		return nil, fmt.Errorf("smtp: invalid host/port")
	}

	return &SMTPMailer{
		host:        strings.TrimSpace(host),
		port:        port,
		username:    strings.TrimSpace(username),
		password:    password,
		defaultFrom: strings.TrimSpace(defaultFrom),
		tlsConfig:   &tls.Config{ServerName: strings.TrimSpace(host)},
		dialTimeout: 10 * time.Second,
	}, nil
}

// Send mengirim email menggunakan SMTP.
func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("smtp: message is nil")
	}

	from := strings.TrimSpace(msg.From)
	if from == "" {
		from = m.defaultFrom
	}
	if from == "" {
		return fmt.Errorf("smtp: from is empty")
	}

	// envelopeFrom adalah ALAMAT MURNI untuk perintah SMTP MAIL FROM, terpisah
	// dari `from` yang dipakai BuildMime untuk header MIME From (boleh berformat
	// "Nama <alamat>"). net/smtp.Client.Mail mengirim argumennya apa adanya di
	// dalam "MAIL FROM:<...>", jadi memberinya string berformat nama menghasilkan
	// kurung siku bersarang -- perintah SMTP yang tidak valid.
	//
	// Kegagalan parse dibiarkan lewat dengan envelopeFrom = from, sama seperti
	// perilaku sebelum perubahan ini: masukan yang sudah rusak hari ini tidak
	// mendadak berhenti terkirim gara-gara ParseAddress lebih ketat.
	envelopeFrom := from
	if addr, err := netmail.ParseAddress(from); err == nil {
		envelopeFrom = addr.Address
	}

	// Kumpulkan seluruh penerima (envelope) termasuk Cc dan Bcc
	var rcpts []string
	rcpts = append(rcpts, NormalizeEmails(msg.To)...)
	rcpts = append(rcpts, NormalizeEmails(msg.Cc)...)
	rcpts = append(rcpts, NormalizeEmails(msg.Bcc)...)
	if len(rcpts) == 0 {
		return fmt.Errorf("smtp: no recipients")
	}

	// Bangun MIME message
	raw, err := BuildMime(from, msg)
	if err != nil {
		return err
	}

	// Dial dengan context
	d := &net.Dialer{Timeout: m.dialTimeout}
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var conn net.Conn
	dialErrCh := make(chan error, 1)
	doneDial := make(chan struct{})
	go func() {
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			dialErrCh <- err
			return
		}
		conn = c
		close(doneDial)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp: dial canceled: %w", ctx.Err())
	case err := <-dialErrCh:
		return fmt.Errorf("smtp: dial error: %w", err)
	case <-doneDial:
	}
	defer conn.Close()

	// Gunakan implicit TLS jika port 465; selain itu gunakan plain + STARTTLS bila tersedia
	var rawConn net.Conn = conn
	if m.port == 465 {
		tlsConn := tls.Client(conn, m.tlsConfig)
		// Gunakan deadline dari context saat handshake jika ada
		if dl, ok := ctx.Deadline(); ok {
			_ = tlsConn.SetDeadline(dl)
		}
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("smtp: tls handshake: %w", err)
		}
		_ = tlsConn.SetDeadline(time.Time{})
		rawConn = tlsConn
	}

	client, err := smtp.NewClient(rawConn, m.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	// STARTTLS jika tersedia (untuk koneksi non-implicit TLS)
	if m.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(m.tlsConfig); err != nil {
				return fmt.Errorf("smtp: starttls: %w", err)
			}
		}
	}

	// AUTH jika tersedia
	if m.username != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			auth := smtp.PlainAuth("", m.username, m.password, m.host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp: auth failed: %w", err)
			}
		}
	}

	// Envelope
	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	for _, rcpt := range rcpts {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: RCPT TO %s: %w", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := wc.Write(raw); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp: write data: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}

	return nil
}
