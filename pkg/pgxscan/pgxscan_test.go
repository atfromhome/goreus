package pgxscan

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// mockRow implementasi pgx.CollectableRow untuk testing.
type mockRow struct {
	cols   []string
	values []any
	err    error
}

func (m *mockRow) FieldDescriptions() []pgconn.FieldDescription {
	descs := make([]pgconn.FieldDescription, len(m.cols))
	for i, c := range m.cols {
		descs[i] = pgconn.FieldDescription{Name: c}
	}
	return descs
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	for i, d := range dest {
		if i >= len(m.values) {
			break
		}
		if err := assign(d, m.values[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockRow) Values() ([]any, error)  { return m.values, nil }
func (m *mockRow) RawValues() [][]byte     { return nil }

// assign menyalin nilai src ke pointer dst via reflect.
func assign(dst, src any) error {
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return fmt.Errorf("assign: dst must be a non-nil pointer")
	}
	dv = dv.Elem()

	// Cek sql.Scanner terlebih dahulu (misal JsonSlice).
	if scanner, ok := dst.(interface{ Scan(any) error }); ok {
		return scanner.Scan(src)
	}

	if src == nil {
		dv.Set(reflect.Zero(dv.Type()))
		return nil
	}

	sv := reflect.ValueOf(src)
	if sv.Type().AssignableTo(dv.Type()) {
		dv.Set(sv)
		return nil
	}
	if sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}
	return fmt.Errorf("assign: cannot assign %T to %T", src, dst)
}

// --- RowToStruct ---

func TestRowToStruct_FlatStruct(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	row := &mockRow{
		cols:   []string{"id", "name"},
		values: []any{42, "Alice"},
	}

	got, err := RowToStruct[User](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 || got.Name != "Alice" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestRowToStruct_DbTag(t *testing.T) {
	type Order struct {
		OrderID int    `db:"order_id"`
		Status  string `db:"status"`
	}

	row := &mockRow{
		cols:   []string{"order_id", "status"},
		values: []any{99, "shipped"},
	}

	got, err := RowToStruct[Order](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.OrderID != 99 || got.Status != "shipped" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestRowToStruct_SkipTag(t *testing.T) {
	type Product struct {
		ID    int    `db:"id"`
		Skip  string `db:"-"`
		Label string `db:"label"`
	}

	info, err := getFieldInfo(reflect.TypeFor[Product]())
	if err != nil {
		t.Fatalf("getFieldInfo error: %v", err)
	}
	if _, ok := info.fields["skip"]; ok {
		t.Error("field with db:\"-\" should not appear in field map")
	}
	if _, ok := info.fields["-"]; ok {
		t.Error("field with db:\"-\" should not appear as \"-\" key")
	}
}

func TestRowToStruct_UnknownColumnDiscarded(t *testing.T) {
	type Simple struct {
		ID int `db:"id"`
	}

	row := &mockRow{
		cols:   []string{"id", "unknown_col"},
		values: []any{7, "ignored"},
	}

	got, err := RowToStruct[Simple](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("expected ID=7, got %d", got.ID)
	}
}

func TestRowToStruct_NestedStruct(t *testing.T) {
	type Address struct {
		City string `db:"city"`
	}
	type Person struct {
		Name    string  `db:"name"`
		Address Address `db:"address"`
	}

	// Pastikan key map terbentuk dengan benar.
	info, err := getFieldInfo(reflect.TypeFor[Person]())
	if err != nil {
		t.Fatalf("getFieldInfo error: %v", err)
	}
	if _, ok := info.fields["address.city"]; !ok {
		t.Error("expected key \"address.city\" in field map")
	}

	// Pastikan nilai aktual terscan dengan benar.
	row := &mockRow{
		cols:   []string{"name", "address.city"},
		values: []any{"Bob", "Jakarta"},
	}
	got, err := RowToStruct[Person](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Bob" || got.Address.City != "Jakarta" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestRowToStruct_NestedPointerStruct(t *testing.T) {
	type Address struct {
		City string `db:"city"`
	}
	type Person struct {
		Name    string   `db:"name"`
		Address *Address `db:"address"`
	}

	row := &mockRow{
		cols:   []string{"name", "address.city"},
		values: []any{"Alice", "Bandung"},
	}
	got, err := RowToStruct[Person](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected Name=Alice, got %q", got.Name)
	}
	if got.Address == nil {
		t.Fatal("expected Address to be auto-allocated, got nil")
	}
	if got.Address.City != "Bandung" {
		t.Errorf("expected City=Bandung, got %q", got.Address.City)
	}
}

func TestRowToStruct_JsonSliceField(t *testing.T) {
	type Row struct {
		Tags JsonSlice[string] `db:"tags"`
	}

	row := &mockRow{
		cols:   []string{"tags"},
		values: []any{`["a","b","c"]`},
	}

	got, err := RowToStruct[Row](row)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Tags.Len() != 3 || got.Tags[0] != "a" {
		t.Errorf("unexpected Tags: %v", got.Tags)
	}
}

func TestRowToStruct_ScanError(t *testing.T) {
	type Simple struct {
		ID int `db:"id"`
	}

	row := &mockRow{
		cols:   []string{"id"},
		values: []any{1},
		err:    errors.New("db connection lost"),
	}

	_, err := RowToStruct[Simple](row)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, row.err) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestRowToStruct_NonStructError(t *testing.T) {
	row := &mockRow{cols: []string{"x"}, values: []any{1}}
	_, err := RowToStruct[int](row)
	if err == nil {
		t.Fatal("expected error for non-struct type, got nil")
	}
}

// --- buildFieldMap ---

func TestBuildFieldMap_UnexportedSkipped(t *testing.T) {
	type S struct {
		Exported   int
		unexported string //nolint:unused
	}
	out := map[string][]int{}
	if err := buildFieldMap(reflect.TypeFor[S](), "", nil, out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["unexported"]; ok {
		t.Error("unexported field should be skipped")
	}
	if _, ok := out["exported"]; !ok {
		t.Error("exported field should be present")
	}
}

func TestBuildFieldMap_PointerToStruct(t *testing.T) {
	type S struct {
		ID int `db:"id"`
	}
	out := map[string][]int{}
	if err := buildFieldMap(reflect.TypeFor[*S](), "", nil, out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["id"]; !ok {
		t.Error("expected key \"id\" from pointer-to-struct")
	}
}

// --- implementsSqlScanner ---

func TestImplementsSqlScanner_JsonSlice(t *testing.T) {
	t.Run("value_receiver", func(t *testing.T) {
		got := implementsSqlScanner(reflect.TypeFor[JsonSlice[int]]())
		if !got {
			t.Error("JsonSlice should implement sql.Scanner")
		}
	})
	t.Run("pointer_receiver", func(t *testing.T) {
		got := implementsSqlScanner(reflect.TypeFor[*JsonSlice[int]]())
		if !got {
			t.Error("*JsonSlice should implement sql.Scanner")
		}
	})
}

func TestImplementsSqlScanner_PlainStruct(t *testing.T) {
	type S struct{ ID int }
	if implementsSqlScanner(reflect.TypeFor[S]()) {
		t.Error("plain struct should not implement sql.Scanner")
	}
}

// --- isScannableBuiltin ---

func TestIsScannableBuiltin_TimeTime(t *testing.T) {
	if !isScannableBuiltin(reflect.TypeFor[time.Time]()) {
		t.Error("time.Time should be a scannable builtin")
	}
}

func TestIsScannableBuiltin_Unknown(t *testing.T) {
	type MyStruct struct{}
	if isScannableBuiltin(reflect.TypeFor[MyStruct]()) {
		t.Error("unknown struct should not be a scannable builtin")
	}
}

// --- fieldByIndexPath ---

func TestFieldByIndexPath_Simple(t *testing.T) {
	type S struct {
		A int
		B string
	}
	v := reflect.ValueOf(S{A: 10, B: "hello"})
	fv := fieldByIndexPath(v, []int{1})
	if fv.String() != "hello" {
		t.Errorf("expected \"hello\", got %q", fv.String())
	}
}

func TestFieldByIndexPath_NilPointerAllocated(t *testing.T) {
	type Inner struct{ X int }
	type Outer struct{ Inner *Inner }

	v := reflect.New(reflect.TypeFor[Outer]()).Elem()
	fv := fieldByIndexPath(v, []int{0, 0})
	fv.SetInt(99)

	outer := v.Interface().(Outer)
	if outer.Inner == nil || outer.Inner.X != 99 {
		t.Errorf("expected nil pointer to be auto-allocated, got %+v", outer)
	}
}

// --- Benchmark ---

func BenchmarkRowToStruct(b *testing.B) {
	type User struct {
		ID        int    `db:"id"`
		Name      string `db:"name"`
		Email     string `db:"email"`
		CreatedAt string `db:"created_at"`
	}

	row := &mockRow{
		cols:   []string{"id", "name", "email", "created_at"},
		values: []any{1, "Alice", "alice@example.com", "2024-01-01"},
	}

	b.ResetTimer()
	for range b.N {
		_, _ = RowToStruct[User](row)
	}
}
