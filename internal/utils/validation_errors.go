// internal/utils/validation_errors.go
package utils

import (
	"fmt" // Paket Go standar untuk formatting string.

	"github.com/go-playground/validator/v10" // Library populer untuk validasi struct di Go.
)

// File ini berisi fungsi utilitas untuk memformat error validasi yang dihasilkan
// oleh library `go-playground/validator/v10` menjadi format yang lebih ramah
// untuk ditampilkan dalam response API.

// ====================================================================================
// Fungsi Pemformatan Error Validasi
// ====================================================================================

// FormatValidationErrors menerima error generik dan memeriksa apakah itu adalah
// error validasi dari `validator.ValidationErrors`. Jika ya, fungsi ini akan
// mengiterasi setiap field yang gagal validasi dan membuat map[string]string
// yang berisi nama field sebagai kunci dan pesan error validasi sebagai nilai.
// Jika error yang diterima bukan `validator.ValidationErrors`, fungsi ini akan
// mengembalikan map dengan satu entri error generik.
//
// Map yang dihasilkan cocok untuk disertakan dalam response JSON API (misalnya,
// dalam field 'errors' atau 'details') untuk memberikan feedback yang jelas
// kepada klien tentang field mana yang perlu diperbaiki.
//
// Parameter:
//   - err: Error yang diterima (biasanya dari `c.BodyParser()` atau validasi manual).
//
// Mengembalikan:
//   - map[string]string: Map yang berisi detail error validasi per field,
//     atau map dengan error generik jika input `err` bukan `validator.ValidationErrors`.
func FormatValidationErrors(err error) map[string]string {
	// Buat map kosong untuk menampung hasil format error.
	errorsMap := make(map[string]string)

	// Lakukan type assertion untuk memeriksa apakah `err` adalah tipe `validator.ValidationErrors`.
	// `validator.ValidationErrors` adalah slice dari `FieldError`.
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		// Jika type assertion berhasil (ini adalah error validasi dari validator):
		// Iterasi melalui setiap `FieldError` dalam slice `validationErrors`.
		for _, fieldErr := range validationErrors {
			// Dapatkan nama field yang gagal validasi.
			fieldName := fieldErr.Field()
			// Dapatkan tag validasi yang gagal (misal: "required", "min", "email").
			tag := fieldErr.Tag()

			// Buat pesan error yang deskriptif.
			// Anda bisa membuat pesan yang lebih spesifik dan user-friendly di sini
			// berdasarkan kombinasi `fieldName` dan `tag`.
			// Contoh:
			// switch tag {
			// case "required":
			// 	errorsMap[fieldName] = fmt.Sprintf("Field '%s' wajib diisi.", fieldName)
			// case "email":
			// 	errorsMap[fieldName] = fmt.Sprintf("Field '%s' harus berupa alamat email yang valid.", fieldName)
			// case "min":
			// 	errorsMap[fieldName] = fmt.Sprintf("Field '%s' harus memiliki panjang minimal %s karakter.", fieldName, fieldErr.Param())
			// default:
			// 	errorsMap[fieldName] = fmt.Sprintf("Validasi field '%s' gagal pada aturan '%s'.", fieldName, tag)
			// }

			// Pesan error default yang lebih sederhana:
			errorsMap[fieldName] = fmt.Sprintf("Validation for field '%s' failed on the '%s' rule.", fieldName, tag)
		}
	} else {
		// Jika `err` bukan `validator.ValidationErrors` (misalnya error parsing JSON, dll.):
		// Kembalikan pesan error generik. Sebaiknya log error asli `err` di tempat lain (handler).
		errorsMap["error"] = "Invalid input data or incorrect format."
	}

	// Kembalikan map yang berisi pesan error yang sudah diformat.
	return errorsMap
}
