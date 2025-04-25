// internal/utils/jwt.go
package utils

import (
	"fmt"     // Paket standar untuk formatting error dan string.
	"os"      // Paket standar untuk interaksi OS, digunakan untuk membaca environment variable (JWT_SECRET).
	"strconv" // Paket standar untuk konversi string ke integer (ExtractUserIDFromParam).
	"strings" // Paket standar untuk manipulasi string (ExtractToken, EqualFold).
	"time"    // Paket standar untuk fungsionalitas waktu (menentukan waktu kedaluwarsa token).

	"github.com/gofiber/fiber/v2"    // Framework Fiber, digunakan untuk context (c *fiber.Ctx).
	"github.com/golang-jwt/jwt/v5"   // Library populer untuk membuat dan memvalidasi JSON Web Tokens (JWT).
	zlog "github.com/rs/zerolog/log" // Logger global Zerolog yang sudah dikonfigurasi.
)

// File ini berisi fungsi-fungsi utilitas yang berkaitan dengan pembuatan,
// validasi, dan ekstraksi informasi dari JSON Web Tokens (JWT).

// ====================================================================================
// Definisi Struktur Claims JWT
// ====================================================================================

// JwtClaims mendefinisikan struktur data (payload) yang akan disematkan di dalam token JWT.
// Struktur ini mencakup claims standar yang terdaftar dalam spesifikasi JWT (`jwt.RegisteredClaims`)
// serta claims kustom yang spesifik untuk aplikasi ini (UserID, Username, Role).
type JwtClaims struct {
	UserID               int    `json:"user_id"`  // ID unik pengguna yang memiliki token.
	Username             string `json:"username"` // Username pengguna.
	Role                 string `json:"role"`     // Peran pengguna (misal: "Admin", "Parent", "Child").
	jwt.RegisteredClaims        // Menyematkan claims standar JWT (seperti ExpiresAt, IssuedAt, Issuer, dll.).
}

// ====================================================================================
// Variabel Global (Konfigurasi JWT)
// ====================================================================================

// jwtSecret adalah kunci rahasia (secret key) yang digunakan untuk menandatangani (sign)
// dan memverifikasi (verify) token JWT. Kunci ini diambil dari environment variable "JWT_SECRET".
// **PENTING:** Kunci ini HARUS dijaga kerahasiaannya dan idealnya memiliki kompleksitas yang tinggi.
// Variabel ini diinisialisasi saat paket `utils` dimuat pertama kali.
// Pemeriksaan keberadaan variabel ini dilakukan di configs.LoadConfig().
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ====================================================================================
// Fungsi Pembuatan Token JWT
// ====================================================================================

// GenerateJWT membuat string token JWT baru yang ditandatangani (signed) untuk pengguna tertentu.
// Fungsi ini menerima ID, username, dan peran pengguna sebagai input.
// Token yang dihasilkan akan memiliki masa berlaku (expiration time) yang ditentukan.
//
// Parameter:
//   - userID: ID integer pengguna.
//   - username: Username pengguna.
//   - role: Peran pengguna (string).
//
// Mengembalikan:
//   - string: Token JWT yang sudah ditandatangani.
//   - error: Error jika terjadi kegagalan saat proses penandatanganan token.
func GenerateJWT(userID int, username, role string) (string, error) {
	// Tentukan masa berlaku token (misalnya: 72 jam dari waktu sekarang).
	// Durasi ini sebaiknya dapat dikonfigurasi melalui environment variable.
	expirationTime := time.Now().Add(72 * time.Hour) // TODO: Buat durasi ini dapat dikonfigurasi

	// Buat instance JwtClaims dengan data pengguna dan claims standar JWT.
	claims := JwtClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: Waktu kedaluwarsa token. Setelah waktu ini, token tidak valid.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			// IssuedAt: Waktu token ini dibuat.
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// NotBefore: Waktu token mulai berlaku. Biasanya sama dengan IssuedAt.
			NotBefore: jwt.NewNumericDate(time.Now()),
			// Issuer: Pengenal aplikasi/pihak yang mengeluarkan token (opsional).
			Issuer: "digital-parenting-app",
			// Subject: Subjek token, bisa berupa ID pengguna (opsional).
			// Subject: strconv.Itoa(userID),
		},
	}

	// Buat token baru dengan struktur claims yang sudah dibuat dan metode signing HS256 (HMAC SHA-256).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Tandatangani token menggunakan `jwtSecret` yang sudah diinisialisasi.
	// Proses ini menghasilkan string token final yang siap dikirim ke klien.
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		// Jika terjadi error saat signing (misal: secret tidak valid), log error dan kembalikan.
		zlog.Error().Err(err).Msg("Failed to sign JWT token")
		return "", fmt.Errorf("error signing token: %w", err) // Bungkus error asli
	}

	// Log (level debug) bahwa token berhasil dibuat (hindari logging token itu sendiri).
	zlog.Debug().Int("user_id", userID).Str("username", username).Str("role", role).Msg("JWT token generated successfully")
	return signedToken, nil // Kembalikan string token yang sudah ditandatangani
}

// ====================================================================================
// Fungsi Validasi Token JWT
// ====================================================================================

// ValidateJWT memverifikasi token JWT string yang diberikan.
// Fungsi ini melakukan beberapa pemeriksaan:
// 1. Memastikan format token benar.
// 2. Memverifikasi signature token menggunakan `jwtSecret`.
// 3. Memeriksa masa berlaku token (apakah sudah kedaluwarsa).
// 4. Memastikan algoritma signing yang digunakan adalah HS256 (sesuai yang diharapkan).
// 5. Mem-parsing payload (claims) token ke dalam struct `*JwtClaims`.
//
// Parameter:
//   - tokenString: String token JWT yang diterima dari klien.
//
// Mengembalikan:
//   - *JwtClaims: Pointer ke struct claims jika token valid.
//   - error: Error jika token tidak valid (format salah, signature mismatch, kedaluwarsa, algoritma salah, dll.).
func ValidateJWT(tokenString string) (*JwtClaims, error) {
	// Parse token string. Fungsi ini secara otomatis memvalidasi signature dan expiry
	// jika keyFunc mengembalikan secret yang benar.
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// --- Validasi Metode Signing (Sangat Penting!) ---
		// Memastikan token menggunakan algoritma HMAC (seperti HS256) yang kita harapkan.
		// Ini mencegah serangan di mana penyerang mengganti header algoritma menjadi "none"
		// atau algoritma lain yang tidak aman.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// Jika algoritma tidak sesuai, log peringatan dan kembalikan error.
			algo := "unknown"
			if algStr, okAlg := token.Header["alg"].(string); okAlg {
				algo = algStr
			}
			zlog.Warn().Str("algorithm", algo).Msg("Unexpected signing method encountered during JWT validation")
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Jika algoritma sesuai, kembalikan secret key yang benar untuk verifikasi signature.
		return jwtSecret, nil
	})

	// Tangani error yang mungkin terjadi selama proses parsing dan validasi awal.
	// Error ini bisa berupa: token malformed, token expired, signature invalid, dll.
	if err != nil {
		// Log sebagai Warning karena ini seringkali disebabkan oleh input dari klien (token lama, token rusak).
		zlog.Warn().Err(err).Msg("Error parsing or validating JWT token")
		// Kembalikan error yang jelas ke pemanggil (misal: middleware).
		return nil, fmt.Errorf("token validation failed: %w", err) // Bungkus error asli
	}

	// --- Pemeriksaan Akhir Validitas Token dan Claims ---
	// Setelah parsing berhasil, periksa apakah token secara keseluruhan valid (`token.Valid`)
	// dan apakah claims berhasil di-decode ke dalam struct `*JwtClaims`.
	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		// Jika semua valid, log (level debug) dan kembalikan pointer ke claims.
		zlog.Debug().Str("username", claims.Username).Int("user_id", claims.UserID).Str("role", claims.Role).Msg("JWT token validated successfully")
		return claims, nil
	}

	// Jika `token.Valid` bernilai false atau casting claims gagal (seharusnya jarang terjadi jika parsing berhasil).
	zlog.Warn().Msg("Token parsed but marked as invalid or claims could not be cast")
	return nil, fmt.Errorf("invalid token") // Kembalikan error generik "invalid token"
}

// ====================================================================================
// Fungsi Helper Ekstraksi Informasi dari Request/Context
// ====================================================================================

// ExtractToken adalah fungsi helper untuk mengambil token string dari header "Authorization" sebuah request Fiber.
// Fungsi ini mengharapkan format header "Bearer <token>".
//
// Parameter:
//   - c: Pointer ke context request Fiber (*fiber.Ctx).
//
// Mengembalikan:
//   - string: Bagian token dari header jika formatnya benar.
//   - string kosong: Jika header "Authorization" tidak ada atau formatnya salah.
func ExtractToken(c *fiber.Ctx) string {
	// Ambil nilai header "Authorization".
	authHeader := c.Get(fiber.HeaderAuthorization)
	if authHeader == "" {
		zlog.Debug().Str("path", c.Path()).Msg("Authorization header missing")
		return "" // Header tidak ada
	}

	// Pisahkan string header berdasarkan spasi. Harusnya menghasilkan ["Bearer", "<token>"].
	parts := strings.Split(authHeader, " ")

	// Periksa apakah formatnya benar: harus ada 2 bagian dan bagian pertama adalah "Bearer" (case-insensitive).
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		// Jika format benar, kembalikan bagian kedua, yaitu token string.
		return parts[1]
	}

	// Jika format salah, log peringatan dan kembalikan string kosong.
	zlog.Warn().Str("path", c.Path()).Str("AuthorizationHeaderValue", authHeader).Msg("Invalid Authorization header format (Expected 'Bearer <token>')")
	return ""
}

// ExtractUserIDFromJWT adalah fungsi helper untuk mengambil UserID dari context Fiber (`c.Locals`).
// Fungsi ini mengasumsikan bahwa middleware `Protected()` sudah berjalan sebelumnya dan
// berhasil menyimpan pointer ke `*JwtClaims` di `c.Locals("user")`.
//
// Parameter:
//   - c: Pointer ke context request Fiber (*fiber.Ctx).
//
// Mengembalikan:
//   - int: UserID yang diekstrak dari claims.
//   - error: Error jika data claims tidak ditemukan di context atau tipenya salah (menandakan masalah konfigurasi middleware).
func ExtractUserIDFromJWT(c *fiber.Ctx) (int, error) {
	// Ambil data dari Locals dengan kunci "user".
	claimsData := c.Locals("user")
	if claimsData == nil {
		zlog.Error().Str("path", c.Path()).Msg("User claims not found in Fiber context (key 'user' is nil). Ensure Protected middleware runs first.")
		return 0, fmt.Errorf("user claims not found in context")
	}

	// Lakukan type assertion yang aman ke *JwtClaims.
	claims, ok := claimsData.(*JwtClaims)
	if !ok {
		// Jika type assertion gagal, berarti data yang disimpan di Locals bukan tipe yang diharapkan.
		zlog.Error().Str("path", c.Path()).Interface("locals_user_type", fmt.Sprintf("%T", claimsData)).Msg("Invalid type for user claims in Fiber context. Expected *JwtClaims.")
		return 0, fmt.Errorf("invalid user claims type in context")
	}

	// Jika berhasil, kembalikan UserID dari claims.
	return claims.UserID, nil
}

// ExtractRoleFromJWT adalah fungsi helper untuk mengambil Role (peran) pengguna dari context Fiber (`c.Locals`).
// Sama seperti `ExtractUserIDFromJWT`, fungsi ini mengasumsikan `Protected()` sudah berjalan.
//
// Parameter:
//   - c: Pointer ke context request Fiber (*fiber.Ctx).
//
// Mengembalikan:
//   - string: String peran pengguna yang diekstrak dari claims.
//   - error: Error jika data claims tidak ditemukan atau tipenya salah.
func ExtractRoleFromJWT(c *fiber.Ctx) (string, error) {
	// Ambil data dari Locals dengan kunci "user".
	claimsData := c.Locals("user")
	if claimsData == nil {
		zlog.Error().Str("path", c.Path()).Msg("User claims not found in Fiber context (key 'user' is nil). Ensure Protected middleware runs first.")
		return "", fmt.Errorf("user claims not found in context")
	}

	// Lakukan type assertion yang aman ke *JwtClaims.
	claims, ok := claimsData.(*JwtClaims)
	if !ok {
		zlog.Error().Str("path", c.Path()).Interface("locals_user_type", fmt.Sprintf("%T", claimsData)).Msg("Invalid type for user claims in Fiber context. Expected *JwtClaims.")
		return "", fmt.Errorf("invalid user claims type in context")
	}

	// Jika berhasil, kembalikan Role dari claims.
	return claims.Role, nil
}

// ExtractUserIDFromParam adalah fungsi helper untuk mengambil ID (biasanya UserID, TaskID, RewardID, dll.)
// dari parameter path URL sebuah request Fiber.
// Contoh: Untuk route `/users/:userId`, panggil `ExtractUserIDFromParam(c, "userId")`.
//
// Parameter:
//   - c: Pointer ke context request Fiber (*fiber.Ctx).
//   - paramName: Nama parameter path yang didefinisikan dalam route (misal: "userId", "taskId").
//
// Mengembalikan:
//   - int: Nilai ID integer yang diekstrak dari parameter.
//   - error: Error jika parameter dengan nama `paramName` tidak ditemukan di URL atau nilainya bukan angka yang valid.
func ExtractUserIDFromParam(c *fiber.Ctx, paramName string) (int, error) {
	// Ambil nilai parameter dari path URL menggunakan nama yang diberikan.
	idStr := c.Params(paramName)
	if idStr == "" {
		// Jika parameter tidak ditemukan di URL.
		zlog.Warn().Str("paramName", paramName).Str("path", c.Path()).Msg("Missing required ID parameter in URL path")
		return 0, fmt.Errorf("missing required parameter '%s' in URL path", paramName)
	}

	// Coba konversi nilai parameter string ke integer.
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// Jika konversi gagal (nilai parameter bukan angka).
		zlog.Warn().Err(err).Str("paramName", paramName).Str("value", idStr).Str("path", c.Path()).Msg("Invalid numeric value for ID parameter in URL path")
		return 0, fmt.Errorf("invalid parameter '%s': expected a number, got '%s'", paramName, idStr)
	}

	// Jika konversi berhasil, kembalikan nilai ID integer.
	return id, nil
}
