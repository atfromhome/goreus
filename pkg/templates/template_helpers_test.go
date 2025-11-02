package templates

import (
	"log/slog"
	"strings"
	"testing"
	"text/template"
	"time"
)

func TestHelperUpper(t *testing.T) {
	helpers := DefaultHelpers()
	upper := helpers["upper"].(func(string) string)

	result := upper("hello")
	if result != "HELLO" {
		t.Errorf("upper helper failed: got %q, want %q", result, "HELLO")
	}
}

func TestHelperLower(t *testing.T) {
	helpers := DefaultHelpers()
	lower := helpers["lower"].(func(string) string)

	result := lower("HELLO")
	if result != "hello" {
		t.Errorf("lower helper failed: got %q, want %q", result, "hello")
	}
}

func TestHelperTrim(t *testing.T) {
	helpers := DefaultHelpers()
	trim := helpers["trim"].(func(string) string)

	result := trim("  hello  ")
	if result != "hello" {
		t.Errorf("trim helper failed: got %q, want %q", result, "hello")
	}
}

func TestHelperJoin(t *testing.T) {
	helpers := DefaultHelpers()
	join := helpers["join"].(func(string, []string) string)

	result := join(",", []string{"a", "b", "c"})
	if result != "a,b,c" {
		t.Errorf("join helper failed: got %q, want %q", result, "a,b,c")
	}
}

func TestHelperSplit(t *testing.T) {
	helpers := DefaultHelpers()
	split := helpers["split"].(func(string, string) []string)

	result := split("a,b,c", ",")
	if len(result) != 3 || result[0] != "a" {
		t.Errorf("split helper failed: got %v", result)
	}
}

func TestHelperContains(t *testing.T) {
	helpers := DefaultHelpers()
	contains := helpers["contains"].(func(string, string) bool)

	if !contains("hello world", "world") {
		t.Error("contains helper should return true")
	}
	if contains("hello", "world") {
		t.Error("contains helper should return false")
	}
}

func TestHelperAdd(t *testing.T) {
	helpers := DefaultHelpers()
	add := helpers["add"].(func(int, int) int)

	result := add(2, 3)
	if result != 5 {
		t.Errorf("add helper failed: got %d, want %d", result, 5)
	}
}

func TestHelperSub(t *testing.T) {
	helpers := DefaultHelpers()
	sub := helpers["sub"].(func(int, int) int)

	result := sub(10, 3)
	if result != 7 {
		t.Errorf("sub helper failed: got %d, want %d", result, 7)
	}
}

func TestHelperMul(t *testing.T) {
	helpers := DefaultHelpers()
	mul := helpers["mul"].(func(int, int) int)

	result := mul(4, 5)
	if result != 20 {
		t.Errorf("mul helper failed: got %d, want %d", result, 20)
	}
}

func TestHelperDiv(t *testing.T) {
	helpers := DefaultHelpers()
	div := helpers["div"].(func(int, int) int)

	result := div(20, 4)
	if result != 5 {
		t.Errorf("div helper failed: got %d, want %d", result, 5)
	}

	result = div(20, 0)
	if result != 0 {
		t.Errorf("div by zero should return 0, got %d", result)
	}
}

func TestHelperEq(t *testing.T) {
	helpers := DefaultHelpers()
	eq := helpers["eq"].(func(interface{}, interface{}) bool)

	if !eq(5, 5) {
		t.Error("eq helper should return true for equal values")
	}
	if eq(5, 6) {
		t.Error("eq helper should return false for unequal values")
	}
}

func TestHelperNe(t *testing.T) {
	helpers := DefaultHelpers()
	ne := helpers["ne"].(func(interface{}, interface{}) bool)

	if !ne(5, 6) {
		t.Error("ne helper should return true for unequal values")
	}
	if ne(5, 5) {
		t.Error("ne helper should return false for equal values")
	}
}

func TestHelperGt(t *testing.T) {
	helpers := DefaultHelpers()
	gt := helpers["gt"].(func(int, int) bool)

	if !gt(10, 5) {
		t.Error("gt helper failed: 10 > 5 should be true")
	}
	if gt(5, 10) {
		t.Error("gt helper failed: 5 > 10 should be false")
	}
}

func TestHelperLt(t *testing.T) {
	helpers := DefaultHelpers()
	lt := helpers["lt"].(func(int, int) bool)

	if !lt(5, 10) {
		t.Error("lt helper failed: 5 < 10 should be true")
	}
	if lt(10, 5) {
		t.Error("lt helper failed: 10 < 5 should be false")
	}
}

func TestHelperLen(t *testing.T) {
	helpers := DefaultHelpers()
	lenHelper := helpers["len"].(func(interface{}) int)

	if lenHelper([]string{"a", "b"}) != 2 {
		t.Error("len helper failed for slice")
	}
	if lenHelper(map[string]interface{}{"a": 1, "b": 2}) != 2 {
		t.Error("len helper failed for map")
	}
}

func TestHelperTypeOf(t *testing.T) {
	helpers := DefaultHelpers()
	typeOf := helpers["typeof"].(func(interface{}) string)

	result := typeOf("hello")
	if !strings.Contains(result, "string") {
		t.Errorf("typeof helper failed: got %q", result)
	}
}

func TestHelperFormatTime(t *testing.T) {
	helpers := DefaultHelpers()
	formatTime := helpers["formatTime"].(func(time.Time, string) string)

	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := formatTime(tm, "2006-01-02")

	if result != "2024-01-15" {
		t.Errorf("formatTime failed: got %q, want %q", result, "2024-01-15")
	}
}

func TestHelperNow(t *testing.T) {
	helpers := DefaultHelpers()
	now := helpers["now"].(func() time.Time)

	result := now()
	if result.IsZero() {
		t.Error("now helper returned zero time")
	}
	if time.Since(result) > 1*time.Second {
		t.Error("now helper returned outdated time")
	}
}

func TestHelperFirstLast(t *testing.T) {
	helpers := DefaultHelpers()
	first := helpers["first"].(func([]interface{}) interface{})
	last := helpers["last"].(func([]interface{}) interface{})

	items := []interface{}{"a", "b", "c"}

	if first(items) != "a" {
		t.Error("first helper failed")
	}
	if last(items) != "c" {
		t.Error("last helper failed")
	}
}

func TestHelperHasPrefix(t *testing.T) {
	helpers := DefaultHelpers()
	hasPrefix := helpers["hasPrefix"].(func(string, string) bool)

	if !hasPrefix("hello", "hel") {
		t.Error("hasPrefix should return true")
	}
	if hasPrefix("hello", "world") {
		t.Error("hasPrefix should return false")
	}
}

func TestHelperHasSuffix(t *testing.T) {
	helpers := DefaultHelpers()
	hasSuffix := helpers["hasSuffix"].(func(string, string) bool)

	if !hasSuffix("hello", "lo") {
		t.Error("hasSuffix should return true")
	}
	if hasSuffix("hello", "world") {
		t.Error("hasSuffix should return false")
	}
}

func TestIsTemplateFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"tpl extension", "template.tpl", true},
		{"tmpl extension", "template.tmpl", true},
		{"html extension", "template.html", true},
		{"txt extension", "template.txt", true},
		{"go extension", "file.go", false},
		{"no extension", "template", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTemplateFile(tt.filename)
			if result != tt.want {
				t.Errorf("isTemplateFile(%q) = %v, want %v", tt.filename, result, tt.want)
			}
		})
	}
}

func TestIsHTMLPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"html file", "templates/page.html", true},
		{"tpl file not in plain", "templates/page.tpl", true},
		{"plain tpl file", "templates/plain/email.tpl", false},
		{"txt file", "templates/email.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTMLPath(tt.path)
			if result != tt.want {
				t.Errorf("isHTMLPath(%q) = %v, want %v", tt.path, result, tt.want)
			}
		})
	}
}

func TestIsPlainPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"txt file", "templates/email.txt", true},
		{"plain tpl file", "templates/plain/email.tpl", true},
		{"html file", "templates/page.html", false},
		{"tpl file not in plain", "templates/page.tpl", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPlainPath(tt.path)
			if result != tt.want {
				t.Errorf("isPlainPath(%q) = %v, want %v", tt.path, result, tt.want)
			}
		})
	}
}

func TestManagerRenderWithHelpers(t *testing.T) {
	cfg := Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.Default(),
	}

	manager, _ := New(cfg)

	tmpl := template.New("test")
	tmpl.Funcs(manager.helpers)
	tmpl, _ = tmpl.Parse(`{{ upper "hello" }}`)

	result, err := manager.Render(tmpl, nil, "test")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(result, "HELLO") {
		t.Errorf("expected HELLO in result, got %s", result)
	}
}

func TestManagerRegisterHelperCustom(t *testing.T) {
	cfg := Config{
		FS:           testFS,
		TemplatesDir: "testdata",
		Logger:       slog.Default(),
	}

	manager, _ := New(cfg)

	manager.RegisterHelper("doubleIt", func(n int) int {
		return n * 2
	})

	if _, ok := manager.helpers["doubleIt"]; !ok {
		t.Error("custom helper not registered")
	}
}
