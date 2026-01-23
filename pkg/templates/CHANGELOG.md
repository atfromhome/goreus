# Changelog

## [v1.1.0] - 2026-01-23

### Added
- **Global Components**: New `GlobalComponents` field in `Config` to pre-load shared components (e.g., "components/*.html").
- **Glob Support**: `LoadHTML` and `LoadPlaintext` now support glob patterns (e.g., "widgets/*.html").
- **XSS Protection**: HTML templates now use `html/template` instead of `text/template` for context-aware escaping.
- **DebugPrint**: New method `DebugPrint(tmpl)` to list defined templates for debugging.
- **Consistent Naming**: Templates are now named using their relative paths to avoid collisions.

### Changed
- **Breaking Change**: `LoadHTML` now returns `*html/template.Template` instead of `*text/template.Template`.
- **Breaking Change**: `RenderHTML` now accepts `*html/template.Template`.
- `Render` method signature updated to accept `TemplateExecutor` interface to support both HTML and text templates.

### Migration Guide
Most code using type inference (`:=`) will continue to work without changes.

**If you use explicit types:**

Before:
```go
import "text/template"

var tmpl *template.Template
tmpl, _ = mgr.LoadHTML("page.html")
```

After:
```go
import (
    "html/template"
    // "text/template" // still needed for LoadPlaintext
)

var tmpl *template.Template
tmpl, _ = mgr.LoadHTML("page.html")
```
