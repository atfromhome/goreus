package templates

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// TemplateExecutor interface that both html/template and text/template satisfy
type TemplateExecutor interface {
	ExecuteTemplate(wr io.Writer, name string, data any) error
	Name() string
}

// Manager mengelola HTML dan plaintext templates dengan caching, helpers, dan metrics
type Manager struct {
	htmlTmpls    map[string]*htmltemplate.Template
	plainTmpls   map[string]*template.Template
	baseHtml     *htmltemplate.Template
	fs           embed.FS
	templatesDir string
	mu           sync.RWMutex
	logger       *slog.Logger
	metrics      *Metrics
	helpers      template.FuncMap
}

// Metrics melacak penggunaan template
type Metrics struct {
	mu          sync.RWMutex
	htmlHits    int64
	htmlMisses  int64
	plainHits   int64
	plainMisses int64
	renders     map[string]int64
	errors      map[string]int64
}

// Config untuk inisialisasi Manager
type Config struct {
	FS               embed.FS
	TemplatesDir     string
	Logger           *slog.Logger
	Helpers          template.FuncMap
	GlobalComponents []string // Glob patterns for global HTML components (e.g. "components/*.html")
}

// New membuat Manager baru dengan pre-load semua templates
func New(cfg Config) (*Manager, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	if cfg.Helpers == nil {
		cfg.Helpers = make(template.FuncMap)
	}

	// Merge default helpers dengan custom helpers
	helpers := DefaultHelpers()
	for k, v := range cfg.Helpers {
		helpers[k] = v
	}

	// Initialize base HTML template with helpers and global components
	baseHtml := htmltemplate.New("base").Funcs(helpers)
	if len(cfg.GlobalComponents) > 0 {
		var patterns []string
		for _, p := range cfg.GlobalComponents {
			patterns = append(patterns, filepath.ToSlash(filepath.Join(cfg.TemplatesDir, p)))
		}
		var err error
		baseHtml, err = baseHtml.ParseFS(cfg.FS, patterns...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse global components: %w", err)
		}
	}

	m := &Manager{
		htmlTmpls:    make(map[string]*htmltemplate.Template),
		plainTmpls:   make(map[string]*template.Template),
		baseHtml:     baseHtml,
		fs:           cfg.FS,
		templatesDir: cfg.TemplatesDir,
		logger:       cfg.Logger,
		metrics: &Metrics{
			renders: make(map[string]int64),
			errors:  make(map[string]int64),
		},
		helpers: helpers,
	}

	// Pre-load semua templates
	if err := m.preloadAllTemplates(); err != nil {
		return nil, fmt.Errorf("failed to preload templates: %w", err)
	}

	m.logger.Info("template manager initialized",
		"templates_dir", cfg.TemplatesDir,
		"html_count", len(m.htmlTmpls),
		"plain_count", len(m.plainTmpls),
	)

	return m, nil
}

// preloadAllTemplates scan dan load semua templates saat init
func (m *Manager) preloadAllTemplates() error {
	return fs.WalkDir(m.fs, m.templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !isTemplateFile(d.Name()) {
			return nil
		}

		// Tentukan tipe template berdasarkan lokasi file
		relPath, _ := filepath.Rel(m.templatesDir, path)
		relPath = filepath.ToSlash(relPath)

		if isHTMLPath(path) {
			if err := m.loadHTMLTemplate(path, relPath); err != nil {
				m.logger.Warn("failed to load html template", "path", path, "error", err)
			}
		} else if isPlainPath(path) {
			if err := m.loadPlainTemplate(path, relPath); err != nil {
				m.logger.Warn("failed to load plain template", "path", path, "error", err)
			}
		}

		return nil
	})
}

// loadHTMLTemplate load single HTML template dengan support nested layouts
func (m *Manager) loadHTMLTemplate(path string, relPath string) error {
	// Clone from base to get helpers and global components
	tmpl, err := m.baseHtml.Clone()
	if err != nil {
		return err
	}
	
	// Set name to relative path for consistency
	// Note: The main template name will still be what ParseFS determines (usually the filename),
	// but we store it with the relPath key. 
	// To enforce the name, we would need to manually parse string content, but ParseFS is preferred.
	// We can wrap the execution to use the correct template if we know it.
	
	// Parse the specific file
	tmpl, err = tmpl.ParseFS(m.fs, path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.htmlTmpls[relPath] = tmpl
	m.mu.Unlock()

	return nil
}

// loadPlainTemplate load single plaintext template
func (m *Manager) loadPlainTemplate(path string, relPath string) error {
	tmpl := template.New(filepath.Base(path))
	tmpl.Funcs(m.helpers)

	tmpl, err := tmpl.ParseFS(m.fs, path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.plainTmpls[relPath] = tmpl
	m.mu.Unlock()

	return nil
}

// LoadHTML load atau get cached HTML template dengan layout support
// layoutPaths bisa berupa full paths atau glob patterns
func (m *Manager) LoadHTML(templatePath string, layoutPatterns ...string) (*htmltemplate.Template, error) {
	key := templatePath
	if len(layoutPatterns) > 0 {
		key += "::" + strings.Join(layoutPatterns, "::")
	}

	m.mu.RLock()
	if tmpl, ok := m.htmlTmpls[key]; ok {
		m.mu.RUnlock()
		m.recordHit("html")
		return tmpl, nil
	}
	m.mu.RUnlock()

	m.recordMiss("html")

	// Clone from base
	tmpl, err := m.baseHtml.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone base template: %w", err)
	}

	// Prepare patterns: main file + layouts
	patterns := make([]string, 0, len(layoutPatterns)+1)
	patterns = append(patterns, filepath.ToSlash(filepath.Join(m.templatesDir, templatePath)))
	
	for _, layout := range layoutPatterns {
		// Assume patterns are relative to templatesDir if not absolute (embed fs always relative to root)
		// But here user likely passes "components/*.html" relative to templatesDir
		patterns = append(patterns, filepath.ToSlash(filepath.Join(m.templatesDir, layout)))
	}

	tmpl, err = tmpl.ParseFS(m.fs, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML template %s: %w", templatePath, err)
	}

	m.mu.Lock()
	m.htmlTmpls[key] = tmpl
	m.mu.Unlock()

	return tmpl, nil
}

// LoadPlaintext load atau get cached plaintext template dengan layout support
func (m *Manager) LoadPlaintext(templatePath string, layoutPatterns ...string) (*template.Template, error) {
	key := templatePath
	if len(layoutPatterns) > 0 {
		key += "::" + strings.Join(layoutPatterns, "::")
	}

	m.mu.RLock()
	if tmpl, ok := m.plainTmpls[key]; ok {
		m.mu.RUnlock()
		m.recordHit("plain")
		return tmpl, nil
	}
	m.mu.RUnlock()

	m.recordMiss("plain")

	files := make([]string, 0, len(layoutPatterns)+1)
	for _, layout := range layoutPatterns {
		files = append(files, filepath.ToSlash(filepath.Join(m.templatesDir, layout)))
	}
	files = append(files, filepath.ToSlash(filepath.Join(m.templatesDir, templatePath)))

	tmpl := template.New(filepath.Base(templatePath))
	tmpl.Funcs(m.helpers)

	tmpl, err := tmpl.ParseFS(m.fs, files...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plaintext template %s: %w", templatePath, err)
	}

	m.mu.Lock()
	m.plainTmpls[key] = tmpl
	m.mu.Unlock()

	return tmpl, nil
}

// Render unified render method - render template dengan nama template utama
// templateName default ke template name jika kosong
func (m *Manager) Render(tmpl TemplateExecutor, data any, templateName ...string) (string, error) {
	name := tmpl.Name()
	if len(templateName) > 0 && templateName[0] != "" {
		name = templateName[0]
	}

	start := time.Now()
	var buf bytes.Buffer

	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		m.recordError(name)
		return "", fmt.Errorf("failed to render template %s: %w", name, err)
	}

	m.recordRender(name, time.Since(start))
	return buf.String(), nil
}

// RenderHTML render template HTML
func (m *Manager) RenderHTML(tmpl *htmltemplate.Template, templateName string, data any) (string, error) {
	return m.Render(tmpl, data, templateName)
}

// RenderPlaintext render template plaintext
func (m *Manager) RenderPlaintext(tmpl *template.Template, templateName string, data any) (string, error) {
	return m.Render(tmpl, data, templateName)
}

// DebugPrint prints the defined templates in the given template instance for debugging
func (m *Manager) DebugPrint(tmpl TemplateExecutor) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Template: %s\n", tmpl.Name()))
	
	// html/template and text/template both have DefinedTemplates() string
	// but it's not part of the interface we defined.
	// We need type assertion.
	
	switch t := tmpl.(type) {
	case *htmltemplate.Template:
		buf.WriteString(t.DefinedTemplates())
	case *template.Template:
		buf.WriteString(t.DefinedTemplates())
	default:
		buf.WriteString("Unknown template type")
	}
	
	return buf.String()
}

// ListHTML return list available HTML template keys
func (m *Manager) ListHTML() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.htmlTmpls))
	for k := range m.htmlTmpls {
		keys = append(keys, k)
	}
	return keys
}

// ListPlaintext return list available plaintext template keys
func (m *Manager) ListPlaintext() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.plainTmpls))
	for k := range m.plainTmpls {
		keys = append(keys, k)
	}
	return keys
}

// ClearCache membersihkan semua cached templates
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.htmlTmpls = make(map[string]*htmltemplate.Template)
	m.plainTmpls = make(map[string]*template.Template)

	m.logger.Debug("template cache cleared")
}

// Reload clear cache dan re-load semua templates
func (m *Manager) Reload() error {
	m.ClearCache()
	return m.preloadAllTemplates()
}

// RegisterHelper tambah atau update helper function
func (m *Manager) RegisterHelper(name string, fn any) {
	m.helpers[name] = fn
	m.logger.Debug("helper registered", "name", name)
}

// GetMetrics return current metrics snapshot
func (m *Manager) GetMetrics() map[string]any {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	htmlTotal := m.metrics.htmlHits + m.metrics.htmlMisses
	plainTotal := m.metrics.plainHits + m.metrics.plainMisses

	htmlHitRate := 0.0
	if htmlTotal > 0 {
		htmlHitRate = float64(m.metrics.htmlHits) / float64(htmlTotal) * 100
	}

	plainHitRate := 0.0
	if plainTotal > 0 {
		plainHitRate = float64(m.metrics.plainHits) / float64(plainTotal) * 100
	}

	return map[string]any{
		"html_hits":       m.metrics.htmlHits,
		"html_misses":     m.metrics.htmlMisses,
		"html_hit_rate":   fmt.Sprintf("%.2f%%", htmlHitRate),
		"plain_hits":      m.metrics.plainHits,
		"plain_misses":    m.metrics.plainMisses,
		"plain_hit_rate":  fmt.Sprintf("%.2f%%", plainHitRate),
		"renders":         m.metrics.renders,
		"errors":          m.metrics.errors,
		"total_templates": len(m.htmlTmpls) + len(m.plainTmpls),
		"html_templates":  len(m.htmlTmpls),
		"plain_templates": len(m.plainTmpls),
	}
}

// --- Helper functions ---

func isTemplateFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".tpl" || ext == ".tmpl" || ext == ".html" || ext == ".txt"
}

func isHTMLPath(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".html" {
		return true
	}
	if (ext == ".tpl" || ext == ".tmpl") && !strings.Contains(path, "plain") {
		return true
	}
	return false
}

func isPlainPath(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".txt" {
		return true
	}
	if (ext == ".tpl" || ext == ".tmpl") && strings.Contains(path, "plain") {
		return true
	}
	return false
}

func (m *Manager) recordHit(ttype string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	switch ttype {
	case "html":
		m.metrics.htmlHits++
	case "plain":
		m.metrics.plainHits++
	}
}

func (m *Manager) recordMiss(ttype string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	switch ttype {
	case "html":
		m.metrics.htmlMisses++
	case "plain":
		m.metrics.plainMisses++
	}
}

func (m *Manager) recordRender(name string, duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.renders[name]++

	if duration > 100*time.Millisecond {
		m.logger.Warn("slow template render",
			"template", name,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

func (m *Manager) recordError(name string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.errors[name]++
}

// DefaultHelpers return built-in template helper functions
func DefaultHelpers() template.FuncMap {
	return template.FuncMap{
		// String helpers
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"split":     strings.Split,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// Time helpers
		"now": func() time.Time {
			return time.Now()
		},
		"formatTime": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"formatUnix": func(unix int64) string {
			return time.Unix(0, unix*1000000).Format(time.RFC3339)
		},

		// Math helpers
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// Comparison helpers
		"eq": func(a, b any) bool {
			return a == b
		},
		"ne": func(a, b any) bool {
			return a != b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"lte": func(a, b int) bool {
			return a <= b
		},

		// Slice helpers
		"len": func(v any) int {
			switch val := v.(type) {
			case []string:
				return len(val)
			case []any:
				return len(val)
			case map[string]any:
				return len(val)
			default:
				return 0
			}
		},
		"first": func(items []any) any {
			if len(items) > 0 {
				return items[0]
			}
			return nil
		},
		"last": func(items []any) any {
			if len(items) > 0 {
				return items[len(items)-1]
			}
			return nil
		},

		// Type helpers
		"typeof": func(v any) string {
			return fmt.Sprintf("%T", v)
		},
	}
}
