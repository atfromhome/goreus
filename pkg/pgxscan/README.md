# pgxscan

Paket `pgxscan` menyediakan utilitas untuk memetakan hasil query [pgx v5](https://github.com/jackc/pgx) ke Go struct secara otomatis, tanpa menulis `row.Scan(&a, &b, &c, ...)` secara manual.

## Fitur

- Pemetaan kolom otomatis berdasarkan tag `db` atau konversi snake_case dari nama field
- Dukungan nested struct dengan dot notation di SQL (`"user.id"`, `"user.name"`)
- `JsonSlice[T]` untuk kolom JSON array hasil `jsonb_agg` — fully typed via generics
- Cache hasil reflect per tipe — overhead hanya terjadi sekali selama lifetime aplikasi

## Instalasi

```bash
go get github.com/atfromhome/goreus
```

## Penggunaan

### RowToStruct

`RowToStruct[T]` menerima `pgx.CollectableRow` dan mengisi struct `T` berdasarkan
nama kolom hasil query. Kompatibel dengan `pgx.CollectRows` dan `pgx.CollectOneRow`.

```go
import (
    "github.com/atfromhome/goreus/pkg/pgxscan"
    "github.com/jackc/pgx/v5"
)

type User struct {
    ID        int       `db:"id"`
    Name      string    `db:"name"`
    Email     string    `db:"email"`
    CreatedAt time.Time `db:"created_at"`
}

// Satu baris
user, err := pgx.CollectOneRow(rows, pgxscan.RowToStruct[User])

// Banyak baris
users, err := pgx.CollectRows(rows, pgxscan.RowToStruct[User])
```

### Aturan Pemetaan Kolom

| Kondisi field       | Nama kolom yang dipakai                |
|---------------------|----------------------------------------|
| Tag `db:"nama"`     | `nama`                                 |
| Tanpa tag           | Nama field dikonversi ke snake_case    |
| Tag `db:"-"`        | Field diabaikan                        |
| Nested struct       | Rekursi dengan prefix `"parent.child"` |

### Nested Struct

Untuk nested struct, kolom di SQL menggunakan dot notation dalam double quote.
Kolom flat tetap menggunakan underscore biasa.

```go
type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

type Order struct {
    OrderID     int     `db:"order_id"`
    Status      string  `db:"status"`
    FinalAmount float64 `db:"final_amount"`
    User        User    `db:"user"` // → kolom "user.id", "user.name", "user.email"
}
```

```sql
SELECT
    o.id            AS order_id,
    o.status,
    o.final_amount,
    u.id            AS "user.id",
    u.name          AS "user.name",
    u.email         AS "user.email"
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.id = $1
```

### JsonSlice

`JsonSlice[T]` adalah tipe generik untuk kolom PostgreSQL yang berisi JSON array.
Biasanya dipakai untuk hasil `jsonb_agg` pada relasi one-to-many.

Mengimplementasikan `sql.Scanner`, `driver.Valuer`, `json.Marshaler`, dan `json.Unmarshaler`.

```go
type OrderItem struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Quantity int    `json:"quantity"`
}

type Order struct {
    OrderID     int                            `db:"order_id"`
    Status      string                         `db:"status"`
    FinalAmount float64                        `db:"final_amount"`
    User        User                           `db:"user"`
    Items       pgxscan.JsonSlice[OrderItem]   `db:"items"`
}
```

```sql
SELECT
    o.id            AS order_id,
    o.status,
    o.final_amount,
    u.id            AS "user.id",
    u.name          AS "user.name",
    u.email         AS "user.email",
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id',       oi.id,
                'name',     p.name,
                'quantity', oi.quantity
            ) ORDER BY oi.id
        ) FILTER (WHERE oi.id IS NOT NULL),
        '[]'::jsonb
    ) AS items
FROM orders o
JOIN users u             ON u.id = o.user_id
LEFT JOIN order_items oi ON oi.order_id = o.id
LEFT JOIN products p     ON p.id = oi.product_id
WHERE o.id = $1
GROUP BY o.id, u.id, u.name, u.email
```

```go
// Akses elemen — fully typed
for _, item := range order.Items {
    fmt.Println(item.Name)
}

// Serialisasi ke JSON — nil menghasilkan "[]", bukan "null"
b, _ := json.Marshal(order)

// Dipakai sebagai parameter query (driver.Valuer)
tags := pgxscan.JsonSlice[string]{"go", "postgres"}
pool.Exec(ctx, "UPDATE products SET tags = $1 WHERE id = $2", tags, id)
```

### Perilaku nil dan null pada JsonSlice

| Nilai kolom DB   | Hasil `JsonSlice` |
|------------------|-------------------|
| `NULL` (SQL)     | `[]T{}`           |
| `"null"` (JSON)  | `[]T{}`           |
| `"[]"` (JSON)    | `[]T{}`           |
| `"[1,2,3]"`      | `[]T{1, 2, 3}`    |

### Pagination dengan Nested dan Array

Untuk list endpoint dengan pagination, total count bisa disertakan dalam query
yang sama menggunakan CTE:

```go
type OrderRow struct {
    OrderID     int                          `db:"order_id"`
    Status      string                       `db:"status"`
    FinalAmount float64                      `db:"final_amount"`
    User        User                         `db:"user"`
    Items       pgxscan.JsonSlice[OrderItem] `db:"items"`
    TotalCount  int                          `db:"total_count"`
}
```

```sql
WITH filtered AS (
    SELECT id FROM orders
    WHERE status = $1
    ORDER BY created_at DESC
    LIMIT $2 OFFSET $3
),
total AS (
    SELECT COUNT(*) AS total_count
    FROM orders WHERE status = $1
)
SELECT
    o.id            AS order_id,
    o.status,
    o.final_amount,
    u.id            AS "user.id",
    u.name          AS "user.name",
    u.email         AS "user.email",
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id',       oi.id,
                'name',     p.name,
                'quantity', oi.quantity
            ) ORDER BY oi.id
        ) FILTER (WHERE oi.id IS NOT NULL),
        '[]'::jsonb
    )               AS items,
    (SELECT total_count FROM total) AS total_count
FROM filtered f
JOIN orders o        ON o.id = f.id
JOIN users u         ON u.id = o.user_id
LEFT JOIN order_items oi ON oi.order_id = o.id
LEFT JOIN products p     ON p.id = oi.product_id
GROUP BY o.id, o.status, o.final_amount, u.id, u.name, u.email
ORDER BY o.created_at DESC
```

```go
rows, err := pool.Query(ctx, query, status, limit, offset)
if err != nil {
    return nil, err
}

orderRows, err := pgx.CollectRows(rows, pgxscan.RowToStruct[OrderRow])
if err != nil {
    return nil, err
}

totalCount := 0
if len(orderRows) > 0 {
    totalCount = orderRows[0].TotalCount
}
```

## Batasan

- Hanya mendukung PostgreSQL via pgx v5
- Nested struct didukung di segala kedalaman (`a.b.c`, dst.) selama alias SQL menggunakan dot notation dalam double quote (`"a.b.c"`)
- Array of struct tidak didukung secara native — gunakan `JsonSlice[T]`
- Kolom tidak ada di struct di-discard tanpa error

## Tipe yang Dikenali sebagai Scalar

Tipe struct berikut tidak direkursi dan diserahkan langsung ke pgx untuk di-scan:

| Package    | Tipe                                                              |
|------------|-------------------------------------------------------------------|
| `time`     | `Time`                                                            |
| `net`      | `IP`, `IPNet`                                                     |
| `netip`    | `Addr`                                                            |
| `pgtype`   | `Text`, `Int4`, `Int8`, `Float8`, `Bool`, `Timestamptz`, `Date`, `UUID`, `Numeric`, `JSONB` |

Tipe yang mengimplementasikan `sql.Scanner` juga otomatis diperlakukan sebagai scalar.

## Konvensi SQL

| Jenis relasi    | Konvensi SQL                          | Contoh                        |
|-----------------|---------------------------------------|-------------------------------|
| Flat kolom      | Underscore biasa                      | `o.id AS order_id`            |
| Nested one-one  | Dot notation dalam double quote       | `u.id AS "user.id"`           |
| One-to-many     | `jsonb_agg` + `jsonb_build_object`    | `jsonb_agg(...) AS items`     |
