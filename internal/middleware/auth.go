// internal/middleware/auth.go
package middleware

import (
	"strings" // Digunakan untuk perbandingan string case-insensitive (EqualFold)

	"github.com/gofiber/fiber/v2"                                  // Framework Fiber
	"github.com/rakaarfi/digital-parenting-app-be/internal/models" // Model untuk struktur Response standar
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"  // Utilitas untuk JWT (ExtractToken, ValidateJWT, JwtClaims)
	zlog "github.com/rs/zerolog/log"                               // Logger global Zerolog
)

// File ini berisi middleware yang berkaitan dengan autentikasi (memastikan pengguna valid)
// dan otorisasi (memastikan pengguna memiliki hak akses yang sesuai).

// ====================================================================================
// Middleware: Protected (JWT Authentication)
// ====================================================================================

// Protected adalah middleware Fiber yang berfungsi untuk melindungi route.
// Middleware ini memastikan bahwa request yang masuk memiliki token JWT yang valid
// di header Authorization (format: "Bearer <token>").
// Jika token valid, informasi pengguna (claims) akan disimpan di `c.Locals("user")`
// untuk digunakan oleh handler atau middleware selanjutnya.
// Jika token tidak ada atau tidak valid, request akan dihentikan dengan response 401 Unauthorized.
//
// Middleware ini harus dijalankan *sebelum* handler atau middleware lain yang
// memerlukan data pengguna yang terautentikasi (seperti middleware Authorize).
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// --- Langkah 1: Ekstrak Token dari Header Authorization ---
		tokenString := utils.ExtractToken(c)
		if tokenString == "" {
			// Kasus: Header Authorization tidak ada atau tidak menggunakan format Bearer.
			zlog.Warn().
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Protected route access attempt: Missing or malformed Authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(models.Response{
				Success: false, Message: "Unauthorized: Missing or invalid authentication token",
			})
		}

		// --- Langkah 2: Validasi Token JWT ---
		// Memverifikasi signature, masa berlaku (expiry), dan mem-parsing claims dari token.
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			// Kasus: Token tidak valid (kadaluarsa, signature salah, format rusak, dll.).
			zlog.Warn().
				Err(err). // Sertakan detail error validasi JWT
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Protected route access attempt: Invalid JWT token")
			return c.Status(fiber.StatusUnauthorized).JSON(models.Response{
				Success: false, Message: "Unauthorized: Invalid or expired token",
			})
		}

		// --- Langkah 3: Simpan Claims Pengguna ke Context Locals ---
		// Jika token valid, simpan data claims (*utils.JwtClaims) ke dalam context request Fiber (c.Locals).
		// Kunci "user" digunakan secara konvensi agar mudah diakses oleh komponen selanjutnya.
		c.Locals("user", claims) // Menyimpan pointer ke struct JwtClaims

		// --- Langkah 4: Lanjutkan ke Middleware/Handler Berikutnya ---
		// Log level debug untuk menandakan autentikasi berhasil (hanya muncul jika LOG_LEVEL=debug).
		zlog.Debug().
			Str("username", claims.Username).
			Int("user_id", claims.UserID).
			Str("role", claims.Role).
			Str("path", c.Path()).
			Msg("JWT authentication successful, proceeding to next handler/middleware")

		// Melanjutkan eksekusi ke middleware atau handler selanjutnya dalam rantai.
		return c.Next()
	}
}

// ====================================================================================
// Middleware: Authorize (Role-Based Authorization)
// ====================================================================================

// Authorize adalah middleware Fiber yang berfungsi untuk membatasi akses ke route
// berdasarkan peran (role) pengguna yang terautentikasi.
// Middleware ini memeriksa apakah peran pengguna (yang didapat dari claims JWT)
// termasuk dalam daftar peran yang diizinkan (`allowedRoles`).
//
// PENTING: Middleware ini WAJIB dijalankan *setelah* middleware `Protected()`
// karena memerlukan data claims pengguna yang sudah disimpan di `c.Locals("user")`.
//
// Jika peran pengguna diizinkan, request akan dilanjutkan ke handler berikutnya.
// Jika tidak diizinkan, request akan dihentikan dengan response 403 Forbidden.
//
// Parameter:
//   - allowedRoles: Daftar string nama peran yang diizinkan mengakses route ini (varargs).
//     Perbandingan peran dilakukan secara case-insensitive.
func Authorize(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// --- Langkah 1: Ambil Claims Pengguna dari Context Locals ---
		// Mengambil data claims (*utils.JwtClaims) yang seharusnya sudah disimpan oleh `Protected()`.
		userClaims, ok := c.Locals("user").(*utils.JwtClaims)
		if !ok || userClaims == nil {
			// Kasus: Claims tidak ditemukan atau tipenya salah. Ini mengindikasikan kesalahan
			// urutan middleware (Protected() tidak dijalankan sebelumnya).
			zlog.Error().
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Authorization middleware error: User claims not found in context. Ensure Protected() runs first.")
			// Mengembalikan 500 Internal Server Error karena ini masalah konfigurasi server.
			return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
				Success: false, Message: "Internal Server Error: User context unavailable for authorization",
			})
		}

		// --- Langkah 2: Periksa Kecocokan Peran Pengguna ---
		isAllowed := false
		userRole := userClaims.Role // Peran pengguna dari token JWT
		for _, allowedRole := range allowedRoles {
			// Membandingkan peran pengguna dengan setiap peran yang diizinkan.
			// `strings.EqualFold` melakukan perbandingan case-insensitive (misal: "Admin" cocok dengan "admin").
			if strings.EqualFold(userRole, allowedRole) {
				isAllowed = true // Jika ditemukan kecocokan, tandai sebagai diizinkan dan keluar dari loop.
				break
			}
		}

		// --- Langkah 3: Proses Hasil Pemeriksaan Peran ---
		if !isAllowed {
			// Kasus: Peran pengguna tidak termasuk dalam daftar peran yang diizinkan.
			zlog.Warn().
				Str("username", userClaims.Username).
				Int("user_id", userClaims.UserID).
				Str("user_role", userRole).
				Strs("required_roles", allowedRoles). // Log peran yang dibutuhkan
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Authorization failed: User role not permitted for this route")
			// Mengembalikan 403 Forbidden karena pengguna terautentikasi tetapi tidak punya hak akses.
			return c.Status(fiber.StatusForbidden).JSON(models.Response{
				Success: false, Message: "Forbidden: You do not have sufficient privileges to access this resource",
			})
		}

		// --- Langkah 4: Lanjutkan Jika Peran Diizinkan ---
		// Log level debug untuk menandakan otorisasi berhasil.
		zlog.Debug().
			Str("username", userClaims.Username).
			Int("user_id", userClaims.UserID).
			Str("role", userRole).
			Strs("allowed_roles", allowedRoles).
			Str("path", c.Path()).
			Msg("Authorization successful, proceeding to next handler")

		// Melanjutkan eksekusi ke handler atau middleware selanjutnya.
		return c.Next()
	}
}
