package pgxscan

import "testing"

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"word", "word"},
		{"Word", "word"},
		{"camelCase", "camel_case"},
		{"PascalCase", "pascal_case"},
		{"UserID", "user_id"},
		{"UserIDCard", "user_id_card"},
		{"HTML", "html"},
		{"HTMLParser", "html_parser"},
		{"JSONBody", "json_body"},
		{"GetHTTPSResponse", "get_https_response"},
		{"Column1", "column1"},
		{"User1Name", "user1_name"},
		{"Col1ID", "col1_id"},
		{"Page2Result", "page2_result"},
		{"ID", "id"},
		{"CreatedAt", "created_at"},
		{"UpdatedAt", "updated_at"},
		{"DeletedAt", "deleted_at"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := toSnakeCase(tc.input)
			if got != tc.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func BenchmarkToSnakeCase(b *testing.B) {
	inputs := []string{"CamelCase", "HTMLParser", "GetHTTPSResponse", "UserID", "CreatedAt"}
	b.ResetTimer()
	for range b.N {
		for _, s := range inputs {
			toSnakeCase(s)
		}
	}
}
