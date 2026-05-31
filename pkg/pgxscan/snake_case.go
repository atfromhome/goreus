package pgxscan

import (
	"strings"
	"unicode"
)

// toSnakeCase mengonversi nama field Go (PascalCase/camelCase) ke snake_case
// untuk dipakai sebagai nama kolom DB secara otomatis.
//
// Aturan penyisipan underscore:
//   - Huruf kapital didahului huruf kecil → "camelCase" → "camel_case"
//   - Huruf kapital didahului digit → "user1Name" → "user1_name"
//   - Huruf kapital dalam rangkaian akronim diikuti huruf kecil → "HTMLParser" → "html_parser"
//
// Contoh: UserID → user_id, GetHTTPSResponse → get_https_response, User1Name → user1_name
func toSnakeCase(s string) string {
	runes := []rune(s)
	n := len(runes)

	var b strings.Builder
	// Alokasi awal dengan estimasi panjang output + ruang untuk underscore.
	b.Grow(n + n/2)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				var next rune
				if i+1 < n {
					next = runes[i+1]
				}
				// Sisipkan underscore dalam tiga situasi:
				// 1. Transisi huruf kecil→kapital: "camel|Case" → "camel_case"
				// 2. Transisi digit→kapital: "user1|Name" → "user1_name"
				// 3. Akhir akronim sebelum huruf kecil: "HTM|L|Parser" → "html_parser"
				//    (kapital didahului kapital tapi diikuti huruf kecil)
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && unicode.IsLower(next)) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}
