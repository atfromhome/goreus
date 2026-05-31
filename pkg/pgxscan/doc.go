// Package pgxscan menyediakan utilitas untuk memetakan hasil query pgx ke Go struct
// secara otomatis tanpa perlu menulis row.Scan(&a, &b, &c, ...) secara manual.
//
// # RowToStruct
//
// Fungsi utama package ini adalah [RowToStruct], yang menerima [pgx.CollectableRow]
// dan mengisi struct T berdasarkan nama kolom hasil query.
//
// Nama kolom dipetakan ke field struct melalui aturan berikut (prioritas dari atas):
//   - Tag `db:"nama_kolom"` pada field
//   - Konversi otomatis nama field ke snake_case (misal: CreatedAt → created_at)
//   - Tag `db:"-"` untuk mengecualikan field dari pemetaan
//
// Field yang merupakan nested struct akan direkursi dengan prefix bertitik,
// sehingga kolom "address.city" memetakan ke field Address.City.
//
// # JsonSlice
//
// [JsonSlice] adalah tipe generik untuk kolom PostgreSQL yang berisi JSON array.
// Ia mengimplementasikan [sql.Scanner], [json.Marshaler], dan [json.Unmarshaler]
// sehingga dapat digunakan langsung sebagai field struct dan di-scan oleh RowToStruct.
//
// # Contoh
//
//	type User struct {
//	    ID        int               `db:"id"`
//	    Name      string            `db:"name"`
//	    Tags      pgxscan.JsonSlice[string] `db:"tags"`
//	    CreatedAt time.Time
//	}
//
//	rows, _ := pool.Query(ctx, "SELECT id, name, tags, created_at FROM users")
//	users, err := pgx.CollectRows(rows, pgxscan.RowToStruct[User])
//
// # Cache
//
// Pemetaan field struct di-cache per tipe menggunakan [sync.Map] sehingga
// overhead reflect hanya terjadi sekali per tipe sepanjang lifetime aplikasi.
package pgxscan
