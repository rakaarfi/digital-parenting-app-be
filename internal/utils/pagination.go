// internal/utils/pagination.go
package utils

import (
	"math"    // Digunakan untuk math.Ceil (pembulatan ke atas) untuk menghitung total halaman.
	"strconv" // Digunakan untuk konversi string dari query parameter ke integer.

	"github.com/gofiber/fiber/v2"    // Framework Fiber, digunakan untuk mengakses query parameter dari context (c *fiber.Ctx).
	zlog "github.com/rs/zerolog/log" // Logger global Zerolog untuk mencatat peringatan jika parameter tidak valid.
)

// File ini berisi fungsi-fungsi utilitas dan struktur data yang berkaitan dengan
// implementasi pagination pada response API.

// ====================================================================================
// Konstanta Pagination
// ====================================================================================

// Konstanta ini mendefinisikan nilai default dan batasan untuk parameter pagination.
const (
	DefaultPage  = 1   // Halaman default yang digunakan jika query parameter 'page' tidak ada atau tidak valid.
	DefaultLimit = 10  // Jumlah item default per halaman jika query parameter 'limit' tidak ada atau tidak valid.
	MaxLimit     = 100 // Batas maksimum jumlah item per halaman yang diizinkan untuk mencegah request yang terlalu membebani server.
)

// ====================================================================================
// Struktur dan Fungsi Parsing Parameter Pagination
// ====================================================================================

// PaginationQuery menampung parameter pagination yang sudah dibersihkan (sanitized)
// dan divalidasi dari query string request. Struct ini siap digunakan untuk
// membangun query database (misalnya dengan klausa LIMIT dan OFFSET).
type PaginationQuery struct {
	Page   int // Nomor halaman yang diminta (dimulai dari 1).
	Limit  int // Jumlah item per halaman yang diminta (setelah dibatasi oleh MaxLimit).
	Offset int // Jumlah item yang harus dilewati (skip) dalam query database (dihitung sebagai (Page - 1) * Limit).
}

// ParsePaginationParams membaca dan memvalidasi parameter 'page' dan 'limit'
// dari query string request Fiber (`c.Query`).
// Fungsi ini melakukan hal berikut:
// - Mengambil nilai 'page' dan 'limit' dari query string.
// - Memberikan nilai default (DefaultPage, DefaultLimit) jika parameter tidak ada atau tidak valid (bukan angka positif).
// - Membatasi nilai 'limit' agar tidak melebihi MaxLimit.
// - Menghitung nilai 'offset' yang sesuai untuk digunakan dalam query database.
//
// Parameter:
//   - c: Pointer ke context request Fiber (*fiber.Ctx).
//
// Mengembalikan:
//   - PaginationQuery: Struct yang berisi nilai page, limit, dan offset yang sudah valid dan siap pakai.
func ParsePaginationParams(c *fiber.Ctx) PaginationQuery {
	// --- Ambil dan Validasi Parameter 'page' ---
	pageStr := c.Query("page", strconv.Itoa(DefaultPage)) // Ambil 'page', default ke DefaultPage jika kosong
	page, err := strconv.Atoi(pageStr)
	// Jika error konversi atau nilai page < 1, gunakan DefaultPage.
	if err != nil || page < 1 {
		if pageStr != strconv.Itoa(DefaultPage) { // Hanya log jika nilai query berbeda dari default
			zlog.Warn().Str("query_param", "page").Str("value", pageStr).Int("default", DefaultPage).Msg("Invalid or missing 'page' query parameter, using default")
		}
		page = DefaultPage
	}

	// --- Ambil dan Validasi Parameter 'limit' ---
	limitStr := c.Query("limit", strconv.Itoa(DefaultLimit)) // Ambil 'limit', default ke DefaultLimit jika kosong
	limit, err := strconv.Atoi(limitStr)
	// Jika error konversi atau nilai limit < 1, gunakan DefaultLimit.
	if err != nil || limit < 1 {
		if limitStr != strconv.Itoa(DefaultLimit) { // Hanya log jika nilai query berbeda dari default
			zlog.Warn().Str("query_param", "limit").Str("value", limitStr).Int("default", DefaultLimit).Msg("Invalid or missing 'limit' query parameter, using default")
		}
		limit = DefaultLimit
	}

	// --- Batasi Nilai 'limit' ---
	// Jika limit yang diminta melebihi batas maksimum, gunakan MaxLimit.
	if limit > MaxLimit {
		zlog.Warn().Int("requested_limit", limit).Int("max_limit", MaxLimit).Msg("Requested 'limit' exceeds maximum allowed, capping to max limit")
		limit = MaxLimit
	}

	// --- Hitung Offset ---
	// Offset adalah jumlah baris yang dilewati sebelum mulai mengambil data.
	// Rumus: (Nomor Halaman - 1) * Jumlah Item Per Halaman
	offset := (page - 1) * limit

	// Kembalikan struct PaginationQuery yang sudah terisi.
	return PaginationQuery{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// ====================================================================================
// Struktur dan Fungsi Pembuatan Metadata Pagination
// ====================================================================================

// PaginationMeta berisi informasi metadata yang dikirim bersama data dalam response API
// yang menggunakan pagination. Metadata ini sangat berguna bagi frontend untuk
// membangun kontrol navigasi halaman (misalnya tombol 'next', 'previous', nomor halaman).
type PaginationMeta struct {
	CurrentPage int `json:"current_page"` // Nomor halaman saat ini yang ditampilkan.
	PerPage     int `json:"per_page"`     // Jumlah item per halaman yang digunakan (nilai 'limit' setelah divalidasi).
	TotalItems  int `json:"total_items"`  // Total jumlah item yang tersedia di semua halaman (hasil COUNT(*) dari database).
	TotalPages  int `json:"total_pages"`  // Total jumlah halaman yang tersedia berdasarkan TotalItems dan PerPage.
}

// BuildPaginationMeta menghitung dan membuat struct PaginationMeta berdasarkan
// total item, limit yang digunakan, dan halaman saat ini.
//
// Parameter:
//   - totalItems: Total jumlah item keseluruhan yang cocok dengan query (biasanya hasil COUNT dari database).
//   - limit: Jumlah item per halaman yang digunakan (nilai dari PaginationQuery.Limit).
//   - page: Nomor halaman saat ini (nilai dari PaginationQuery.Page).
//
// Mengembalikan:
//   - PaginationMeta: Struct metadata pagination yang sudah dihitung.
func BuildPaginationMeta(totalItems, limit, page int) PaginationMeta {
	totalPages := 0
	// Hitung total halaman hanya jika ada item dan limit valid (lebih besar dari 0).
	// Ini mencegah pembagian dengan nol.
	if totalItems > 0 && limit > 0 {
		// Gunakan math.Ceil untuk pembulatan ke atas. Contoh:
		// - 10 item / 10 limit = 1 halaman
		// - 11 item / 10 limit = 2 halaman
		// - 20 item / 10 limit = 2 halaman
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	} else if totalItems == 0 {
		totalPages = 0 // Jika tidak ada item, tidak ada halaman
	} else {
		totalPages = 1 // Jika ada item tapi limit tidak valid (seharusnya tidak terjadi), anggap 1 halaman
	}

	// Pastikan CurrentPage tidak melebihi TotalPages (kasus jika page > totalPages diminta)
	// Walaupun data mungkin kosong, metadatanya harus konsisten.
	currentPage := page
	if currentPage > totalPages && totalPages > 0 {
		currentPage = totalPages
	} else if totalPages == 0 && currentPage > 1 {
		currentPage = 1 // Jika tidak ada halaman, kembalikan ke halaman 1
	}


	return PaginationMeta{
		CurrentPage: currentPage,
		PerPage:     limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}

// ====================================================================================
// Struktur Response API Terpaginasi (Menggunakan Generics)
// ====================================================================================

// PaginatedResponse adalah struktur generik (memanfaatkan Go Generics 1.18+)
// yang digunakan untuk membungkus data hasil query yang terpaginasi beserta metadatanya
// dalam format response JSON standar aplikasi.
// Tipe parameter `T` mewakili tipe data dari item individual dalam slice `Data`.
// Contoh penggunaan: `PaginatedResponse[models.User]`, `PaginatedResponse[models.Task]`.
type PaginatedResponse[T any] struct {
	Success bool           `json:"success"` // Menandakan apakah request API berhasil (selalu true untuk response ini).
	Message string         `json:"message"` // Pesan deskriptif (misal: "Users retrieved successfully").
	Data    []T            `json:"data"`    // Slice yang berisi data untuk halaman saat ini. Tipe datanya sesuai dengan `T`.
	Meta    PaginationMeta `json:"meta"`    // Metadata pagination yang berisi informasi halaman.
}

// NewPaginatedResponse adalah fungsi helper (konstruktor) untuk membuat instance
// `PaginatedResponse[T]` dengan lebih ringkas.
//
// Parameter:
//   - message: Pesan deskriptif untuk response.
//   - data: Slice data (`[]T`) untuk halaman saat ini.
//   - meta: Struct `PaginationMeta` yang sudah dihitung.
//
// Mengembalikan:
//   - PaginatedResponse[T]: Instance `PaginatedResponse` yang siap dikirim sebagai JSON.
func NewPaginatedResponse[T any](message string, data []T, meta PaginationMeta) PaginatedResponse[T] {
	if data == nil {
		// Pastikan data tidak nil, tapi slice kosong jika memang tidak ada data.
		// Ini penting untuk konsistensi response JSON (menghasilkan `[]` bukan `null`).
		data = make([]T, 0)
	}
	return PaginatedResponse[T]{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	}
}

// ====================================================================================
// Struktur Response Generik untuk Dokumentasi Swagger
// ====================================================================================

// PaginatedResponseGeneric adalah struktur yang digunakan KHUSUS untuk keperluan
// dokumentasi Swagger/OpenAPI. Karena Swagger v2 (yang umum digunakan dengan `swaggo`)
// tidak sepenuhnya mendukung generics Go, kita menggunakan `[]interface{}` untuk `Data`
// sebagai representasi generik dalam dokumentasi.
// JANGAN gunakan struct ini dalam kode Go aktual untuk response API.
// Gunakan `PaginatedResponse[T]` sebagai gantinya.
type PaginatedResponseGeneric struct {
	Success bool           `json:"success"`           // Status keberhasilan.
	Message string         `json:"message"`           // Pesan deskriptif.
	Data    []interface{}  `json:"data"`              // Representasi data generik untuk Swagger.
	Meta    PaginationMeta `json:"meta"`              // Metadata pagination.
}
