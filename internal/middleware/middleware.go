// internal/middleware/middleware.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"  // Middleware untuk kompresi response (Gzip)
	"github.com/gofiber/fiber/v2/middleware/cors"      // Middleware untuk Cross-Origin Resource Sharing
	"github.com/gofiber/fiber/v2/middleware/limiter"   // Middleware untuk membatasi rate request
	"github.com/gofiber/fiber/v2/middleware/recover"   // Middleware untuk menangkap panic
	"github.com/gofiber/fiber/v2/middleware/requestid" // Middleware untuk menambahkan ID unik ke request
	"github.com/rs/zerolog"                            // Digunakan oleh logger request
	zlog "github.com/rs/zerolog/log"                   // Logger global Zerolog
)

// File ini berisi fungsi untuk mendaftarkan middleware global yang berlaku
// untuk seluruh atau sebagian besar request yang masuk ke aplikasi Fiber.

// ====================================================================================
// Fungsi Setup Middleware Global
// ====================================================================================

// SetupGlobalMiddleware mendaftarkan serangkaian middleware standar ke instance aplikasi Fiber.
// Middleware ini dieksekusi secara berurutan untuk setiap request yang masuk,
// sebelum request tersebut mencapai handler spesifik route.
// Urutan pendaftaran middleware sangat penting karena menentukan urutan eksekusinya.
func SetupGlobalMiddleware(app *fiber.App) {
	zlog.Info().Msg("Registering global middleware...")

	// --- 1. Recover Middleware (Harus Paling Awal) ---
	// Middleware ini menangkap 'panic' yang mungkin terjadi di handler atau middleware lain
	// dalam rantai eksekusi. Ini mencegah server crash dan mengembalikan response
	// 500 Internal Server Error secara otomatis. Sangat penting untuk didaftarkan pertama.
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true, // Aktifkan stack trace dalam log saat panic (berguna untuk debugging)
	}))
	zlog.Info().Msg("-> Recover middleware registered (with stack trace on panic)")

	// --- 2. Request ID Middleware ---
	// Middleware ini menambahkan header unik 'X-Request-ID' ke setiap request (jika belum ada)
	// dan juga menyimpannya di context locals (`c.Locals("requestid")`).
	// ID ini sangat berguna untuk melacak (tracing) alur sebuah request melalui log di berbagai komponen.
	app.Use(requestid.New())
	zlog.Info().Msg("-> RequestID middleware registered")

	// --- 3. CORS (Cross-Origin Resource Sharing) Middleware ---
	// Middleware ini mengatur header HTTP yang diperlukan agar browser mengizinkan
	// request dari domain frontend yang berbeda (misal: http://localhost:5173)
	// ke domain backend API ini (misal: http://localhost:3001).
	// Konfigurasi ini harus disesuaikan untuk lingkungan production.
	app.Use(cors.New(cors.Config{
		// AllowOrigins: "*", // Izinkan SEMUA origin (TIDAK AMAN untuk production!)
		// Ganti dengan daftar origin frontend Anda yang spesifik di production, pisahkan dengan koma.
		AllowOrigins: "http://localhost:5173, http://127.0.0.1:5173, https://your-frontend-domain.com", // Contoh
		AllowMethods: "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD",                                  // Metode HTTP yang diizinkan. OPTIONS penting untuk preflight request.
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Requested-With",                  // Header yang boleh dikirim oleh klien. Authorization penting untuk JWT.
		// AllowCredentials: true, // Set true jika frontend perlu mengirim/menerima cookie atau header Authorization dengan credentials.
		MaxAge: 12 * 3600, // Cache preflight request selama 12 jam (opsional)
	}))
	zlog.Info().Msg("-> CORS middleware registered")

	// --- 4. Rate Limiter Middleware ---
	// Middleware ini membatasi jumlah request yang dapat diterima dari satu IP address
	// dalam periode waktu tertentu. Ini membantu melindungi API dari serangan brute-force,
	// scraping, atau penyalahgunaan (abuse) lainnya.
	app.Use(limiter.New(limiter.Config{
		Max:        200,             // Maksimum 200 request per IP...
		Expiration: 1 * time.Minute, // ...dalam jendela waktu 1 menit.
		// KeyGenerator: func(c *fiber.Ctx) string {
		// 	// Gunakan header X-Forwarded-For jika aplikasi berjalan di belakang reverse proxy/load balancer
		// 	// untuk mendapatkan IP asli klien. Pastikan proxy Anda terkonfigurasi dengan benar.
		// 	ip := c.Get("X-Forwarded-For")
		// 	if ip == "" {
		// 		ip = c.IP()
		// 	}
		// 	return ip
		// },
		LimiterMiddleware: limiter.SlidingWindow{}, // Menggunakan algoritma Sliding Window untuk rate limiting.
		LimitReached: func(c *fiber.Ctx) error { // Handler kustom saat limit tercapai
			zlog.Warn().
				Str("ip", c.IP()).
				Str("path", c.Path()).
				Msg("Rate limit exceeded")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Too many requests, please try again later.",
			})
		},
	}))
	zlog.Info().Msg("-> Rate limiter middleware registered (200 req/min per IP)")

	// --- 5. Custom Request Logger Middleware (Menggunakan Zerolog) ---
	// Middleware ini mencatat informasi penting tentang setiap request HTTP yang masuk
	// dan response yang dikirim. Log ini sangat berguna untuk monitoring dan debugging.
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now() // Catat waktu mulai pemrosesan request

		// Lanjutkan eksekusi ke middleware atau handler berikutnya dalam rantai.
		// Jika ada error yang dikembalikan oleh handler/middleware selanjutnya,
		// error tersebut akan ditangkap di sini.
		err := c.Next()

		stop := time.Now()                      // Catat waktu selesai pemrosesan request
		latency := stop.Sub(start)              // Hitung durasi pemrosesan
		statusCode := c.Response().StatusCode() // Dapatkan status code HTTP dari response

		// Ambil Request ID yang sudah ditambahkan oleh middleware requestid
		requestID := c.Locals("requestid").(string) // Type assertion langsung (aman karena requestid selalu ada)

		// Tentukan level log berdasarkan status code atau adanya error dari handler
		var logEvent *zerolog.Event
		if err != nil {
			// Jika handler mengembalikan error (akan ditangani juga oleh ErrorHandler global),
			// log sebagai warning atau error di sini. ErrorHandler global yang akan mengirim response ke klien.
			// Kita tetap log di sini untuk konteks request.
			logEvent = zlog.Warn().Err(err) // Default ke Warn jika ada error dari handler
			// Anda bisa menambahkan logika untuk level Error jika error-nya spesifik
		} else {
			// Log request yang (kemungkinan) sukses (tidak ada error dari handler)
			switch {
			case statusCode >= 500:
				logEvent = zlog.Error() // Status 5xx (Server Error) -> Level Error
			case statusCode >= 400:
				logEvent = zlog.Warn() // Status 4xx (Client Error) -> Level Warn
			default:
				logEvent = zlog.Info() // Status 1xx, 2xx, 3xx -> Level Info
			}
		}

		// Bangun field-field log yang relevan
		logEvent.
			Str("request_id", requestID).                   // ID unik request untuk tracing
			Str("method", c.Method()).                      // Metode HTTP (GET, POST, dll.)
			Str("path", c.Path()).                          // Path URL yang diminta
			Int("status", statusCode).                      // Status code HTTP response
			Dur("latency", latency).                        // Durasi pemrosesan request
			Str("ip", c.IP()).                              // IP address klien
			Str("user_agent", c.Get(fiber.HeaderUserAgent)). // User agent dari browser/klien
			Msg("Incoming request handled")                 // Pesan log utama

		// Kembalikan error (jika ada) agar bisa ditangani oleh ErrorHandler global Fiber
		// atau middleware Recover jika itu adalah panic.
		return err
	})
	zlog.Info().Msg("-> Custom request logger middleware registered")

	// --- 6. Compression Middleware (Sebaiknya Mendekati Akhir) ---
	// Middleware ini mengompresi body response menggunakan Gzip (jika klien mendukungnya
	// melalui header 'Accept-Encoding'). Ini dapat menghemat bandwidth secara signifikan.
	// Diletakkan mendekati akhir agar response yang sudah final yang dikompres.
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // Prioritaskan kecepatan kompresi. Opsi lain: LevelDefault, LevelBestCompression.
	}))
	zlog.Info().Msg("-> Compress middleware registered (level: best speed)")

	// --- Middleware Global Lainnya ---
	// Anda bisa menambahkan middleware global lain di sini sesuai kebutuhan.
	// Contoh:
	// app.Use(helmet.New()) // Menambahkan header keamanan (perlu library terpisah: gofiber/helmet)
	// zlog.Info().Msg("-> Helmet middleware registered")

	zlog.Info().Msg("Global middleware registration complete.")
}
