package pgxscan

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JsonSlice adalah slice generik untuk kolom PostgreSQL yang berisi JSON array.
// Mengimplementasikan sql.Scanner sehingga dapat di-scan langsung oleh RowToStruct,
// serta json.Marshaler dan json.Unmarshaler untuk serialisasi di layer HTTP.
type JsonSlice[T any] []T //nolint:revive,stylecheck // intentional naming

// Scan mengisi JsonSlice dari nilai kolom DB yang dikirim pgx.
// pgx mengirim JSON array sebagai []byte atau string; keduanya didukung.
// nil dan "null" JSON diperlakukan sebagai slice kosong, bukan nil,
// agar caller tidak perlu cek nil setelah scan.
func (j *JsonSlice[T]) Scan(src any) error {
	if src == nil {
		*j = []T{}
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("JsonSlice: cannot scan type %T", src)
	}

	// JSON null dari kolom nullable diperlakukan sama dengan nil: slice kosong.
	if bytes.Equal(b, []byte("null")) {
		*j = []T{}
		return nil
	}

	var result []T
	if err := json.Unmarshal(b, &result); err != nil {
		return fmt.Errorf("JsonSlice: unmarshal failed: %w", err)
	}

	*j = result
	return nil
}

// MarshalJSON menghasilkan JSON array. Nil slice menghasilkan "[]" bukan "null"
// agar respons API konsisten meski field belum pernah diisi.
func (j JsonSlice[T]) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(j))
}

// UnmarshalJSON mengisi JsonSlice dari JSON array.
// Konversi eksplisit ke []T dilakukan agar json.Unmarshal tidak masuk infinite loop
// akibat JsonSlice sendiri mengimplementasikan json.Unmarshaler.
func (j *JsonSlice[T]) UnmarshalJSON(b []byte) error {
	var result []T
	if err := json.Unmarshal(b, &result); err != nil {
		return err
	}
	*j = result
	return nil
}

// Value mengimplementasikan driver.Valuer agar JsonSlice bisa dipakai sebagai
// parameter query (INSERT/UPDATE). Nil slice menghasilkan NULL di DB.
func (j JsonSlice[T]) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal([]T(j))
	if err != nil {
		return nil, fmt.Errorf("JsonSlice: marshal failed: %w", err)
	}
	return string(b), nil
}

// ToSlice mengembalikan representasi []T biasa, berguna saat perlu diteruskan
// ke fungsi yang tidak menerima tipe alias.
func (j JsonSlice[T]) ToSlice() []T {
	return []T(j)
}

// Len mengembalikan jumlah elemen. Konsisten dengan len() standar Go.
func (j JsonSlice[T]) Len() int {
	return len(j)
}
