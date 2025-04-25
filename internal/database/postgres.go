// internal/database/postgres.go
package database

import (
	"context" // Paket standar untuk mengelola context, terutama untuk timeout dan pembatalan.
	"fmt"     // Paket standar untuk formatting string.
	"os"      // Paket standar untuk interaksi OS, digunakan di sini untuk membaca environment variables.
	"time"    // Paket standar untuk fungsionalitas waktu (durasi, timeout).

	"github.com/jackc/pgx/v5/pgxpool" // Driver PostgreSQL modern dan efisien dari JackC, fokus pada connection pool.
	zlog "github.com/rs/zerolog/log"  // Logger global Zerolog yang sudah dikonfigurasi.
)

// File ini bertanggung jawab untuk menginisialisasi koneksi ke database PostgreSQL
// menggunakan library pgxpool.

// ====================================================================================
// Fungsi Inisialisasi Connection Pool PostgreSQL
// ====================================================================================

// NewPgxPool membuat dan mengkonfigurasi sebuah connection pool baru ke database PostgreSQL.
// Fungsi ini melakukan langkah-langkah berikut:
// 1. Membaca konfigurasi koneksi database dari environment variables.
// 2. Mem-parsing Data Source Name (DSN) string.
// 3. Mengkonfigurasi parameter connection pool (jumlah koneksi, lifetime, dll.).
// 4. Membuat instance connection pool (*pgxpool.Pool).
// 5. Melakukan ping ke database untuk memverifikasi koneksi awal.
//
// Mengembalikan instance *pgxpool.Pool yang siap digunakan dan error jika terjadi kegagalan
// pada salah satu langkah di atas.
func NewPgxPool() (*pgxpool.Pool, error) {
	zlog.Info().Msg("Initializing PostgreSQL connection pool...")

	// --- Langkah 1: Baca Konfigurasi Database dari Environment Variables ---
	zlog.Debug().Msg("Reading database configuration from environment variables...")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD") // Password sebaiknya tidak di-log
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE") // Contoh: 'disable', 'require', 'verify-full'

	// Validasi minimal (opsional, tapi baik untuk dilakukan)
	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		zlog.Error().Msg("One or more required database environment variables (DB_HOST, DB_PORT, DB_USER, DB_NAME) are not set.")
		return nil, fmt.Errorf("missing required database configuration environment variables")
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable" // Default ke 'disable' jika tidak diset, sesuaikan jika perlu
		zlog.Warn().Msg("DB_SSLMODE environment variable not set, defaulting to 'disable'. Consider setting it explicitly for production.")
	}

	// Membuat Data Source Name (DSN) string sesuai format yang dibutuhkan oleh pgx.
	// Hindari logging password secara langsung.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	dsnLoggable := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s", // Versi tanpa password untuk logging
		dbHost, dbPort, dbUser, dbName, dbSSLMode)
	zlog.Debug().Str("dsn_loggable", dsnLoggable).Msg("Constructed database DSN")

	// --- Langkah 2: Parse Konfigurasi DSN ---
	zlog.Debug().Msg("Parsing database DSN string...")
	// Mengubah DSN string menjadi struct konfigurasi *pgxpool.Config.
	// Fungsi ini juga melakukan validasi dasar pada format DSN.
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// Jika DSN tidak valid, log error dan kembalikan error sebelum mencoba membuat pool.
		zlog.Error().Err(err).Str("dsn_loggable", dsnLoggable).Msg("Failed to parse database DSN")
		// %w digunakan untuk membungkus error asli, mempertahankan konteks error.
		return nil, fmt.Errorf("unable to parse database configuration: %w", err)
	}

	// --- Langkah 3: Konfigurasi Detail Connection Pool (Opsional tapi Direkomendasikan) ---
	zlog.Debug().Msg("Configuring connection pool parameters...")
	// Menyesuaikan parameter pool untuk mengontrol jumlah koneksi, masa hidup, dll.
	// Nilai default mungkin cukup untuk development, tapi sangat direkomendasikan
	// untuk disesuaikan (tuned) berdasarkan beban kerja dan sumber daya server di production.
	config.MaxConns = 10                               // Jumlah maksimum koneksi (aktif + idle) yang diizinkan dalam pool. Sesuaikan berdasarkan load.
	config.MinConns = 2                                // Jumlah minimum koneksi yang coba dipertahankan idle oleh pool. Membantu mengurangi latensi saat ada lonjakan request.
	config.MaxConnLifetime = time.Hour                 // Durasi maksimum sebuah koneksi bisa digunakan sebelum ditutup paksa (bahkan jika sedang idle). Membantu load balancing di cluster DB dan mencegah koneksi 'basi'.
	config.MaxConnIdleTime = 30 * time.Minute          // Durasi maksimum koneksi idle bisa bertahan di pool sebelum ditutup. Menghemat resource di DB server.
	config.HealthCheckPeriod = time.Minute             // Seberapa sering pool secara proaktif memeriksa koneksi idle yang mungkin 'rusak' atau terputus.
	config.ConnConfig.ConnectTimeout = 5 * time.Second // Waktu maksimum yang diizinkan untuk mencoba membuat satu koneksi *baru* ke database.
	zlog.Debug().
		Int32("max_conns", config.MaxConns).
		Int32("min_conns", config.MinConns).
		Dur("max_conn_lifetime", config.MaxConnLifetime).
		Dur("max_conn_idle_time", config.MaxConnIdleTime).
		Dur("health_check_period", config.HealthCheckPeriod).
		Dur("connect_timeout", config.ConnConfig.ConnectTimeout).
		Msg("Connection pool parameters set")

	// --- Langkah 4: Buat Connection Pool ---
	zlog.Info().Msg("Attempting to create database connection pool...")
	// Mencoba membuat pool koneksi menggunakan konfigurasi yang sudah di-parse dan disesuaikan.
	// `context.Background()` digunakan karena inisialisasi pool ini terjadi saat startup aplikasi,
	// di luar konteks request HTTP individual.
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		// Jika pembuatan pool gagal (misal: masalah jaringan awal, autentikasi salah, DB down),
		// log error fatal dan kembalikan error.
		zlog.Error().Err(err).Msg("Failed to create database connection pool")
		return nil, fmt.Errorf("unable to create database connection pool: %w", err)
	}
	zlog.Info().Msg("Database connection pool structure created.")

	// --- Langkah 5: Verifikasi Koneksi Awal (Ping) ---
	zlog.Info().Msg("Pinging database to verify initial connection...")
	// Melakukan 'ping' ke database menggunakan salah satu koneksi dari pool yang baru dibuat.
	// Ini adalah cara terbaik untuk memastikan bahwa konfigurasi benar dan database dapat dijangkau.
	// Menggunakan context dengan timeout agar proses ping tidak menggantung tanpa batas jika ada masalah jaringan.
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Timeout 5 detik untuk operasi ping.
	defer cancel()                                                              // Pastikan context dibatalkan setelah selesai untuk melepaskan resource terkait timeout.

	if err = pool.Ping(pingCtx); err != nil {
		// Jika ping gagal (DB tidak running, firewall memblokir, kredensial salah, dll.):
		zlog.Error().Err(err).Msg("Database ping failed. Closing unusable pool.")
		pool.Close() // Penting: Tutup pool yang sudah terlanjur dibuat tapi ternyata tidak bisa digunakan.
		return nil, fmt.Errorf("unable to ping database after pool creation: %w", err)
	}
	zlog.Info().Msg("Database ping successful.")

	// --- Langkah 6: Koneksi Berhasil ---
	// Jika semua langkah berhasil, catat pesan sukses dan kembalikan instance pool yang siap pakai.
	zlog.Info().Msg("Successfully connected to PostgreSQL database and verified connection pool!")
	return pool, nil // Kembalikan pool yang valid dan error nil.
}
