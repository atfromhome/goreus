# pkg/mail

Mesin pengiriman email sederhana dan minimal untuk Ziswapp dengan satu tanggung jawab:
- Mengirim email dengan body text/html, text/plain, atau keduanya (multipart/alternative).
- Mengirim lampiran (attachments) dengan multipart/mixed.
- Menyisipkan header kustom untuk kebutuhan tracking/analytics.

Prioritas transport:
- SMTP (STARTTLS/port 587, implicit TLS/port 465)
- SES (AWS Simple Email Service)
- Null/Log (untuk pengujian)

Semua pengiriman mengikuti kontrak yang kecil dan konsisten sehingga mudah diganti transport-nya tanpa mengubah kode pemanggil.

---

## Fitur

- API sangat kecil: `Send(ctx, *Message) error`
- Factory: pilih transport via environment atau secara programatik
- Implementasi SMTP dan Null/Log untuk pengujian
- Body:
  - text/plain dan/atau text/html
  - multipart/alternative bila keduanya ada
- Attachment:
  - multipart/mixed
  - base64 encoding, nama file aman (disanitasi)
- Custom headers:
  - Header arbitrary (mis. `X-Tracking-ID`) disisipkan setelah sanitasi CR/LF
- Reply-To, Message-ID, dan email tags (SES) untuk tracking/analytics lanjutan
- Konteks:
  - `Send` menghormati context (timeout/cancel), termasuk saat dial dan TLS handshake
- Tidak ada business logic; murni lapisan integrasi

---

## Instalasi dan dependensi

Paket ini berada di dalam repository dan menggunakan standar library (SMTP via `net/smtp`).

Jika Anda baru menambahkan paket ini di aplikasi Anda, pastikan dependensi tersinkron (mis. jalankan `go mod tidy` di proyek Anda).

---

## Variabel environment

Pemilihan transport
- `MAIL_TRANSPORT`: `smtp` | `ses` | `null` | `log` | `mock` (default: `null`)

Pengirim default
- `MAIL_FROM_ADDRESS`: alamat email default pengirim
- `MAIL_FROM_NAME`: nama pengirim default (opsional)

SMTP
- `SMTP_HOST`: host server SMTP (wajib untuk `MAIL_TRANSPORT=smtp`)
- `SMTP_PORT`: port server SMTP (mis. `587` atau `465`)
- `SMTP_USERNAME`: username SMTP (opsional, tergantung server)
- `SMTP_PASSWORD`: password SMTP
- `SMTP_FROM`: alamat email default pengirim (fallback; gunakan `MAIL_FROM_ADDRESS`/`MAIL_FROM_NAME` sebagai sumber utama)

Contoh .env
```
# Transport email
MAIL_TRANSPORT=smtp

# Pengirim default (disarankan)
MAIL_FROM_ADDRESS=no-reply@ziswapp.org
MAIL_FROM_NAME=Ziswapp

# Konfigurasi SMTP
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=secret
# SMTP_FROM=no-reply@ziswapp.org   # fallback (opsional)

# AWS SES (alternatif transport)
# MAIL_TRANSPORT=ses
#
# AWS_REGION=ap-southeast-1
# AWS_ACCESS_KEY_ID=AKIA...
# AWS_SECRET_ACCESS_KEY=...
#
# SES_CONFIGURATION_SET=optional-config-set
# SES_FROM=no-reply@ziswapp.org    # fallback (opsional)

# Pengujian (null/log)
# MAIL_TRANSPORT=null
```

---

## API

Konstruktor
- `NewTransportFromEnv(ctx) (Mailer, error)`
- `NewTransport(ctx, cfg *Config) (Mailer, error)`
- `NewNullMailer() Mailer`
- `NewSMTPMailer(host string, port int, username, password, defaultFrom string) (Mailer, error)`
- `NewSESMailer(region, accessKeyID, secretAccessKey, defaultFrom, configurationSet string) (Mailer, error)`

Antarmuka
- `Send(ctx context.Context, msg *Message) error`

Tipe
- `Message`:
  - `From string`
  - `To []string`
  - `Cc []string`
  - `Bcc []string`
  - `Subject string`
  - `PlainText string`
  - `HTML string`
  - `ReplyTo []string`
  - `MessageID string`
  - `Headers map[string]string`
  - `Tags map[string]string`
  - `Attachments []Attachment`
- `Attachment`:
  - `Filename string`
  - `ContentType string` (default: `application/octet-stream` bila kosong)
  - `Data []byte`

Catatan internal (opsional)
- `BuildMime(from string, msg *Message) ([]byte, error)` — membangun raw MIME (headers + body) yang reusable lintas transport.
- Berbagai helper tersedia (sanitasi header/nama file, boundary generator, encoder).

---

## Quick start

Inisialisasi dari environment
```go
ctx := context.Background()

mailer, err := mail.NewTransportFromEnv(ctx)
if err != nil {
    // tangani error konfigurasi (mis. SMTP_HOST/PORT kosong)
}
```

Kirim email dasar
```go
msg := &mail.Message{
    // From boleh kosong → fallback ke SMTP_FROM (untuk transport SMTP)
    To:        []string{"penerima@example.com"},
    Subject:   "Halo dari Ziswapp",
    PlainText: "Hai, ini body versi teks.",
    HTML:      "<h1>Hai</h1><p>Ini body versi <b>HTML</b>.</p>",
}

if err := mailer.Send(ctx, msg); err != nil {
    // tangani error pengiriman
}
```

Kirim email dengan attachment
```go
pdf := []byte("%PDF-1.7 ...") // contoh data PDF

msg := &mail.Message{
    To:        []string{"penerima@example.com"},
    Subject:   "Invoice Bulan Ini",
    PlainText: "Silakan lihat lampiran.",
    HTML:      "<p>Silakan lihat lampiran.</p>",
    Attachments: []mail.Attachment{
        {
            Filename:    "invoice.pdf",
            ContentType: "application/pdf",
            Data:        pdf,
        },
    },
}

if err := mailer.Send(ctx, msg); err != nil {
    // tangani error
}
```

Menambahkan custom headers (tracking/analytics)
```go
msg := &mail.Message{
    To:        []string{"penerima@example.com"},
    Subject:   "Kampanye Welcome",
    PlainText: "Selamat datang!",
    Headers: map[string]string{
        "X-Tracking-ID": "abc-123",
        "X-Campaign":    "welcome-2025",
    },
}

_ = mailer.Send(ctx, msg)
```

Konfigurasi programatik
```go
cfg := &mail.Config{
    Transport: "smtp", // atau "null"/"log"/"mock"
}
cfg.SMTP.Host     = "smtp.example.com"
cfg.SMTP.Port     = 587
cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
cfg.SMTP.From     = "no-reply@ziswapp.org"

mailer, err := mail.NewTransport(context.Background(), cfg)
if err != nil {
    // tangani error
}
```

Praktik terbaik
- Selalu gunakan `context.WithTimeout` untuk membatasi durasi pengiriman.
- Isi `SMTP_FROM` dan biarkan `Message.From` kosong bila Anda ingin default konsisten.
- Isi `PlainText` untuk kompatibilitas klien email yang tidak menampilkan HTML.
- Gunakan header kustom untuk trace id kampanye atau audit non-PII.
- Jangan menyertakan data sensitif dalam headers/body/log.

---

## Contoh Penggunaan

Contoh lebih komprehensif yang mencakup:
- Inisialisasi via ENV dengan timeout.
- Pengisian To, Cc, Bcc.
- Body gabungan text/plain + text/html.
- Header kustom untuk tracking.
- Beberapa lampiran (PDF dan CSV).
- Preview raw MIME (opsional; untuk debugging/dev).
- Fallback ke transport null untuk pengujian lokal.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/ziswapp/ziswapp/pkg/mail"
)

func main() {
    // (Opsional) Set ENV secara programatik saat demo/pengujian
    os.Setenv("MAIL_TRANSPORT", "smtp")
    os.Setenv("SMTP_HOST", "smtp.example.com")
    os.Setenv("SMTP_PORT", "587")
    os.Setenv("SMTP_USERNAME", "apikey")
    os.Setenv("SMTP_PASSWORD", "secret")
    os.Setenv("SMTP_FROM", "no-reply@ziswapp.org")

    // Konteks dengan timeout (disarankan)
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    // Inisialisasi mailer dari ENV
    mailer, err := mail.NewTransportFromEnv(ctx)
    if err != nil {
        log.Fatalf("mail init: %v", err)
    }

    // Siapkan lampiran contoh
    pdf := []byte("%PDF-1.7 ...binary content...")
    csv := []byte("name,email\nAlice,alice@example.com\nBob,bob@example.com\n")

    // Bangun pesan
    msg := &mail.Message{
        // From kosong → fallback ke SMTP_FROM
        To:      []string{"to1@example.com", "to2@example.com"},
        Cc:      []string{"cc@example.com"},
        Bcc:     []string{"internal-audit@example.com"},
        Subject: "Welcome to Ziswapp",
        PlainText: "Halo,\n\n" +
            "Selamat datang di Ziswapp.\n" +
            "Silakan lihat lampiran untuk informasi lebih lanjut.\n\n" +
            "Terima kasih.",
        HTML: `<h1>Halo</h1>
<p>Selamat datang di <b>Ziswapp</b>.</p>
<p>Silakan lihat <i>lampiran</i> untuk informasi lebih lanjut.</p>
<p>Terima kasih.</p>`,
        ReplyTo: []string{"support@ziswapp.org"},
        MessageID: "welcome-2025-0001@ziswapp.org",
        Headers: map[string]string{
            "X-Tracking-ID": "onboarding-2025-0001",
            "X-Campaign":    "onboarding-2025",
        },
        Tags: map[string]string{
            "campaign": "onboarding-2025",
            "segment":  "early-access",
        },
        Attachments: []mail.Attachment{
            {
                Filename:    "welcome.pdf",
                ContentType: "application/pdf",
                Data:        pdf,
            },
            {
                Filename:    "contacts.csv",
                ContentType: "text/csv",
                Data:        csv,
            },
        },
    }

    // (Opsional) Preview raw MIME untuk debugging (jangan dipakai di produksi)
    if raw, err := mail.BuildMime("no-reply@ziswapp.org", msg); err == nil {
        // Cetak sebagian untuk verifikasi
        n := 512
        if len(raw) < n {
            n = len(raw)
        }
        fmt.Printf("[DEBUG MIME PREVIEW]\n%s\n...\n\n", string(raw[:n]))
    } else {
        log.Printf("build mime failed: %v", err)
    }

    // Kirim
    if err := mailer.Send(ctx, msg); err != nil {
        log.Fatalf("send: %v", err)
    }
    log.Println("email terkirim")

    // (Opsional) Fallback ke transport null saat dev/testing
    os.Setenv("MAIL_TRANSPORT", "null")
    nullMailer, _ := mail.NewTransportFromEnv(context.Background())
    _ = nullMailer.Send(context.Background(), &mail.Message{
        To:        []string{"dev@example.com"},
        Subject:   "Test (Null Transport)",
        PlainText: "Ini hanya log, tidak terkirim.",
    })
}
```

Tips:
- Pastikan `MAIL_FROM_ADDRESS` valid dan domain telah dikonfigurasi SPF/DKIM/DMARC agar deliverability baik.
- Hindari melampirkan file besar; pertimbangkan tautan unduhan aman.

---

## Rincian transport dan catatan

SMTP
- STARTTLS vs implicit TLS:
  - Port 465: implicit TLS (TLS handshake sebelum perintah SMTP)
  - Port 587/25: plaintext TCP kemudian upgrade STARTTLS bila server mendukung
- Autentikasi:
  - Jika `SMTP_USERNAME` diset, akan mencoba AUTH PLAIN bila server mendukung
- Envelope & Header:
  - RCPT TO mencakup To, Cc, dan Bcc
  - Bcc tidak dituliskan ke header (sesuai standar)
- Subject & karakter non-ASCII:
  - Subject di-encode MIME Q (utf-8) secara otomatis
- Reply-To dan Message-ID:
  - Didukung via header MIME pada semua transport (SMTP, SES, Null/Log). Set melalui field `ReplyTo` dan `MessageID` pada `Message`.
- Lampiran:
  - base64 dengan line wrap 76 karakter (sesuai rekomendasi MIME)
  - overhead ~33% dari ukuran asli
- Kinerja & keandalan:
  - `Send` menghormati context; gunakan timeout yang realistis
  - Tangani error spesifik (auth gagal, STARTTLS gagal, RCPT gagal)

SES
- Menggunakan AWS SDK v2 (SESv2) dengan `SendEmail` konten Raw MIME (dibangun via `BuildMime`)
- Gunakan `AWS_REGION`; pengirim default berasal dari `MAIL_FROM_ADDRESS`/`MAIL_FROM_NAME` (fallback `SES_FROM` bila diisi).
- Kredensial dapat memakai `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` atau default provider chain AWS.
- Mendukung `Destination` (To/Cc/Bcc) dan header kustom di raw MIME
- Mendukung `SES_CONFIGURATION_SET` (opsional) untuk integrasi eventing/tracking SES
- Perhatikan batasan SES (sandbox mode, verified identity/domains, sending limits, region)

Null/Log
- Tidak benar-benar mengirim email; mencetak detail (From/To/Cc/Bcc/Subject/Reply-To/Message-ID/Headers/Tags/Body/Attachments) ke stdout
- Cocok untuk pengembangan/pengujian tanpa dependensi eksternal

---

## Pengujian dengan transport null

Paksa transport null melalui environment
```go
os.Setenv("MAIL_TRANSPORT", "null")

m, err := mail.NewTransportFromEnv(context.Background())
if err != nil {
    t.Fatalf("mail init: %v", err)
}

if err := m.Send(context.Background(), &mail.Message{
    From:      "sender@example.com",
    To:        []string{"recipient@example.com"},
    Subject:   "Test",
    PlainText: "Hello!",
}); err != nil {
    t.Fatalf("send: %v", err)
}
```

Buat langsung NullMailer
```go
m := mail.NewNullMailer()
_ = m.Send(context.Background(), &mail.Message{
    To:        []string{"recipient@example.com"},
    Subject:   "Test",
    PlainText: "Hello!",
})
```

---

## Catatan dan keterbatasan

- Paket ini tidak melakukan DKIM/DMARC signing. Pastikan SPF/DKIM/DMARC domain dikonfigurasi di infrastruktur email Anda agar deliverability baik.
- Header kustom akan disanitasi dari CR/LF untuk mencegah header injection.
- Email yang sangat besar (banyak lampiran) dapat memperlambat pengiriman; pertimbangkan untuk memberi tautan unduhan aman alih-alih melampirkan file besar.
- Validasi alamat email tidak dilakukan secara agresif; lakukan validasi di layer input bila perlu.
- Logging bawaan transport null akan menulis isi email ke stdout—hindari menjalankannya di lingkungan produksi.
