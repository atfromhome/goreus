package pgxscan

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

// fieldInfo menyimpan hasil parsing reflect satu tipe struct.
// Di-cache agar parsing hanya terjadi sekali per tipe selama lifetime aplikasi.
type fieldInfo struct {
	// key   = nama kolom DB (misal "order_id", "address.city" untuk nested)
	// value = urutan index field untuk sampai ke field tersebut via reflect
	fields map[string][]int
}

// fieldCache menyimpan *fieldInfo per reflect.Type.
// sync.Map dipilih karena read-heavy: setelah warm-up, hampir selalu cache hit.
var fieldCache sync.Map

// RowToStruct memetakan satu baris hasil query pgx ke struct T.
// Nama kolom dari query dicocokkan ke field struct berdasarkan tag `db` atau
// konversi snake_case otomatis. Field yang tidak cocok dengan kolom manapun dibuang.
func RowToStruct[T any](row pgx.CollectableRow) (T, error) {
	var result T

	// FieldDescriptions mengembalikan metadata kolom tanpa membaca data baris.
	descs := row.FieldDescriptions()
	colNames := make([]string, len(descs))
	for i, d := range descs {
		colNames[i] = string(d.Name)
	}

	// Ambil peta kolom→field untuk tipe T, dari cache jika sudah ada.
	rv := reflect.ValueOf(&result).Elem()
	info, err := getFieldInfo(rv.Type())
	if err != nil {
		var zero T
		return zero, err
	}

	// Bangun slice pointer tujuan scan, satu elemen per kolom.
	// Kolom yang tidak dikenal diarahkan ke variabel buang agar Scan tidak gagal.
	dest := make([]any, len(colNames))
	for i, col := range colNames {
		idxPath, ok := info.fields[col]
		if !ok {
			var discard any
			dest[i] = &discard
			continue
		}
		dest[i] = fieldByIndexPath(rv, idxPath).Addr().Interface()
	}

	if err := row.Scan(dest...); err != nil {
		var zero T
		return zero, fmt.Errorf("RowToStruct scan: %w", err)
	}

	return result, nil
}

// getFieldInfo mengembalikan fieldInfo untuk tipe t.
// Jika belum ada di cache, buildFieldMap dipanggil sekali lalu hasilnya disimpan.
func getFieldInfo(t reflect.Type) (*fieldInfo, error) {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.(*fieldInfo), nil
	}

	info := &fieldInfo{
		fields: make(map[string][]int),
	}

	if err := buildFieldMap(t, "", []int{}, info.fields); err != nil {
		return nil, err
	}

	fieldCache.Store(t, info)
	return info, nil
}

// buildFieldMap mengisi out dengan pemetaan nama kolom → index path field.
// Dipanggil rekursif untuk nested struct dengan prefix bertitik (misal "address.city").
func buildFieldMap(t reflect.Type, prefix string, path []int, out map[string][]int) error {
	// Dereference *T agar bisa dipakai dengan pointer-to-struct.
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("pgxscan: expected struct, got %s", t.Kind())
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fType := field.Type

		// Field unexported tidak bisa di-set via reflect; lewati.
		if !field.IsExported() {
			continue
		}

		dbTag := field.Tag.Get("db")
		// Tag db:"-" adalah sinyal eksplisit untuk tidak memetakan field ini.
		if dbTag == "-" {
			continue
		}

		// Gunakan tag db jika ada, jika tidak konversi nama field ke snake_case.
		colName := dbTag
		if colName == "" {
			colName = toSnakeCase(field.Name)
		}

		// Untuk nested struct, key berformat "parent.child" agar unik secara global.
		fullKey := colName
		if prefix != "" {
			fullKey = prefix + "." + colName
		}

		// Salin path agar setiap field punya slice-nya sendiri, bukan referensi bersama.
		currentPath := append(append([]int{}, path...), i)

		// Dereference pointer type untuk inspeksi kind-nya.
		actualType := fType
		if actualType.Kind() == reflect.Pointer {
			actualType = actualType.Elem()
		}

		// Jika field mengimplementasikan sql.Scanner, pgx akan memanggilnya langsung;
		// tidak perlu rekursi meski field adalah struct (contoh: JsonSlice, pgtype.*).
		if implementsSqlScanner(fType) {
			out[fullKey] = currentPath
			continue
		}

		// Struct yang bukan scannable builtin (misal time.Time, net.IP) direkursi
		// sehingga field-field di dalamnya bisa dipetakan ke kolom individual.
		if actualType.Kind() == reflect.Struct && !isScannableBuiltin(actualType) {
			if err := buildFieldMap(actualType, fullKey, currentPath, out); err != nil {
				return err
			}
			continue
		}

		out[fullKey] = currentPath
	}

	return nil
}

// fieldByIndexPath mengikuti urutan index untuk sampai ke field yang dituju.
// Jika ada pointer di tengah path yang nilnya nil, pointer tersebut dialokasikan
// secara otomatis agar tidak panic saat field di-set.
func fieldByIndexPath(v reflect.Value, path []int) reflect.Value {
	for _, idx := range path {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(idx)
	}
	return v
}

// sqlScannerType adalah representasi reflect dari interface sql.Scanner,
// dipakai untuk mengecek apakah suatu tipe mengimplementasikannya.
var sqlScannerType = reflect.TypeFor[sql.Scanner]()

// implementsSqlScanner mengembalikan true jika t atau *t mengimplementasikan sql.Scanner.
// Pointer receiver dicek lebih dulu karena method Scan hampir selalu didefinisikan di *T.
func implementsSqlScanner(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(sqlScannerType) || t.Implements(sqlScannerType)
}

// scannableBuiltins adalah whitelist tipe struct yang harus diperlakukan sebagai scalar.
// Tipe-tipe ini sudah diketahui pgx cara men-scan-nya, sehingga tidak boleh direkursi
// oleh buildFieldMap meski kind-nya adalah struct.
// Key berformat "<pkg-last-segment>.<TypeName>" untuk O(1) lookup tanpa alokasi string.
var scannableBuiltins = map[string]struct{}{
	"time.Time":          {},
	"net.IP":             {},
	"net.IPNet":          {},
	"netip.Addr":         {},
	"pgtype.Text":        {},
	"pgtype.Int4":        {},
	"pgtype.Int8":        {},
	"pgtype.Float8":      {},
	"pgtype.Bool":        {},
	"pgtype.Timestamptz": {},
	"pgtype.Date":        {},
	"pgtype.UUID":        {},
	"pgtype.Numeric":     {},
	"pgtype.JSONB":       {},
}

// isScannableBuiltin mengecek apakah t ada di whitelist scannableBuiltins.
// PkgPath mengandung full module path (misal "github.com/jackc/pgx/v5/pgtype"),
// sehingga hanya segmen terakhir setelah "/" yang diambil sebagai prefix key.
func isScannableBuiltin(t reflect.Type) bool {
	pkg := t.PkgPath()
	if idx := strings.LastIndexByte(pkg, '/'); idx >= 0 {
		pkg = pkg[idx+1:]
	}
	_, ok := scannableBuiltins[pkg+"."+t.Name()]
	return ok
}
