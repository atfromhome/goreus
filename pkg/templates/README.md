# pkg/templates

Mesin template sederhana dan minimal untuk Ziswapp dengan satu tanggung jawab:
- Memuat dan cache template HTML dan plaintext dari embed.FS.
- Render template dengan data konteks dan built-in helper functions.
- Track penggunaan template via metrics.

Prioritas:
- Pre-load semua templates saat inisialisasi (fail-fast).
- Support nested layouts via `{{ define }}` blocks.
- Built-in helpers: string, time, math, comparison, slice operations.
- Custom helpers extensible via Config atau `RegisterHelper()`.

---

## Fitur

- API kecil dan konsisten: `New()`, `LoadHTML()`, `LoadPlaintext()`, `RenderHTML()`, `RenderPlaintext()`, `Render()`.
- Pre-load semua templates dari embed.FS saat init (tidak ada lazy-loading per-render).
- Thread-safe dengan `sync.RWMutex` untuk concurrent rendering.
- Auto-detect template tipe (HTML vs plaintext) berdasarkan file extension dan direktori.
- Support nested layouts: load multiple layout files sebelum template utama.
- 26+ built-in template helpers (string, time, math, comparison, slice, type).
- Custom helpers: register via Config atau `RegisterHelper()` setelah init.
- Comprehensive metrics: cache hit/miss, render count, error tracking, hit rate, slow render warnings.
- Structured logging via `slog`.
- Cache clearing dan reloading untuk development.

---

## Instalasi dan dependensi

Paket ini berada di dalam repository dan hanya menggunakan standard library (`text/template`, `embed`, `sync`, `log/slog`).

Jika Anda baru menambahkan paket ini, pastikan dependensi tersinkron (mis. jalankan `go mod tidy`).

---

## Variabel environment

Tidak ada environment variables yang wajib. Konfigurasi dilakukan via `Config` struct saat inisialisasi.

Rekomendasi untuk development:
```
# Jika menggunakan air untuk hot-reload, templates akan reload otomatis
# Tidak ada env var khusus yang diperlukan
```

---

## API

### Konstruktor

- `New(cfg Config) (*Manager, error)` — inisialisasi manager dengan pre-load semua templates.

### Antarmuka Manager

- `LoadHTML(templatePath string, layoutPaths ...string) (*template.Template, error)` — load atau get cached HTML template dengan optional layouts.
- `LoadPlaintext(templatePath string, layoutPaths ...string) (*template.Template, error)` — load atau get cached plaintext template dengan optional layouts.
- `RenderHTML(tmpl *template.Template, templateName string, data any) (string, error)` — render HTML template dengan data.
- `RenderPlaintext(tmpl *template.Template, templateName string, data any) (string, error)` — render plaintext template dengan data.
- `Render(tmpl *template.Template, data any, templateName ...string) (string, error)` — unified render method (backward compat).
- `ListHTML() []string` — list semua available HTML template keys.
- `ListPlaintext() []string` — list semua available plaintext template keys.
- `GetMetrics() map[string]any` — get current metrics snapshot.
- `RegisterHelper(name string, fn any)` — register custom helper function.
- `ClearCache()` — clear semua cached templates.
- `Reload() error` — clear cache dan re-load semua templates dari FS.

### Tipe

```go
// Config untuk inisialisasi Manager
type Config struct {
    FS           embed.FS
    TemplatesDir string
    Logger       *slog.Logger         // optional; default: slog.Default()
    Helpers      template.FuncMap     // optional; custom helpers
}

// Manager mengelola templates dengan caching dan metrics
type Manager struct { ... }

// Metrics melacak penggunaan template
type Metrics struct { ... }
```

### Built-in Helpers

| Category | Helpers |
|----------|---------|
| String | `upper`, `lower`, `title`, `trim`, `join`, `split`, `contains`, `hasPrefix`, `hasSuffix` |
| Time | `now`, `formatTime`, `formatUnix` |
| Math | `add`, `sub`, `mul`, `div`, `mod` |
| Compare | `eq`, `ne`, `gt`, `lt`, `gte`, `lte` |
| Slice | `len`, `first`, `last` |
| Type | `typeof` |

---

## Quick start

### Inisialisasi dari embed.FS

```go
package main

import (
    "embed"
    "context"
    "log/slog"

    "github.com/ziswapp/ziswapp/pkg/templates"
)

//go:embed email_templates
var emailTemplatesFS embed.FS

func main() {
    // Inisialisasi manager
    mgr, err := templates.New(templates.Config{
        FS:           emailTemplatesFS,
        TemplatesDir: "email_templates",
        Logger:       slog.Default(),
    })
    if err != nil {
        // tangani error (mis. directory tidak ada, atau template invalid)
        panic(err)
    }

    // Manager siap digunakan; all templates sudah pre-loaded
    // Simpan ke DI container atau global context untuk akses dari handlers/services
    app.TemplateManager = mgr
}
```

### Load dan render template HTML

```go
// Load template dengan layout (auto-cached)
tmpl, err := mgr.LoadHTML("welcome.html", "layouts/base.html", "layouts/footer.html")
if err != nil {
    // tangani error
}

// Render dengan data
data := map[string]any{
    "Name":    "John",
    "Email":   "john@example.com",
    "ActivURL": "https://example.com/activate?token=abc123",
}

html, err := mgr.RenderHTML(tmpl, "base", data)
if err != nil {
    // tangani error
}

// html sekarang berisi rendered HTML string
fmt.Println(html)
```

### Load dan render template plaintext

```go
// Load plaintext template
tmpl, err := mgr.LoadPlaintext("plain/welcome.txt")
if err != nil {
    // tangani error
}

// Render
data := map[string]any{
    "Name": "Alice",
}

text, err := mgr.RenderPlaintext(tmpl, "welcome", data)
if err != nil {
    // tangani error
}

fmt.Println(text)
```

### Register custom helpers

```go
// Via Config saat init
mgr, _ := templates.New(templates.Config{
    FS:           fs,
    TemplatesDir: "templates",
    Helpers: template.FuncMap{
        "formatPrice": func(cents int) string {
            return fmt.Sprintf("$%.2f", float64(cents)/100)
        },
    },
})

// Atau register setelah init
mgr.RegisterHelper("formatDate", func(t time.Time) string {
    return t.Format("2006-01-02")
})
```

### Metrics dan debugging

```go
// Get metrics
metrics := mgr.GetMetrics()
fmt.Printf("HTML hit rate: %v\n", metrics["html_hit_rate"])
fmt.Printf("Render count: %v\n", metrics["renders"])
fmt.Printf("Errors: %v\n", metrics["errors"])

// List templates
htmlTemplates := mgr.ListHTML()
plainTemplates := mgr.ListPlaintext()

// Clear cache (development)
mgr.ClearCache()

// Reload dari filesystem (development, terutama dengan air)
_ = mgr.Reload()
```

---

## Struktur direktori template

Template dapat diorganisir sesuai kebutuhan Anda. Contoh untuk email templates:

```
email_templates/
├── welcome.html
├── reset-password.html
├── invoice.html
├── layouts/
│   ├── base.html        (define "base" block)
│   ├── header.html      (define "header" block)
│   └── footer.html      (define "footer" block)
└── plain/
    ├── welcome.txt
    ├── reset-password.txt
    └── invoice.txt
```

Contoh untuk UI templates:

```
ui_templates/
├── dashboard.html
├── users/
│   ├── list.html
│   └── detail.html
├── layouts/
│   ├── main.html        (define "main" block)
│   └── sidebar.html     (define "sidebar" block)
└── plain/
    └── sitemap.txt
```

---

## Template file examples

### HTML dengan nested layouts

```html
<!-- email_templates/welcome.html -->
{{ define "welcome" }}
<section class="content">
    <h1>Welcome, {{ upper .Name }}!</h1>
    <p>Your email: {{ lower .Email }}</p>
    
    {{ if gt (len .RecentPurchases) 0 }}
    <div class="recent">
        <h3>Your Recent Purchases:</h3>
        <ul>
        {{ range .RecentPurchases }}
            <li>{{ .Name }} - {{ formatPrice .Amount }}</li>
        {{ end }}
        </ul>
    </div>
    {{ end }}
    
    <p>
        Account Status: 
        {{ if eq .Status "active" }}<span style="color:green;">Active</span>{{ else }}<span style="color:red;">{{ title .Status }}</span>{{ end }}
    </p>
    
    <a href="{{ .ActivURL }}">Activate Account</a>
</section>
{{ end }}

<!-- email_templates/layouts/base.html -->
{{ define "base" }}
<!DOCTYPE html>
<html>
<head>
    <title>Ziswapp Email</title>
    <style>body { font-family: Arial; }</style>
</head>
<body>
    {{ template "header" . }}
    {{ template "welcome" . }}
    {{ template "footer" . }}
</body>
</html>
{{ end }}

<!-- email_templates/layouts/header.html -->
{{ define "header" }}
<header>
    <h2>Ziswapp</h2>
    <p>{{ formatTime now "Monday, 2 January 2006" }}</p>
</header>
{{ end }}

<!-- email_templates/layouts/footer.html -->
{{ define "footer" }}
<footer>
    <p>© {{ now.Year }} Ziswapp. All rights reserved.</p>
    <p>
        Contact: {{ lower .SupportEmail }} | 
        <a href="https://ziswapp.org/unsubscribe?token={{ .UnsubscribeToken }}">Unsubscribe</a>
    </p>
    <p style="font-size:12px; color:#666;">
        Sent on {{ formatTime .SentAt "2006-01-02 15:04:05" }}
    </p>
</footer>
{{ end }}
```

**Helper usage dalam contoh:**
- `upper`: Capitalize nama user
- `lower`: Email lowercase untuk consistency
- `gt`: Greater than comparison (cek jika ada purchases)
- `len`: Get jumlah items dalam slice
- `eq`: Equality check untuk status
- `title`: Title case untuk status display
- `formatTime`: Format timestamp dengan custom layout
- `now`: Get current time untuk tahun dan timestamp

### Plaintext template dengan helpers

```
<!-- email_templates/plain/welcome.txt -->
{{ define "welcome" }}
Hello {{ title .Name }},

Welcome to Ziswapp!

EMAIL: {{ lower .Email }}
STATUS: {{ upper .Status }}

{{ if gt (len .RecentPurchases) 0 }}
RECENT PURCHASES:
{{ range .RecentPurchases }}
- {{ .Name }}: ${{ div .Amount 100 }}.{{ mod .Amount 100 }}
{{ end }}

{{ end }}
ACTIVATE YOUR ACCOUNT:
{{ .ActivURL }}

---
Best regards,
Ziswapp Team
Sent: {{ formatTime now "2006-01-02 15:04" }}
{{ end }}
```

**Helper usage:**
- `title`: Title case nama
- `lower`: Email lowercase
- `upper`: Status uppercase
- `gt`, `len`: Conditional rendering
- `div`, `mod`: Math operations untuk format price
- `formatTime`, `now`: Timestamp formatting

### Template invoice dengan comprehensive helpers

```html
<!-- email_templates/invoice.html -->
{{ define "invoice" }}
<div class="invoice">
    <h1>{{ upper .InvoiceType }} INVOICE</h1>
    <p>Invoice #{{ .InvoiceNumber }} - Due: {{ formatTime .DueDate "Jan 2, 2006" }}</p>
    
    <table border="1" cellpadding="8">
        <thead>
            <tr style="background-color:#f0f0f0;">
                <th>Item</th>
                <th>Qty</th>
                <th>Unit Price</th>
                <th>Total</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Items }}
            <tr>
                <td>{{ title .Name }}</td>
                <td style="text-align:center;">{{ .Quantity }}</td>
                <td style="text-align:right;">{{ formatPrice .UnitPrice }}</td>
                <td style="text-align:right;">{{ formatPrice (mul .UnitPrice .Quantity) }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>
    
    <div class="summary">
        <p><strong>Subtotal:</strong> {{ formatPrice .Subtotal }}</p>
        {{ if gt .TaxAmount 0 }}
        <p><strong>Tax ({{ .TaxPercent }}%):</strong> {{ formatPrice .TaxAmount }}</p>
        {{ end }}
        {{ if gt .DiscountAmount 0 }}
        <p><strong>Discount:</strong> -{{ formatPrice .DiscountAmount }}</p>
        {{ end }}
        <h3><strong>Total: {{ formatPrice .Total }}</strong></h3>
    </div>
    
    <div class="details">
        <p><strong>Customer:</strong> {{ title .CustomerName }}</p>
        <p><strong>Email:</strong> {{ lower .CustomerEmail }}</p>
        <p><strong>Invoice Date:</strong> {{ formatTime .CreatedAt "2 Jan 2006" }}</p>
        <p><strong>Due Date:</strong> {{ formatTime .DueDate "2 Jan 2006" }}</p>
        
        {{ if eq .Status "overdue" }}
        <p style="color:red; font-weight:bold;">⚠️ THIS INVOICE IS OVERDUE</p>
        {{ else if eq .Status "paid" }}
        <p style="color:green; font-weight:bold;">✓ PAID on {{ formatTime .PaidAt "2 Jan 2006" }}</p>
        {{ end }}
    </div>
    
    <p style="font-size:12px; color:#666; margin-top:20px;">
        Generated: {{ formatTime now "2006-01-02 15:04:05 MST" }}
    </p>
</div>
{{ end }}
```

**Comprehensive helper usage:**
- `upper`: Invoice type uppercase
- `formatTime`: Multiple date formats dengan custom layouts
- `title`: Capitalize item names & customer name
- `mul`: Calculate line total (price × quantity)
- `gt`: Conditional rendering (tax, discount, status)
- `eq`: Status comparison untuk different messages
- `lower`: Email lowercase
- `formatPrice`: Format currency amounts (custom helper yang di-register)
- `now`: Current timestamp untuk document generation

### Helper setup untuk templates

Template examples di atas menggunakan `formatPrice` helper yang bukan built-in. Berikut cara setup-nya:

```go
// Dalam service init atau main.go
mgr, err := templates.New(templates.Config{
    FS:           emailTemplatesFS,
    TemplatesDir: "email_templates",
    Logger:       slog.Default(),
    Helpers: template.FuncMap{
        // Custom helper untuk format currency
        "formatPrice": func(cents int) string {
            dollars := cents / 100
            remainder := cents % 100
            return fmt.Sprintf("$%d.%02d", dollars, remainder)
        },
    },
})
if err != nil {
    panic(err)
}
```

Dengan setup ini, semua template dapat menggunakan `{{ formatPrice .Amount }}` untuk format currency dengan benar.

---

## Contoh penggunaan di service/handler

### Email service

```go
package service

import (
    "context"
    "fmt"

    "github.com/ziswapp/ziswapp/pkg/mail"
    "github.com/ziswapp/ziswapp/pkg/templates"
)

type EmailService struct {
    mailer     mail.Mailer
    tmplMgr    *templates.Manager
}

func (svc *EmailService) SendWelcome(ctx context.Context, userEmail, userName, activURL string) error {
    // Load template (auto-cached after first load)
    tmpl, err := svc.tmplMgr.LoadHTML("welcome.html", "layouts/base.html", "layouts/header.html", "layouts/footer.html")
    if err != nil {
        return fmt.Errorf("load template: %w", err)
    }

    // Render dengan data
    data := map[string]any{
        "Name":      userName,
        "Email":     userEmail,
        "ActivURL":  activURL,
    }
    html, err := svc.tmplMgr.RenderHTML(tmpl, "base", data)
    if err != nil {
        return fmt.Errorf("render html: %w", err)
    }

    // Load plaintext template
    plainTmpl, err := svc.tmplMgr.LoadPlaintext("plain/welcome.txt")
    if err != nil {
        return fmt.Errorf("load plaintext: %w", err)
    }

    plainText, err := svc.tmplMgr.RenderPlaintext(plainTmpl, "welcome", data)
    if err != nil {
        return fmt.Errorf("render plaintext: %w", err)
    }

    // Send email
    return svc.mailer.Send(ctx, &mail.Message{
        To:        []string{userEmail},
        Subject:   "Welcome to Ziswapp",
        HTML:      html,
        PlainText: plainText,
    })
}

func (svc *EmailService) SendInvoice(ctx context.Context, userEmail string, invoice *Invoice) error {
    tmpl, err := svc.tmplMgr.LoadHTML("invoice.html", "layouts/base.html")
    if err != nil {
        return fmt.Errorf("load template: %w", err)
    }

    data := map[string]any{
        "Invoice": invoice,
        "Items":   invoice.Items,
        "Total":   invoice.Total,
        "Date":    invoice.CreatedAt,
    }

    html, err := svc.tmplMgr.RenderHTML(tmpl, "base", data)
    if err != nil {
        return fmt.Errorf("render: %w", err)
    }

    // Render plaintext untuk fallback
    plainTmpl, _ := svc.tmplMgr.LoadPlaintext("plain/invoice.txt")
    plainText, _ := svc.tmplMgr.RenderPlaintext(plainTmpl, "invoice", data)

    return svc.mailer.Send(ctx, &mail.Message{
        To:          []string{userEmail},
        Subject:     fmt.Sprintf("Invoice #%s", invoice.Number),
        HTML:        html,
        PlainText:   plainText,
        Attachments: []mail.Attachment{
            {
                Filename:    fmt.Sprintf("invoice-%s.pdf", invoice.Number),
                ContentType: "application/pdf",
                Data:        invoice.PDFBytes,
            },
        },
    })
}
```

### HTTP handler untuk render template

```go
package handler

import (
    "net/http"

    "github.com/ziswapp/ziswapp/pkg/templates"
)

func RenderTemplateHandler(tmplMgr *templates.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Load template
        tmpl, err := tmplMgr.LoadHTML("dashboard.html", "layouts/main.html")
        if err != nil {
            http.Error(w, "template not found", http.StatusNotFound)
            return
        }

        // Prepare data
        data := map[string]any{
            "Title": "Dashboard",
            "User":  r.Header.Get("X-User-Name"),
        }

        // Render
        html, err := tmplMgr.RenderHTML(tmpl, "main", data)
        if err != nil {
            http.Error(w, "render failed", http.StatusInternalServerError)
            return
        }

        // Return response
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(html))
    }
}
```

---

## Praktik terbaik

- **Pre-load di init**: Semua templates di-load saat Manager dibuat, bukan lazy-load. Ini memastikan error ditangkap cepat.
- **Caching**: Template di-cache dalam memory setelah load. Rendering subsequent menggunakan cached template tanpa membaca FS lagi.
- **Layouts**: Gunakan `{{ define }}` blocks untuk composable layouts; pass layout paths ke `LoadHTML()`/`LoadPlaintext()`.
- **Helpers**: Gunakan built-in helpers (upper, lower, formatTime, dll.) dalam templates. Register custom helpers untuk business logic khusus.
- **Context**: Render menghormati context; gunakan `context.WithTimeout()` untuk limit durasi render jika perlu.
- **Metrics**: Monitor `GetMetrics()` untuk track cache hit rate dan identify slow renders (>100ms).
- **Development**: Gunakan `Reload()` dengan air untuk hot-reload templates tanpa restart server.
- **Logging**: Structured logs (slog) sudah built-in untuk troubleshoot.

---

## Thread safety

Manager fully thread-safe:
- `sync.RWMutex` protects template caches
- Metrics dilindungi oleh separate mutex
- Safe untuk concurrent calls dari multiple goroutines

Tidak perlu mutex eksternal saat render dari handler concurrent.

---

## Rincian dan catatan

### Template type detection

- **HTML**: file dengan extension `.html` atau `.tpl`/`.tmpl` di direktori non-`plain`
- **Plaintext**: file dengan extension `.txt` atau file di direktori `plain/`

Heuristic detection berdasarkan file location dan extension. Tidak ada explicit type tagging.

### Nested layouts

Layouts dimuat melalui Go template `{{ define }}` blocks. Load multiple layout files sebelum template utama:

```go
tmpl, _ := mgr.LoadHTML("welcome.html", 
    "layouts/base.html",    // define "base"
    "layouts/header.html",  // define "header"
    "layouts/footer.html",  // define "footer"
)
html, _ := mgr.RenderHTML(tmpl, "base", data)  // render "base" block
```

### Caching dan performance

- Templates cached in-memory setelah first load
- Subsequent loads return cached instance instantly
- No file I/O setelah init phase
- Minimal lock contention (RWMutex, read-heavy)

### Metrics tracking

- `html_hits` / `html_misses`: cache lookup results untuk HTML
- `plain_hits` / `plain_misses`: cache lookup results untuk plaintext
- `html_hit_rate` / `plain_hit_rate`: percentage sebagai string (e.g., "75.50%")
- `renders`: map of template name → count
- `errors`: map of template name → error count
- `total_templates`: total loaded templates
- `html_templates` / `plain_templates`: count per type

Slow render warning: logs warn level jika render >100ms.

### Error handling

- Init error: template invalid, directory tidak ada, parse error
- Load error: template key not found di cache, atau FS read error
- Render error: data binding error, undefined variable, helper error

Semua error return dengan context (wrapped via `fmt.Errorf`).

---

## Development dan testing

### Hot-reload dengan air

Jika menggunakan air untuk development:

```toml
# .air.toml
[build]
include_ext = ["go", "tpl", "tmpl", "html"]  # include template files
exclude_dir = ["testdata"]
```

Air akan restart server saat template files berubah. Template Manager akan reload dari FS:

```go
mgr.Reload()  // dalam development handler atau startup
```

### Testing dengan mock templates

```go
package myservice_test

import (
    "embed"
    "testing"

    "github.com/ziswapp/ziswapp/pkg/templates"
)

//go:embed testdata
var testTemplatesFS embed.FS

func TestSendEmail(t *testing.T) {
    mgr, err := templates.New(templates.Config{
        FS:           testTemplatesFS,
        TemplatesDir: "testdata",
    })
    if err != nil {
        t.Fatalf("setup: %v", err)
    }

    // Use mgr for testing
    tmpl, _ := mgr.LoadHTML("test.html")
    result, _ := mgr.RenderHTML(tmpl, "test", map[string]any{
        "Name": "TestUser",
    })

    if !contains(result, "TestUser") {
        t.Errorf("unexpected result: %s", result)
    }
}

func contains(s, substr string) bool {
    for i := 0; i < len(s)-len(substr)+1; i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

---

## Catatan dan keterbatasan

- Template parsing dilakukan saat init. Jika ada template invalid, Manager creation akan fail. Ini adalah feature (fail-fast).
- Helpers dan template variables tidak divalidasi sampai render-time.
- Large templates atau many concurrent renders dapat impact memory. Monitor metrics.
- Custom helpers dapat memperlambat render jika logic kompleks; pertimbangkan precompute di data preparation phase.
- Go template language terbatas: tidak ada custom loop construct, filter chain, dll. Gunakan helpers untuk logic kompleks.