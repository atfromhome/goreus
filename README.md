# goreus

[![CI](https://github.com/atfromhome/goreus/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/atfromhome/goreus/actions/workflows/test.yml)


Repositori ini berisi kumpulan paket Go yang dapat digunakan untuk kebutuhan pembuatan aplikasi. Paket yang tersedia mencakup:
- pkg/cache — cache dalam-proses (concurrent-safe) dengan TTL dan LRU
- pkg/storage — driver penyimpanan (S3/R2/local/null) dengan dukungan streaming
- pkg/mail — pengirim email minimal (SMTP/SES/null) dengan pembangun MIME
- pkg/templates — manajer template dengan helpers, caching, dan metrik

Setiap paket dirancang agar kecil, berfokus pada satu tujuan, dan mudah diintegrasikan.

## Paket

- [Cache](./pkg/cache/README.md): in-memory generic cache dengan TTL dan LRU  
- [Storage](./pkg/storage/README.md): driver penyimpanan pluggable (S3/R2/local/null) dengan dukungan streaming  
- [Mail](./pkg/mail/README.md): pengirim email (SMTP/SES/null) + pembangun MIME  
- [Templates](./pkg/templates/README.md): template HTML/plaintext dengan helpers dan metrik  

## Instalasi

Memerlukan Go 1.23+.

```bash
go get github.com/atfromhome/goreus@latest
```

Impor paket sesuai kebutuhan:

```go
import (
  "github.com/atfromhome/goreus/pkg/cache"
  "github.com/atfromhome/goreus/pkg/storage"
  "github.com/atfromhome/goreus/pkg/mail"
  "github.com/atfromhome/goreus/pkg/templates"
)
```

## Penggunaan Cepat

- Cache

```go
c := cache.NewInMemoryCache[string, string](10_000, 5*time.Minute)
c.Set(context.Background(), "greeting", "hello")
v, ok := c.Get(context.Background(), "greeting")
_ = c.Close()
_ = v; _ = ok
```

- Storage

```go
st, _ := storage.New(context.Background(), &storage.Config{Driver: "null"})
path, _ := st.Upload(context.Background(), "demo/hello.txt", []byte("hello"))
data, _ := st.Get(context.Background(), path)
_ = data
```

- Mail

```go
m := mail.NewNullMailer()
_ = m.Send(context.Background(), &mail.Message{
  To:        []string{"user@example.com"},
  Subject:   "Halo",
  PlainText: "Hai di sana!",
})
```

- Templates

```go
//go:embed all:templates
var tmplFS embed.FS

mgr, _ := templates.New(templates.Config{
  FS:           tmplFS,
  TemplatesDir: "templates",
})
tmpl, _ := mgr.LoadHTML("emails/welcome.html", "layouts/base.html")
out, _ := mgr.RenderHTML(tmpl, "welcome.html", map[string]any{"Name": "Alice"})
_ = out
```

## Pengembangan

- Jalankan seluruh pengujian:

```bash
go test ./...
```
