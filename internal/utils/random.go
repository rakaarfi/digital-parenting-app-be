// internal/utils/random.go
package utils

import (
	"crypto/rand" // Paket untuk random number generator yang aman secara kriptografis
	"encoding/base64" // Untuk encoding yang lebih ramah URL/string
	"fmt"
	"math/big" // Untuk pemilihan karakter acak
)

// charsetAlphanumeric adalah kumpulan karakter yang akan digunakan untuk kode acak.
// Menghilangkan karakter yang bisa ambigu (seperti 0, O, 1, l, I).
// Anda bisa menyesuaikan ini jika perlu (misal, hanya huruf besar, hanya angka).
const charsetAlphanumeric = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Menghindari 0, O, 1, I, l

// GenerateRandomString menghasilkan string acak yang aman secara kriptografis
// dengan panjang yang ditentukan, menggunakan karakter dari charsetAlphanumeric.
func GenerateRandomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("random string length must be positive")
	}

	// Buat byte slice untuk menampung hasil acak
	result := make([]byte, length)
	// Hitung panjang charset
	charsetLen := big.NewInt(int64(len(charsetAlphanumeric)))

	for i := range result {
		// Dapatkan angka acak yang aman dalam rentang [0, charsetLen)
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Error saat membaca dari crypto/rand (jarang terjadi tapi penting ditangani)
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		// Pilih karakter dari charset berdasarkan angka acak
		result[i] = charsetAlphanumeric[num.Int64()]
	}

	// Kembalikan string hasil
	return string(result), nil
}

// GenerateRandomBytes menghasilkan slice byte acak dengan panjang tertentu.
// Berguna untuk membuat secret key atau token lain.
func GenerateRandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("byte length must be positive")
	}
	b := make([]byte, length)
	_, err := rand.Read(b) // Isi slice b dengan byte acak
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomBase64String menghasilkan string acak yang di-encode Base64 (URL safe).
// Panjang string hasil akan sedikit lebih besar dari numBytes karena encoding.
func GenerateRandomBase64String(numBytes int) (string, error) {
	b, err := GenerateRandomBytes(numBytes)
	if err != nil {
		return "", err
	}
	// Gunakan URLEncoding agar aman untuk URL (mengganti '+' dan '/')
	// RawURLEncoding menghilangkan padding '=' di akhir.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Contoh penggunaan GenerateRandomString (bisa Anda coba di test atau main sementara):
/*
func main() {
    code, err := utils.GenerateRandomString(16) // Buat kode 16 karakter
    if err != nil {
        log.Fatalf("Error generating code: %v", err)
    }
    fmt.Println("Generated Code:", code)

	secret, err := utils.GenerateRandomBase64String(32) // Buat secret ~43 karakter base64
    if err != nil {
        log.Fatalf("Error generating secret: %v", err)
    }
	fmt.Println("Generated Base64 Secret:", secret)

}
*/