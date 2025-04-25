// internal/utils/hash.go
package utils

import (
	"fmt" // Paket Go standar untuk formatting error.

	"golang.org/x/crypto/bcrypt" // Paket Go standar (sub-repositori) yang menyediakan implementasi algoritma hashing password bcrypt.
)

// File ini berisi fungsi-fungsi utilitas yang berkaitan dengan hashing
// dan verifikasi password menggunakan algoritma bcrypt. Bcrypt dipilih
// karena dirancang khusus untuk hashing password, tahan terhadap serangan
// brute-force dengan memasukkan 'cost factor' (work factor) dan salt acak
// secara otomatis.

// ====================================================================================
// Fungsi Hashing Password
// ====================================================================================

// HashPassword menghasilkan hash bcrypt dari string password plaintext yang diberikan.
// Fungsi ini menggunakan 'cost factor' default yang disediakan oleh paket `bcrypt`
// (`bcrypt.DefaultCost`), yang merupakan keseimbangan yang baik antara keamanan
// (membuat hashing cukup lambat untuk penyerang) dan performa (tidak terlalu lambat
// untuk pengguna saat login atau registrasi).
// Salt acak akan dibuat secara otomatis dan disertakan dalam string hash yang dihasilkan.
//
// Parameter:
//   - password: String password plaintext yang akan di-hash.
//
// Mengembalikan:
//   - string: String hash bcrypt yang dihasilkan (termasuk salt dan cost factor).
//   - error: Error jika terjadi masalah selama proses hashing (jarang terjadi,
//     biasanya karena masalah sistem atau password yang sangat panjang).
func HashPassword(password string) (string, error) {
	// `bcrypt.GenerateFromPassword` adalah fungsi utama untuk membuat hash.
	// Ia menerima password sebagai byte slice dan cost factor.
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Jika terjadi error, kembalikan string kosong dan error asli yang dibungkus.
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}
	// Konversi hasil hash (byte slice) menjadi string dan kembalikan.
	return string(hashedBytes), nil
}

// ====================================================================================
// Fungsi Verifikasi Password
// ====================================================================================

// CheckPasswordHash membandingkan string password plaintext yang diberikan oleh pengguna
// dengan string hash bcrypt yang sebelumnya disimpan di database.
// Fungsi ini secara otomatis mengekstrak salt dan cost factor dari string `hash`
// yang tersimpan, kemudian melakukan hashing pada `password` plaintext dengan parameter
// yang sama, dan membandingkan hasilnya secara aman (constant-time comparison)
// untuk mencegah timing attacks.
//
// Parameter:
//   - password: String password plaintext yang ingin diverifikasi (misal: dari input login).
//   - hash: String hash bcrypt yang tersimpan di database untuk pengguna tersebut.
//
// Mengembalikan:
//   - bool: `true` jika password plaintext cocok dengan hash, `false` jika tidak cocok.
//     Fungsi ini mengembalikan `false` jika terjadi error internal saat perbandingan
//     (misalnya format hash tidak valid), karena password dianggap tidak cocok jika
//     proses verifikasi gagal.
func CheckPasswordHash(password, hash string) bool {
	// `bcrypt.CompareHashAndPassword` adalah cara yang aman dan direkomendasikan
	// untuk memverifikasi password terhadap hash bcrypt.
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	// Fungsi ini mengembalikan `nil` jika password cocok, dan error jika tidak cocok
	// atau jika format hash tidak valid. Oleh karena itu, kita cukup memeriksa apakah `err == nil`.
	return err == nil
}
