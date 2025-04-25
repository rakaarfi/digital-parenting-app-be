// internal/utils/random.go
package utils

import (
	"crypto/rand" // Paket Go standar untuk menghasilkan angka acak yang aman secara kriptografis (cryptographically secure random number generator - CSRNG).
	"encoding/base64" // Paket Go standar untuk encoding dan decoding data Base64.
	"fmt"             // Paket Go standar untuk formatting string dan error.
	"math/big"        // Paket Go standar untuk aritmatika angka besar, digunakan di sini untuk pemilihan karakter acak yang tidak bias.
)

// File ini berisi fungsi-fungsi utilitas untuk menghasilkan data acak
// (string, byte) yang aman secara kriptografis. Ini penting untuk
// membuat token, kode unik, secret key, dll.

// ====================================================================================
// Konstanta dan Variabel Global
// ====================================================================================

// charsetAlphanumeric adalah kumpulan karakter yang akan digunakan sebagai basis
// untuk menghasilkan string acak melalui `GenerateRandomString`.
// Karakter yang bisa ambigu (seperti 0, O, 1, l, I) sengaja dihilangkan
// untuk meningkatkan keterbacaan kode yang dihasilkan.
// Anda dapat menyesuaikan charset ini jika diperlukan (misal: hanya huruf besar, hanya angka).
const charsetAlphanumeric = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Menghindari 0, O, 1, I, l

// ====================================================================================
// Fungsi Penghasil String Acak
// ====================================================================================

// GenerateRandomString menghasilkan string acak dengan panjang (`length`) yang ditentukan,
// menggunakan karakter dari `charsetAlphanumeric`. String yang dihasilkan aman secara
// kriptografis karena menggunakan `crypto/rand` sebagai sumber keacakan.
//
// Parameter:
//   - length: Panjang string acak yang diinginkan (harus positif).
//
// Mengembalikan:
//   - string: String acak yang dihasilkan.
//   - error: Error jika `length` tidak positif atau jika terjadi kegagalan saat
//     membaca dari sumber acak (`crypto/rand`).
func GenerateRandomString(length int) (string, error) {
	// Validasi input panjang string.
	if length <= 0 {
		return "", fmt.Errorf("random string length must be a positive integer, got %d", length)
	}

	// Buat byte slice dengan ukuran sesuai panjang yang diminta untuk menampung hasil.
	resultBytes := make([]byte, length)
	// Hitung panjang charset sebagai *big.Int untuk digunakan dalam pemilihan acak.
	charsetLength := big.NewInt(int64(len(charsetAlphanumeric)))

	// Loop sebanyak panjang string yang diinginkan.
	for i := range resultBytes {
		// Dapatkan angka acak yang aman dalam rentang [0, charsetLength).
		// Menggunakan `rand.Int` dari `crypto/rand` memastikan keacakan yang kuat.
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			// Error ini jarang terjadi, tetapi penting untuk ditangani jika sumber acak sistem bermasalah.
			return "", fmt.Errorf("failed to generate cryptographically secure random number: %w", err)
		}
		// Pilih karakter dari `charsetAlphanumeric` pada indeks acak yang didapat,
		// dan tempatkan di slice hasil. `Int64()` aman digunakan karena `charsetLength` kecil.
		resultBytes[i] = charsetAlphanumeric[randomIndex.Int64()]
	}

	// Konversi byte slice hasil menjadi string dan kembalikan.
	return string(resultBytes), nil
}

// ====================================================================================
// Fungsi Penghasil Byte Acak
// ====================================================================================

// GenerateRandomBytes menghasilkan slice byte (`[]byte`) acak yang aman secara kriptografis
// dengan panjang (`length`) yang ditentukan.
// Fungsi ini berguna untuk menghasilkan data biner acak seperti secret key, salt,
// atau initialization vector (IV) untuk enkripsi.
//
// Parameter:
//   - length: Jumlah byte acak yang diinginkan (harus positif).
//
// Mengembalikan:
//   - []byte: Slice byte yang berisi data acak.
//   - error: Error jika `length` tidak positif atau jika terjadi kegagalan saat
//     membaca dari sumber acak (`crypto/rand`).
func GenerateRandomBytes(length int) ([]byte, error) {
	// Validasi input panjang byte.
	if length <= 0 {
		return nil, fmt.Errorf("random byte length must be a positive integer, got %d", length)
	}

	// Buat byte slice dengan ukuran yang diminta.
	randomBytes := make([]byte, length)

	// Isi seluruh slice `randomBytes` dengan byte acak yang aman secara kriptografis.
	// `rand.Read` akan mengembalikan error jika tidak bisa membaca jumlah byte yang diminta.
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Tangani error jika sumber acak sistem bermasalah.
		return nil, fmt.Errorf("failed to read cryptographically secure random bytes: %w", err)
	}

	// Kembalikan slice byte yang sudah terisi data acak.
	return randomBytes, nil
}

// ====================================================================================
// Fungsi Penghasil String Base64 Acak
// ====================================================================================

// GenerateRandomBase64String menghasilkan string acak yang aman secara kriptografis
// dan di-encode menggunakan Base64 URL-safe encoding (tanpa padding).
// Fungsi ini pertama-tama menghasilkan `numBytes` byte acak, kemudian meng-encode-nya.
// Panjang string hasil akan sedikit lebih besar dari `numBytes` karena proses encoding Base64
// (sekitar 4/3 kali `numBytes`).
// String ini cocok digunakan untuk token API, session ID, atau nilai acak lain yang perlu
// aman untuk disertakan dalam URL atau header HTTP.
//
// Parameter:
//   - numBytes: Jumlah byte acak yang akan dihasilkan sebelum di-encode (harus positif).
//
// Mengembalikan:
//   - string: String acak yang sudah di-encode Base64 URL-safe.
//   - error: Error jika `numBytes` tidak positif atau jika terjadi kegagalan saat
//     menghasilkan byte acak awal.
func GenerateRandomBase64String(numBytes int) (string, error) {
	// Hasilkan byte acak terlebih dahulu.
	randomBytes, err := GenerateRandomBytes(numBytes)
	if err != nil {
		// Propagasi error dari GenerateRandomBytes.
		return "", err
	}

	// Encode byte acak menjadi string Base64.
	// `RawURLEncoding` digunakan karena:
	// 1. URL-safe: Menggunakan karakter '-' dan '_' sebagai pengganti '+' dan '/'.
	// 2. Raw: Menghilangkan karakter padding '=' di akhir string, membuatnya lebih ringkas.
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// Contoh penggunaan (dapat dihapus atau dipindahkan ke unit test):
/*
import "log"

func main() {
	// Contoh GenerateRandomString
	codeLength := 10
	randomCode, err := GenerateRandomString(codeLength)
	if err != nil {
		log.Fatalf("Error generating random string: %v", err)
	}
	fmt.Printf("Generated %d-character random code: %s\n", codeLength, randomCode)

	// Contoh GenerateRandomBase64String
	byteLength := 32 // Menghasilkan sekitar 43 karakter Base64
	randomSecret, err := GenerateRandomBase64String(byteLength)
	if err != nil {
		log.Fatalf("Error generating random base64 secret: %v", err)
	}
	fmt.Printf("Generated %d-byte random Base64 secret: %s\n", byteLength, randomSecret)
}
*/
