// configs/config.go
package configs

import (
	"fmt" // Paket Go standar untuk formatting string (digunakan untuk pesan error).
	"os"  // Paket Go standar untuk interaksi OS, terutama membaca environment variables.

	"github.com/joho/godotenv" // Library pihak ketiga untuk memuat environment variables dari file .env.
	zlog "github.com/rs/zerolog/log"  // Logger global Zerolog (diasumsikan sudah/akan diinisialisasi).
)

// File ini bertanggung jawab untuk memuat konfigurasi aplikasi dari environment variables.
// Ini termasuk memuat variabel dari file .env (jika ada) dan memvalidasi
// keberadaan variabel lingkungan yang wajib ada.

// ====================================================================================
// Fungsi Pemuatan Konfigurasi
// ====================================================================================

// LoadConfig adalah fungsi utama yang dipanggil saat startup aplikasi (biasanya di awal `main`)
// untuk memuat konfigurasi. Fungsi ini melakukan dua hal utama:
// 1. Mencoba memuat variabel lingkungan dari file `.env` di direktori kerja aplikasi.
//    Jika file `.env` tidak ditemukan, fungsi ini akan melanjutkan tanpa error,
//    mengasumsikan variabel lingkungan mungkin sudah diatur langsung di sistem operasi
//    atau melalui cara lain (misalnya, Docker environment variables, Kubernetes secrets).
// 2. Memvalidasi keberadaan variabel lingkungan yang dianggap wajib (`requiredVars`).
//    Jika salah satu variabel wajib tidak ditemukan (nilainya string kosong),
//    aplikasi akan dihentikan secara paksa (`Fatal`) dengan pesan error yang jelas.
func LoadConfig() {
	fmt.Fprintln(os.Stderr, "[INFO] Loading application configuration...") // Pesan awal ke Stderr

	// --- Langkah 1: Coba Muat Variabel dari File .env ---
	// `godotenv.Load()` akan mencari file `.env` di direktori saat ini dan parent-nya,
	// lalu memuat variabel di dalamnya ke environment process saat ini.
	// Variabel yang sudah ada di environment TIDAK akan ditimpa oleh nilai dari .env.
	err := godotenv.Load()
	if err != nil {
		// Jika `godotenv.Load()` mengembalikan error, kemungkinan besar karena file .env tidak ditemukan.
		// Ini BUKAN kondisi error fatal, jadi kita hanya mencatat peringatan (Warn).
		// Aplikasi masih bisa berjalan jika variabel diatur langsung di environment.
		// Gunakan fmt karena logger utama mungkin belum siap saat fungsi ini dipanggil.
		fmt.Fprintln(os.Stderr, "[WARN] No .env file found or error loading it. Reading environment variables directly.")
		// zlog.Warn().Msg("No .env file found or error loading it. Reading environment variables directly.") // Gunakan jika logger sudah siap
	} else {
		fmt.Fprintln(os.Stderr, "[INFO] Loaded environment variables from .env file (if found).")
		// zlog.Info().Msg("Loaded environment variables from .env file (if found).") // Gunakan jika logger sudah siap
	}

	// --- Langkah 2: Validasi Keberadaan Variabel Lingkungan Wajib ---
	// Definisikan daftar nama variabel lingkungan yang HARUS ada agar aplikasi bisa berjalan.
	requiredVars := []string{
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD", // Meskipun nilainya bisa kosong, variabelnya harus ada
		"DB_NAME",
		"APP_PORT",
		"JWT_SECRET", // Sangat penting untuk keamanan JWT
		// Tambahkan variabel wajib lainnya di sini
		// "SOME_OTHER_API_KEY",
	}

	fmt.Fprintf(os.Stderr, "[INFO] Validating %d required environment variables...\n", len(requiredVars))
	missingVars := []string{} // Slice untuk menampung variabel yang hilang

	// Iterasi melalui daftar variabel wajib.
	for _, varName := range requiredVars {
		// `os.Getenv(varName)` akan mengembalikan string kosong jika variabel tidak diset.
		if os.Getenv(varName) == "" {
			// Jika variabel kosong, tambahkan ke daftar yang hilang.
			missingVars = append(missingVars, varName)
			fmt.Fprintf(os.Stderr, "[ERROR] Required environment variable '%s' is not set.\n", varName)
		}
	}

	// Periksa apakah ada variabel wajib yang hilang.
	if len(missingVars) > 0 {
		// Jika ada yang hilang, log pesan Fatal dan hentikan aplikasi.
		// Menggunakan zlog.Fatal() akan otomatis keluar dari program (exit code 1).
		// Jika logger belum siap, gunakan fmt.Fprintf(os.Stderr, ...) dan os.Exit(1).
		zlog.Fatal().Strs("missing_variables", missingVars).Msg("Missing required environment variables. Application cannot start.")
	}

	// Jika semua variabel wajib ada, log pesan sukses.
	// Pesan ini akan muncul setelah logger utama diinisialisasi jika LoadConfig dipanggil sebelum SetupLogger.
	// Jika dipanggil setelah SetupLogger, pesan ini akan langsung menggunakan logger yang sudah dikonfigurasi.
	zlog.Info().Msg("All required environment variables are set. Configuration loaded successfully.")
}
