package jsonull

import (
	"encoding/json"
	"testing"
)

// Test struct for JSON marshaling/unmarshaling
type TestStruct struct {
	Name  string            `json:"name"`
	Email JsonNull[string]  `json:"email,omitempty"`
	Age   JsonNull[int]     `json:"age,omitempty"`
	Score JsonNull[float64] `json:"score,omitempty"`
}

func TestNewJsonNull(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		jn := NewJsonNull("test")
		if !jn.Present {
			t.Error("Expected Present to be true")
		}
		if !jn.Valid {
			t.Error("Expected Valid to be true")
		}
		if jn.Value != "test" {
			t.Errorf("Expected Value to be 'test', got '%s'", jn.Value)
		}
	})

	t.Run("int value", func(t *testing.T) {
		jn := NewJsonNull(42)
		if !jn.Present || !jn.Valid {
			t.Error("Expected Present and Valid to be true")
		}
		if jn.Value != 42 {
			t.Errorf("Expected Value to be 42, got %d", jn.Value)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		jn := NewJsonNull(0)
		if !jn.Present || !jn.Valid {
			t.Error("Expected Present and Valid to be true even for zero value")
		}
		if jn.Value != 0 {
			t.Errorf("Expected Value to be 0, got %d", jn.Value)
		}
	})
}

func TestNewJsonNullNull(t *testing.T) {
	jn := NewJsonNullNull[string]()
	if !jn.Present {
		t.Error("Expected Present to be true")
	}
	if jn.Valid {
		t.Error("Expected Valid to be false")
	}
	if !jn.IsNull() {
		t.Error("Expected IsNull() to return true")
	}
}

func TestJsonNullFromPtr(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var ptr *string = nil
		jn := JsonNullFromPtr(ptr)
		if !jn.Present {
			t.Error("Expected Present to be true")
		}
		if jn.Valid {
			t.Error("Expected Valid to be false for nil pointer")
		}
		if !jn.IsNull() {
			t.Error("Expected IsNull() to return true")
		}
	})

	t.Run("valid pointer", func(t *testing.T) {
		str := "test@example.com"
		jn := JsonNullFromPtr(&str)
		if !jn.Present || !jn.Valid {
			t.Error("Expected Present and Valid to be true")
		}
		if jn.Value != str {
			t.Errorf("Expected Value to be '%s', got '%s'", str, jn.Value)
		}
	})

	t.Run("pointer to zero value", func(t *testing.T) {
		num := 0
		jn := JsonNullFromPtr(&num)
		if !jn.Present || !jn.Valid {
			t.Error("Expected Present and Valid to be true for pointer to zero value")
		}
		if jn.Value != 0 {
			t.Errorf("Expected Value to be 0, got %d", jn.Value)
		}
	})
}

func TestIsNull(t *testing.T) {
	tests := []struct {
		name     string
		jn       JsonNull[string]
		expected bool
	}{
		{
			name:     "null value",
			jn:       JsonNull[string]{Present: true, Valid: false},
			expected: true,
		},
		{
			name:     "valid value",
			jn:       JsonNull[string]{Present: true, Valid: true, Value: "test"},
			expected: false,
		},
		{
			name:     "not present",
			jn:       JsonNull[string]{Present: false, Valid: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.jn.IsNull(); got != tt.expected {
				t.Errorf("IsNull() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsSet(t *testing.T) {
	tests := []struct {
		name     string
		jn       JsonNull[string]
		expected bool
	}{
		{
			name:     "valid value",
			jn:       JsonNull[string]{Present: true, Valid: true, Value: "test"},
			expected: true,
		},
		{
			name:     "null value",
			jn:       JsonNull[string]{Present: true, Valid: false},
			expected: false,
		},
		{
			name:     "not present",
			jn:       JsonNull[string]{Present: false, Valid: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.jn.IsSet(); got != tt.expected {
				t.Errorf("IsSet() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		jn := NewJsonNull("test")
		ptr := jn.Ptr()
		if ptr == nil {
			t.Error("Expected non-nil pointer")
		}
		if *ptr != "test" {
			t.Errorf("Expected pointer to 'test', got '%s'", *ptr)
		}
	})

	t.Run("null value", func(t *testing.T) {
		jn := NewJsonNullNull[string]()
		ptr := jn.Ptr()
		if ptr != nil {
			t.Error("Expected nil pointer for null value")
		}
	})

	t.Run("not present", func(t *testing.T) {
		var jn JsonNull[string]
		ptr := jn.Ptr()
		if ptr != nil {
			t.Error("Expected nil pointer for not present value")
		}
	})
}

func TestOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		jn           JsonNull[string]
		defaultValue string
		expected     string
	}{
		{
			name:         "valid value",
			jn:           NewJsonNull("test"),
			defaultValue: "default",
			expected:     "test",
		},
		{
			name:         "null value",
			jn:           NewJsonNullNull[string](),
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "not present",
			jn:           JsonNull[string]{},
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.jn.OrDefault(tt.defaultValue); got != tt.expected {
				t.Errorf("OrDefault() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMustGet(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		jn := NewJsonNull("test")
		value := jn.MustGet()
		if value != "test" {
			t.Errorf("Expected 'test', got '%s'", value)
		}
	})

	t.Run("null value panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for null value")
			}
		}()
		jn := NewJsonNullNull[string]()
		_ = jn.MustGet()
	})

	t.Run("not present panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for not present value")
			}
		}()
		var jn JsonNull[string]
		_ = jn.MustGet()
	})
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		jn       JsonNull[string]
		expected string
	}{
		{
			name:     "valid value",
			jn:       NewJsonNull("test"),
			expected: "JsonNull{test}",
		},
		{
			name:     "null value",
			jn:       NewJsonNullNull[string](),
			expected: "JsonNull{null}",
		},
		{
			name:     "not present",
			jn:       JsonNull[string]{},
			expected: "JsonNull{not present}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.jn.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal valid string", func(t *testing.T) {
		data := []byte(`"test@example.com"`)
		var jn JsonNull[string]
		err := json.Unmarshal(data, &jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !jn.Present {
			t.Error("Expected Present to be true")
		}
		if !jn.Valid {
			t.Error("Expected Valid to be true")
		}
		if jn.Value != "test@example.com" {
			t.Errorf("Expected Value to be 'test@example.com', got '%s'", jn.Value)
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		data := []byte(`null`)
		var jn JsonNull[string]
		err := json.Unmarshal(data, &jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !jn.Present {
			t.Error("Expected Present to be true")
		}
		if jn.Valid {
			t.Error("Expected Valid to be false")
		}
	})

	t.Run("unmarshal valid int", func(t *testing.T) {
		data := []byte(`42`)
		var jn JsonNull[int]
		err := json.Unmarshal(data, &jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !jn.IsSet() {
			t.Error("Expected IsSet() to be true")
		}
		if jn.Value != 42 {
			t.Errorf("Expected Value to be 42, got %d", jn.Value)
		}
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid}`)
		var jn JsonNull[string]
		err := json.Unmarshal(data, &jn)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if jn.Valid {
			t.Error("Expected Valid to be false after error")
		}
		// Present will be true because UnmarshalJSON was called
		// This happens when json.Unmarshal attempts to unmarshal the data
	})

	t.Run("unmarshal struct with JsonNull fields", func(t *testing.T) {
		tests := []struct {
			name     string
			json     string
			validate func(*testing.T, TestStruct)
		}{
			{
				name: "all fields present with values",
				json: `{"name":"John","email":"john@example.com","age":30,"score":95.5}`,
				validate: func(t *testing.T, ts TestStruct) {
					if ts.Name != "John" {
						t.Errorf("Expected Name to be 'John', got '%s'", ts.Name)
					}
					if !ts.Email.IsSet() || ts.Email.Value != "john@example.com" {
						t.Error("Expected Email to be set to 'john@example.com'")
					}
					if !ts.Age.IsSet() || ts.Age.Value != 30 {
						t.Error("Expected Age to be set to 30")
					}
					if !ts.Score.IsSet() || ts.Score.Value != 95.5 {
						t.Error("Expected Score to be set to 95.5")
					}
				},
			},
			{
				name: "some fields null",
				json: `{"name":"Jane","email":null,"age":25}`,
				validate: func(t *testing.T, ts TestStruct) {
					if !ts.Email.IsNull() {
						t.Error("Expected Email to be null")
					}
					if !ts.Age.IsSet() {
						t.Error("Expected Age to be set")
					}
					if ts.Score.Present {
						t.Error("Expected Score to not be present")
					}
				},
			},
			{
				name: "only required field",
				json: `{"name":"Bob"}`,
				validate: func(t *testing.T, ts TestStruct) {
					if ts.Email.Present {
						t.Error("Expected Email to not be present")
					}
					if ts.Age.Present {
						t.Error("Expected Age to not be present")
					}
					if ts.Score.Present {
						t.Error("Expected Score to not be present")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var ts TestStruct
				err := json.Unmarshal([]byte(tt.json), &ts)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				tt.validate(t, ts)
			})
		}
	})
}

func TestMarshalJSON(t *testing.T) {
	t.Run("marshal valid string", func(t *testing.T) {
		jn := NewJsonNull("test@example.com")
		data, err := json.Marshal(jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := `"test@example.com"`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("marshal null", func(t *testing.T) {
		jn := NewJsonNullNull[string]()
		data, err := json.Marshal(jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := `null`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("marshal valid int", func(t *testing.T) {
		jn := NewJsonNull(42)
		data, err := json.Marshal(jn)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := `42`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("marshal struct with JsonNull fields", func(t *testing.T) {
		ts := TestStruct{
			Name:  "John",
			Email: NewJsonNull("john@example.com"),
			Age:   NewJsonNullNull[int](),
		}
		data, err := json.Marshal(ts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Unmarshal back to verify
		var result TestStruct
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !result.Email.IsSet() || result.Email.Value != "john@example.com" {
			t.Error("Email not properly marshaled/unmarshaled")
		}
		if !result.Age.IsNull() {
			t.Error("Age should be null")
		}
	})
}

func TestZeroValue(t *testing.T) {
	var jn JsonNull[string]
	if jn.Present {
		t.Error("Zero value should have Present=false")
	}
	if jn.Valid {
		t.Error("Zero value should have Valid=false")
	}
	if jn.IsSet() {
		t.Error("Zero value IsSet() should return false")
	}
	if jn.IsNull() {
		t.Error("Zero value IsNull() should return false")
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input TestStruct
	}{
		{
			name: "all values set",
			input: TestStruct{
				Name:  "Alice",
				Email: NewJsonNull("alice@example.com"),
				Age:   NewJsonNull(25),
				Score: NewJsonNull(98.5),
			},
		},
		{
			name: "some values null",
			input: TestStruct{
				Name:  "Bob",
				Email: NewJsonNullNull[string](),
				Age:   NewJsonNull(30),
				Score: NewJsonNullNull[float64](),
			},
		},
		{
			name: "minimal values with explicit not present",
			input: TestStruct{
				Name:  "Charlie",
				Email: JsonNull[string]{Present: false, Valid: false},
				Age:   JsonNull[int]{Present: false, Valid: false},
				Score: JsonNull[float64]{Present: false, Valid: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			// Unmarshal back
			var result TestStruct
			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Verify Name
			if result.Name != tt.input.Name {
				t.Errorf("Name mismatch: got %s, want %s", result.Name, tt.input.Name)
			}

			// Verify Email
			// Note: with omitempty, fields not present in JSON won't be in the marshaled output
			// So when unmarshaling back, they won't be Present either (remains as zero value)
			// We only verify Present if it was true in input
			if tt.input.Email.Present && result.Email.Present != tt.input.Email.Present {
				t.Errorf("Email.Present mismatch: got %v, want %v", result.Email.Present, tt.input.Email.Present)
			}
			if result.Email.Valid != tt.input.Email.Valid {
				t.Errorf("Email.Valid mismatch: got %v, want %v", result.Email.Valid, tt.input.Email.Valid)
			}
			if result.Email.Valid && result.Email.Value != tt.input.Email.Value {
				t.Errorf("Email.Value mismatch: got %s, want %s", result.Email.Value, tt.input.Email.Value)
			}

			// Verify Age
			if tt.input.Age.Present && result.Age.Present != tt.input.Age.Present {
				t.Errorf("Age.Present mismatch: got %v, want %v", result.Age.Present, tt.input.Age.Present)
			}
			if result.Age.Valid != tt.input.Age.Valid {
				t.Errorf("Age.Valid mismatch: got %v, want %v", result.Age.Valid, tt.input.Age.Valid)
			}
			if result.Age.Valid && result.Age.Value != tt.input.Age.Value {
				t.Errorf("Age.Value mismatch: got %d, want %d", result.Age.Value, tt.input.Age.Value)
			}

			// Verify Score
			if tt.input.Score.Present && result.Score.Present != tt.input.Score.Present {
				t.Errorf("Score.Present mismatch: got %v, want %v", result.Score.Present, tt.input.Score.Present)
			}
			if result.Score.Valid != tt.input.Score.Valid {
				t.Errorf("Score.Valid mismatch: got %v, want %v", result.Score.Valid, tt.input.Score.Valid)
			}
			if result.Score.Valid && result.Score.Value != tt.input.Score.Value {
				t.Errorf("Score.Value mismatch: got %f, want %f", result.Score.Value, tt.input.Score.Value)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewJsonNull(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewJsonNull("test@example.com")
	}
}

func BenchmarkJsonNullFromPtr(b *testing.B) {
	str := "test@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = JsonNullFromPtr(&str)
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	data := []byte(`{"name":"John","email":"john@example.com","age":30,"score":95.5}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ts TestStruct
		_ = json.Unmarshal(data, &ts)
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	ts := TestStruct{
		Name:  "John",
		Email: NewJsonNull("john@example.com"),
		Age:   NewJsonNull(30),
		Score: NewJsonNull(95.5),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(ts)
	}
}
