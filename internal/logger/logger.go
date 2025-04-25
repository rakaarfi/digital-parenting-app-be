// internal/logger/logger.go
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath" // Untuk manipulasi path file dan direktori
	"strconv"       // Untuk konversi string (dari env vars) ke bool/int
	"strings"       // Untuk manipulasi string (ToLower)
	"time"

	"github.com/rs/zerolog"            // Core library Zerolog
	"github.com/rs/zerolog/log"        // Akses ke logger global Zerolog (log.Logger)
	"gopkg.in/natefinch/lumberjack.v2" // Library untuk rotasi file log
)

// File ini bertanggung jawab untuk mengkonfigurasi dan menginisialisasi
// logger global (Zerolog) yang digunakan di seluruh aplikasi.

// ====================================================================================
// Fungsi Setup Logger Global
// ====================================================================================

// SetupLogger mengkonfigurasi logger global Zerolog berdasarkan variabel lingkungan (environment variables).
// Fungsi ini mendukung beberapa fitur:
// - Tingkat log yang dapat dikonfigurasi (trace, debug, info, warn, error, fatal, panic).
// - Output ke konsol (Stderr) dengan format human-readable (untuk development) atau JSON (untuk production/log management).
// - Output opsional ke file log dengan rotasi otomatis (ukuran, jumlah backup, usia, kompresi) menggunakan lumberjack.
//
// Mengembalikan `io.Closer` yang merepresentasikan file logger (jika logging ke file diaktifkan dan berhasil).
// `io.Closer` ini *harus* ditutup menggunakan `defer` di fungsi `main` aplikasi untuk memastikan
// semua buffer log ditulis ke file sebelum aplikasi berhenti. Jika file logging tidak aktif atau
// gagal diinisialisasi, fungsi ini akan mengembalikan `nil`.
//
// Variabel Lingkungan yang Didukung:
//   - LOG_LEVEL: Tingkat log minimum (trace, debug, info, warn, error, fatal, panic). Default: "info".
//   - LOG_FORMAT: Format output konsol ("json" atau lainnya untuk human-readable). Default: human-readable.
//   - LOG_FILE_ENABLED: Aktifkan logging ke file ("true" atau "false"). Default: "false".
//   - LOG_FILE_PATH: Path lengkap ke file log. Default: "./logs/app.log".
//   - LOG_FILE_MAX_SIZE_MB: Ukuran maksimum file log (MB) sebelum dirotasi. Default: 100.
//   - LOG_FILE_MAX_BACKUPS: Jumlah maksimum file log lama yang disimpan. Default: 5.
//   - LOG_FILE_MAX_AGE_DAYS: Usia maksimum file log lama (hari) sebelum dihapus. Default: 30.
//   - LOG_FILE_COMPRESS: Kompres file log lama yang dirotasi ("true" atau "false"). Default: "false".
func SetupLogger() io.Closer {
	fmt.Fprintln(os.Stderr, "[INFO] Initializing global logger...") // Pesan awal ke Stderr

	// --- Langkah 1: Konfigurasi Tingkat Log Global ---
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL")) // Baca & normalisasi ke lowercase
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil || logLevelStr == "" {
		logLevel = zerolog.InfoLevel // Default ke Info jika tidak valid atau kosong
		fmt.Fprintf(os.Stderr, "[WARN] Invalid or missing LOG_LEVEL env var ('%s'), defaulting to: %s\n", logLevelStr, logLevel.String())
	}
	zerolog.SetGlobalLevel(logLevel) // Terapkan level log yang dipilih secara global
	fmt.Fprintf(os.Stderr, "[INFO] Global log level set to: %s\n", logLevel.String())

	// --- Langkah 2: Konfigurasi Writer (Tujuan Output Log) ---
	var writers []io.Writer // Slice untuk menampung semua tujuan output (konsol, file)
	var consoleWriter io.Writer

	// 2a. Konfigurasi Console Writer
	logFormat := strings.ToLower(os.Getenv("LOG_FORMAT"))
	if logFormat == "json" {
		// Format JSON: Cocok untuk dikonsumsi oleh sistem log management (Elasticsearch, Splunk, dll.)
		consoleWriter = os.Stderr // Output JSON langsung ke Stderr
		fmt.Fprintln(os.Stderr, "[INFO] Console log format set to: JSON")
	} else {
		// Format Human-Readable: Lebih mudah dibaca di terminal saat development.
		consoleWriter = zerolog.ConsoleWriter{
			Out:        os.Stderr,              // Tujuan output (Stderr)
			TimeFormat: time.RFC3339,           // Format timestamp yang jelas
			NoColor:    false,                  // Aktifkan warna (jika terminal mendukung)
		}
		fmt.Fprintln(os.Stderr, "[INFO] Console log format set to: Human-Readable (colored)")
	}
	writers = append(writers, consoleWriter) // Tambahkan console writer ke daftar

	// --- Langkah 3: Konfigurasi File Writer (Opsional, dengan Rotasi) ---
	var fileCloser io.Closer // Variabel untuk menyimpan handle lumberjack agar bisa di-Close()
	logFileEnabledStr := strings.ToLower(os.Getenv("LOG_FILE_ENABLED"))
	logFileEnabled, _ := strconv.ParseBool(logFileEnabledStr) // Default false jika error/kosong

	if logFileEnabled {
		fmt.Fprintln(os.Stderr, "[INFO] File logging is enabled. Configuring file writer...")

		// 3a. Tentukan Path File Log
		logFilePath := os.Getenv("LOG_FILE_PATH")
		if logFilePath == "" {
			logFilePath = "./logs/app.log" // Default path jika tidak diset
			fmt.Fprintf(os.Stderr, "[WARN] LOG_FILE_PATH not set, defaulting to: %s\n", logFilePath)
		}

		// 3b. Pastikan Direktori Log Ada
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil { // 0755: rwxr-xr-x
			// Jika gagal membuat direktori, log error ke Stderr dan batalkan file logging.
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to create log directory '%s': %v. File logging will be disabled.\n", logDir, err)
			logFileEnabled = false // Nonaktifkan file logging jika direktori gagal dibuat
		} else {
			fmt.Fprintf(os.Stderr, "[INFO] Log directory ensured: %s\n", logDir)

			// 3c. Baca Konfigurasi Rotasi File
			maxSizeMB, err := strconv.Atoi(os.Getenv("LOG_FILE_MAX_SIZE_MB"))
			if err != nil || maxSizeMB <= 0 {
				maxSizeMB = 100 // Default 100 MB
			}
			maxBackups, err := strconv.Atoi(os.Getenv("LOG_FILE_MAX_BACKUPS"))
			if err != nil || maxBackups < 0 { // Boleh 0 (tidak ada backup)
				maxBackups = 5 // Default 5 backup
			}
			maxAgeDays, err := strconv.Atoi(os.Getenv("LOG_FILE_MAX_AGE_DAYS"))
			if err != nil || maxAgeDays <= 0 {
				maxAgeDays = 30 // Default 30 hari
			}
			compressLogsStr := strings.ToLower(os.Getenv("LOG_FILE_COMPRESS"))
			compressLogs, _ := strconv.ParseBool(compressLogsStr) // Default false

			fmt.Fprintf(os.Stderr, "[INFO] File rotation config: MaxSize=%dMB, MaxBackups=%d, MaxAge=%ddays, Compress=%t\n",
				maxSizeMB, maxBackups, maxAgeDays, compressLogs)

			// 3d. Inisialisasi Lumberjack (File Rotation Handler)
			fileWriter := &lumberjack.Logger{
				Filename:   logFilePath,  // Nama file log utama.
				MaxSize:    maxSizeMB,    // Maksimum ukuran (MB) sebelum file dirotasi.
				MaxBackups: maxBackups,   // Maksimum jumlah file lama (.log.gz jika kompres) yang disimpan.
				MaxAge:     maxAgeDays,   // Maksimum usia (hari) file lama sebelum dihapus.
				Compress:   compressLogs, // Apakah file lama dikompres (gzip).
				LocalTime:  true,         // Gunakan waktu lokal untuk nama file backup (opsional)
			}
			writers = append(writers, fileWriter) // Tambahkan file writer ke daftar output.
			fileCloser = fileWriter               // Simpan handle lumberjack untuk di-Close nanti.
			fmt.Fprintf(os.Stderr, "[INFO] File writer configured for: %s\n", logFilePath)
		}
	} else {
		fmt.Fprintln(os.Stderr, "[INFO] File logging is disabled.")
	}

	// --- Langkah 4: Gabungkan Semua Writer Menjadi Satu ---
	// MultiLevelWriter memungkinkan log ditulis ke semua writer yang terdaftar (konsol dan/atau file).
	multiWriter := zerolog.MultiLevelWriter(writers...)

	// --- Langkah 5: Atur Logger Global Zerolog ---
	// Buat instance logger baru yang akan menulis ke `multiWriter`.
	// `.With()` memulai context builder untuk menambahkan field global ke semua log.
	// `.Timestamp()` menambahkan field timestamp otomatis.
	// `.Caller()` menambahkan field caller (nama file:baris) otomatis.
	// `.Logger()` menyelesaikan pembuatan logger.
	// Logger ini kemudian ditetapkan sebagai logger global `log.Logger`.
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Caller().Logger()

	// Log pesan konfirmasi terakhir menggunakan logger yang sudah sepenuhnya terkonfigurasi.
	log.Info().Msgf("Global logger setup complete. Level: %s. Console Format: %s. File Logging Enabled: %t.",
		zerolog.GlobalLevel().String(),
		logFormat, // Atau tentukan format console secara eksplisit
		logFileEnabled && fileCloser != nil) // Konfirmasi file logging benar-benar aktif

	// Kembalikan fileCloser (bisa nil jika file logging tidak aktif/gagal)
	// agar bisa ditutup dengan benar di fungsi main.
	return fileCloser
}
