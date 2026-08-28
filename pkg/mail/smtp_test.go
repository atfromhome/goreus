package mail

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeSMTPServer adalah server SMTP minimal yang cukup untuk menerima satu
// transaksi (EHLO, MAIL FROM, RCPT TO, DATA, QUIT) dan menangkap baris
// MAIL FROM yang benar-benar dikirim di kabel.
//
// mailFrom diisi lewat channel, bukan variabel dibagi bersama, supaya aman
// dijalankan di bawah -race: goroutine server dan goroutine test tidak pernah
// menyentuh nilai yang sama tanpa sinkronisasi.
type fakeSMTPServer struct {
	listener net.Listener
	mailFrom chan string
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &fakeSMTPServer{listener: ln, mailFrom: make(chan string, 1)}

	go srv.acceptOne(t)

	t.Cleanup(func() { _ = ln.Close() })

	return srv
}

func (s *fakeSMTPServer) acceptOne(t *testing.T) {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	write := func(line string) {
		_, _ = conn.Write([]byte(line + "\r\n"))
	}

	write("220 fake.smtp.test ESMTP")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			write("250 fake.smtp.test")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.mailFrom <- strings.TrimPrefix(line, "MAIL FROM:")
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			write("250 OK")
		case strings.HasPrefix(upper, "DATA"):
			write("354 Start mail input; end with <CRLF>.<CRLF>")
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
			}
			write("250 OK")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (s *fakeSMTPServer) addr() (string, int) {
	host, portStr, _ := net.SplitHostPort(s.listener.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return host, port
}

func TestSMTPMailerEnvelopeFromIsAlwaysABareAddress(t *testing.T) {
	tests := []struct {
		name         string
		from         string
		wantEnvelope string
	}{
		{
			name:         "alamat murni tidak berubah",
			from:         "no-reply@halo.test",
			wantEnvelope: "no-reply@halo.test",
		},
		{
			name:         "alamat berformat nama dipisah untuk envelope",
			from:         "LAZ Uji <no-reply@halo.test>",
			wantEnvelope: "no-reply@halo.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startFakeSMTPServer(t)
			host, port := srv.addr()

			mailer, err := NewSMTPMailer(host, port, "", "", "")
			if err != nil {
				t.Fatalf("NewSMTPMailer: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = mailer.Send(ctx, &Message{
				From:      tt.from,
				To:        []string{"donatur@example.test"},
				Subject:   "Test",
				PlainText: "isi",
			})
			if err != nil {
				t.Fatalf("Send: %v", err)
			}

			select {
			case got := <-srv.mailFrom:
				want := fmt.Sprintf("<%s>", tt.wantEnvelope)
				if got != want {
					t.Errorf("MAIL FROM captured = %q, want %q", got, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("server never received MAIL FROM")
			}

			raw, err := BuildMime(tt.from, &Message{Subject: "Test", PlainText: "isi"})
			if err != nil {
				t.Fatalf("BuildMime: %v", err)
			}
			if !strings.Contains(string(raw), "From: "+tt.from) {
				t.Errorf("header MIME From tidak memuat bentuk lengkap %q, raw:\n%s", tt.from, raw)
			}
		})
	}
}

func TestSMTPMailerFromThatFailsToParseKeepsOldBehavior(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := srv.addr()

	mailer, err := NewSMTPMailer(host, port, "", "", "")
	if err != nil {
		t.Fatalf("NewSMTPMailer: %v", err)
	}

	const brokenFrom = "bukan-alamat-sah"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mailer.Send(ctx, &Message{
		From:      brokenFrom,
		To:        []string{"donatur@example.test"},
		Subject:   "Test",
		PlainText: "isi",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case got := <-srv.mailFrom:
		want := fmt.Sprintf("<%s>", brokenFrom)
		if got != want {
			t.Errorf("MAIL FROM captured = %q, want %q (perilaku sebelum perbaikan: envelope memakai `from` apa adanya saat ParseAddress gagal)",
				got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server never received MAIL FROM")
	}
}

func TestSMTPMailerDefaultFromGoesThroughTheSameSplit(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := srv.addr()

	mailer, err := NewSMTPMailer(host, port, "", "", "LAZ Uji <no-reply@halo.test>")
	if err != nil {
		t.Fatalf("NewSMTPMailer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mailer.Send(ctx, &Message{
		To:        []string{"donatur@example.test"},
		Subject:   "Test",
		PlainText: "isi",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case got := <-srv.mailFrom:
		want := "<no-reply@halo.test>"
		if got != want {
			t.Errorf("MAIL FROM captured = %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server never received MAIL FROM")
	}
}
