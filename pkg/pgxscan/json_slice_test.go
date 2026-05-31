package pgxscan

import (
	"encoding/json"
	"testing"
)

// --- Scan ---

func TestJsonSlice_Scan_Nil(t *testing.T) {
	var j JsonSlice[int]
	if err := j.Scan(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j) != 0 {
		t.Errorf("expected empty slice, got %v", j)
	}
}

func TestJsonSlice_Scan_NullBytes(t *testing.T) {
	var j JsonSlice[int]
	if err := j.Scan([]byte("null")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j) != 0 {
		t.Errorf("expected empty slice, got %v", j)
	}
}

func TestJsonSlice_Scan_NullString(t *testing.T) {
	var j JsonSlice[string]
	if err := j.Scan("null"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(j) != 0 {
		t.Errorf("expected empty slice, got %v", j)
	}
}

func TestJsonSlice_Scan_BytesValid(t *testing.T) {
	var j JsonSlice[int]
	if err := j.Scan([]byte(`[1,2,3]`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Len() != 3 || j[0] != 1 || j[1] != 2 || j[2] != 3 {
		t.Errorf("unexpected result: %v", j)
	}
}

func TestJsonSlice_Scan_StringValid(t *testing.T) {
	var j JsonSlice[string]
	if err := j.Scan(`["a","b","c"]`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Len() != 3 || j[0] != "a" || j[1] != "b" || j[2] != "c" {
		t.Errorf("unexpected result: %v", j)
	}
}

func TestJsonSlice_Scan_InvalidType(t *testing.T) {
	var j JsonSlice[int]
	if err := j.Scan(42); err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestJsonSlice_Scan_InvalidJSON(t *testing.T) {
	var j JsonSlice[int]
	if err := j.Scan([]byte(`not json`)); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func TestJsonSlice_Scan_Struct(t *testing.T) {
	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var j JsonSlice[item]
	if err := j.Scan(`[{"id":1,"name":"foo"},{"id":2,"name":"bar"}]`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Len() != 2 || j[0].ID != 1 || j[1].Name != "bar" {
		t.Errorf("unexpected result: %v", j)
	}
}

// --- MarshalJSON ---

func TestJsonSlice_MarshalJSON_Nil(t *testing.T) {
	var j JsonSlice[int]
	b, err := j.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("expected [], got %s", b)
	}
}

func TestJsonSlice_MarshalJSON_Data(t *testing.T) {
	j := JsonSlice[int]{1, 2, 3}
	b, err := j.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "[1,2,3]" {
		t.Errorf("expected [1,2,3], got %s", b)
	}
}

func TestJsonSlice_MarshalJSON_RoundTrip(t *testing.T) {
	original := JsonSlice[string]{"x", "y", "z"}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got JsonSlice[string]
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Len() != original.Len() || got[0] != original[0] {
		t.Errorf("round-trip mismatch: %v != %v", got, original)
	}
}

// --- UnmarshalJSON ---

func TestJsonSlice_UnmarshalJSON_Valid(t *testing.T) {
	var j JsonSlice[float64]
	if err := json.Unmarshal([]byte(`[1.1,2.2,3.3]`), &j); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j.Len() != 3 {
		t.Errorf("expected 3 elements, got %d", j.Len())
	}
}

func TestJsonSlice_UnmarshalJSON_Invalid(t *testing.T) {
	var j JsonSlice[int]
	if err := json.Unmarshal([]byte(`"not an array"`), &j); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ToSlice / Len ---

func TestJsonSlice_ToSlice(t *testing.T) {
	j := JsonSlice[int]{10, 20, 30}
	s := j.ToSlice()
	if len(s) != 3 || s[0] != 10 {
		t.Errorf("unexpected ToSlice result: %v", s)
	}
}

func TestJsonSlice_Len(t *testing.T) {
	tests := []struct {
		j    JsonSlice[int]
		want int
	}{
		{nil, 0},
		{JsonSlice[int]{}, 0},
		{JsonSlice[int]{1, 2, 3}, 3},
	}
	for _, tc := range tests {
		if got := tc.j.Len(); got != tc.want {
			t.Errorf("Len() = %d, want %d", got, tc.want)
		}
	}
}

// --- Value (driver.Valuer) ---

func TestJsonSlice_Value_Nil(t *testing.T) {
	var j JsonSlice[int]
	v, err := j.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != nil {
		t.Errorf("nil JsonSlice should produce nil driver.Value, got %v", v)
	}
}

func TestJsonSlice_Value_Data(t *testing.T) {
	j := JsonSlice[int]{1, 2, 3}
	v, err := j.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "[1,2,3]" {
		t.Errorf("expected [1,2,3], got %v", v)
	}
}

func TestJsonSlice_Value_Empty(t *testing.T) {
	j := JsonSlice[int]{}
	v, err := j.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "[]" {
		t.Errorf("expected [], got %v", v)
	}
}

// --- Benchmark ---

func BenchmarkJsonSlice_Scan(b *testing.B) {
	data := []byte(`[1,2,3,4,5,6,7,8,9,10]`)
	b.ResetTimer()
	for range b.N {
		var j JsonSlice[int]
		_ = j.Scan(data)
	}
}
