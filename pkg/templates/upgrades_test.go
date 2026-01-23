package templates

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestGlobLoading(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	// Load layout.html along with all components
	tmpl, err := mgr.LoadHTML("layout.html", "components/*.html")
	if err != nil {
		t.Fatalf("LoadHTML with glob failed: %v", err)
	}

	result, err := mgr.RenderHTML(tmpl, "layout.html", map[string]any{"Content": "Body Content"})
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	expectedParts := []string{"<header>Header</header>", "<main>Body Content</main>", "<footer>Footer</footer>"}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing part %q: %s", part, result)
		}
	}
}

func TestXSSProtection(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, _ := mgr.LoadHTML("simple.html")

	// Inject malicious script
	maliciousInput := "<script>alert('xss')</script>"
	result, err := mgr.RenderHTML(tmpl, "simple", map[string]any{
		"Name": maliciousInput,
	})
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	if strings.Contains(result, maliciousInput) {
		t.Error("XSS protection failed: script tag was not escaped")
	}

	if !strings.Contains(result, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;") && !strings.Contains(result, "&lt;script&gt;alert('xss')&lt;/script&gt;") {
		t.Errorf("Expected escaped output, got: %s", result)
	}
}

func TestGlobalComponents(t *testing.T) {
	mgr, err := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
		GlobalComponents: []string{"components/*.html"},
	})
	if err != nil {
		t.Fatalf("New with GlobalComponents failed: %v", err)
	}

	// Load a template that doesn't explicitly load components, but uses them
	// We need a template that uses global components.
	// Let's create a temporary in-memory one or just reuse layout.html but load it without components arg
	
	// Since layout.html refers to "header" and "footer", and we loaded them globally,
	// we should be able to just load layout.html.
	
	tmpl, err := mgr.LoadHTML("layout.html")
	if err != nil {
		t.Fatalf("LoadHTML failed: %v", err)
	}

	result, err := mgr.RenderHTML(tmpl, "layout.html", map[string]any{"Content": "Global Content"})
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	if !strings.Contains(result, "<header>Header</header>") {
		t.Error("Global component 'header' not found in result")
	}
}

func TestDebugPrint(t *testing.T) {
	mgr, _ := New(Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	tmpl, _ := mgr.LoadHTML("simple.html")
	debugInfo := mgr.DebugPrint(tmpl)

	if !strings.Contains(debugInfo, "simple.html") && !strings.Contains(debugInfo, "simple") {
		t.Errorf("DebugPrint output unexpected: %s", debugInfo)
	}
}
