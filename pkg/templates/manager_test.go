package templates

import (
	"embed"
	"log/slog"
	"os"
	"testing"
	"text/template"
)

//go:embed testdata
var testFS embed.FS

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with templates",
			config: Config{
				FS:           testFS,
				TemplatesDir: "testdata",
				Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
			},
			wantErr: false,
		},
		{
			name: "config with custom helpers",
			config: Config{
				FS:           testFS,
				TemplatesDir: "testdata",
				Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
				Helpers: template.FuncMap{
					"custom": func() string { return "custom" },
				},
			},
			wantErr: false,
		},
		{
			name: "invalid templates directory",
			config: Config{
				FS:           testFS,
				TemplatesDir: "nonexistent",
				Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && mgr == nil {
				t.Error("New() returned nil manager without error")
			}
		})
	}
}

func TestLoadHTML(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, err := mgr.LoadHTML("simple.html")
	if err != nil {
		t.Fatalf("LoadHTML() error = %v", err)
	}
	if tmpl == nil {
		t.Fatal("LoadHTML() returned nil template")
	}
}

func TestLoadPlaintext(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, err := mgr.LoadPlaintext("plain/simple.txt")
	if err != nil {
		t.Fatalf("LoadPlaintext() error = %v", err)
	}
	if tmpl == nil {
		t.Fatal("LoadPlaintext() returned nil template")
	}
}

func TestRenderHTML(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, _ := mgr.LoadHTML("simple.html")

	result, err := mgr.RenderHTML(tmpl, "simple", map[string]any{
		"Name": "John",
	})
	if err != nil {
		t.Fatalf("RenderHTML() error = %v", err)
	}
	if result == "" {
		t.Fatal("RenderHTML() returned empty result")
	}
	if !contains(result, "John") {
		t.Errorf("RenderHTML() result doesn't contain expected data: %s", result)
	}
}

func TestRenderPlaintext(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, _ := mgr.LoadPlaintext("plain/simple.txt")

	result, err := mgr.RenderPlaintext(tmpl, "simple", map[string]any{
		"Name": "Jane",
	})
	if err != nil {
		t.Fatalf("RenderPlaintext() error = %v", err)
	}
	if result == "" {
		t.Fatal("RenderPlaintext() returned empty result")
	}
	if !contains(result, "Jane") {
		t.Errorf("RenderPlaintext() result doesn't contain expected data: %s", result)
	}
}

func TestCaching(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	// First load
	tmpl1, err1 := mgr.LoadHTML("simple.html")
	if err1 != nil {
		t.Fatalf("First LoadHTML() error = %v", err1)
	}

	// Second load should return same instance (cached)
	tmpl2, err2 := mgr.LoadHTML("simple.html")
	if err2 != nil {
		t.Fatalf("Second LoadHTML() error = %v", err2)
	}

	if tmpl1 != tmpl2 {
		t.Error("Caching failed: templates are different instances")
	}

	// Check metrics
	metrics := mgr.GetMetrics()
	if metrics["html_hits"].(int64) == 0 {
		t.Error("Cache hit not recorded in metrics")
	}
}

func TestListTemplates(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	htmlList := mgr.ListHTML()
	if len(htmlList) == 0 {
		t.Fatal("ListHTML() returned empty list")
	}

	plainList := mgr.ListPlaintext()
	if len(plainList) == 0 {
		t.Fatal("ListPlaintext() returned empty list")
	}
}

func TestClearCache(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	mgr.LoadHTML("simple.html")
	htmlBefore := len(mgr.ListHTML())

	mgr.ClearCache()
	htmlAfter := len(mgr.ListHTML())

	if htmlBefore == 0 {
		t.Fatal("ListHTML() returned empty list before clear")
	}
	if htmlAfter != 0 {
		t.Errorf("ClearCache() failed: expected 0 templates, got %d", htmlAfter)
	}
}

func TestReload(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	htmlBefore := len(mgr.ListHTML())
	if htmlBefore == 0 {
		t.Fatal("ListHTML() returned empty list before reload")
	}

	err := mgr.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	htmlAfter := len(mgr.ListHTML())
	if htmlAfter != htmlBefore {
		t.Errorf("Reload() changed template count: before=%d, after=%d", htmlBefore, htmlAfter)
	}
}

func TestRegisterHelper(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	mgr.RegisterHelper("testHelper", func() string {
		return "test_value"
	})

	if _, exists := mgr.helpers["testHelper"]; !exists {
		t.Error("RegisterHelper() failed: helper not registered")
	}
}

func TestGetMetrics(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	// Generate some activity
	tmpl, _ := mgr.LoadHTML("simple.html")
	mgr.RenderHTML(tmpl, "simple", map[string]any{"Name": "Test"})

	metrics := mgr.GetMetrics()

	expectedKeys := []string{
		"html_hits",
		"html_misses",
		"html_hit_rate",
		"plain_hits",
		"plain_misses",
		"plain_hit_rate",
		"renders",
		"errors",
		"total_templates",
		"html_templates",
		"plain_templates",
	}

	for _, key := range expectedKeys {
		if _, exists := metrics[key]; !exists {
			t.Errorf("GetMetrics() missing key: %s", key)
		}
	}
}

func TestDefaultHelpers(t *testing.T) {
	helpers := DefaultHelpers()

	expectedHelpers := []string{
		"upper",
		"lower",
		"title",
		"now",
		"formatTime",
		"add",
		"sub",
		"mul",
		"div",
		"eq",
		"ne",
		"gt",
		"lt",
	}

	for _, helper := range expectedHelpers {
		if _, exists := helpers[helper]; !exists {
			t.Errorf("DefaultHelpers() missing helper: %s", helper)
		}
	}
}

func TestRenderWithHelpers(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	// Reload untuk mengapply helpers
	mgr.Reload()

	tmpl, _ := mgr.LoadHTML("simple.html")
	result, err := mgr.RenderHTML(tmpl, "simple", map[string]any{
		"Name": "alice",
	})

	if err != nil {
		t.Fatalf("RenderHTML() with helpers error = %v", err)
	}
	if result == "" {
		t.Fatal("RenderHTML() returned empty result")
	}
}

func TestErrorHandling(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	// Try to load non-existent template
	_, err := mgr.LoadHTML("nonexistent.html")
	if err == nil {
		t.Error("LoadHTML() should return error for non-existent template")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
