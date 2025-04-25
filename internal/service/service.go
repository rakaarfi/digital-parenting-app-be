// internal/service/service.go
package service

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
)

// File ini mendefinisikan **interfaces** untuk lapisan Service (Business Logic).
// Interface ini berfungsi sebagai **kontrak** yang menentukan operasi logika bisnis
// yang harus bisa dilakukan oleh implementasi service konkret (misal: *_service_impl.go).
// Handler (di lapisan API) akan bergantung pada interface ini, bukan langsung ke repository.
// Penggunaan interface ini memisahkan logika bisnis dari detail implementasi dan akses data.

// ====================================================================================
// Authentication Service
// ====================================================================================

// AuthService: Kontrak untuk operasi terkait otentikasi pengguna.
type AuthService interface {
	// RegisterUser menangani logika bisnis untuk membuat akun pengguna baru.
	// Termasuk validasi peran (jika ada), hashing kata sandi, dan penyimpanan pengguna.
	// Mengembalikan ID pengguna baru atau error jika terjadi kesalahan.
	RegisterUser(ctx context.Context, input *models.RegisterUserInput) (int, error)

	// LoginUser menangani logika otentikasi pengguna.
	// Memverifikasi kredensial (username/password) dan menghasilkan token JWT jika valid.
	// Mengembalikan string token JWT atau error jika otentikasi gagal.
	LoginUser(ctx context.Context, input *models.LoginUserInput) (string, error)
}

// ====================================================================================
// User Service
// ====================================================================================

// UserService: Kontrak untuk operasi terkait manajemen pengguna dan profil.
type UserService interface {
	// GetUserProfile mengambil detail profil pengguna berdasarkan ID.
	// Tidak termasuk informasi sensitif seperti hash kata sandi.
	// Mengembalikan data pengguna atau error jika tidak ditemukan.
	GetUserProfile(ctx context.Context, userID int) (*models.User, error)

	// UpdateUserProfile menangani pembaruan data profil pengguna yang tidak sensitif.
	// Memerlukan ID pengguna yang akan diperbarui. Mengembalikan error jika terjadi kesalahan.
	UpdateUserProfile(ctx context.Context, userID int, input *models.UpdateProfileInput) error

	// ChangePassword menangani pembaruan kata sandi pengguna.
	// Biasanya memerlukan verifikasi kata sandi lama sebelum mengizinkan perubahan.
	// Mengembalikan error jika terjadi kesalahan atau verifikasi gagal.
	ChangePassword(ctx context.Context, userID int, input *models.UpdatePasswordInput) error

	// CreateChildAccount menangani logika pembuatan akun pengguna anak baru
	// dan secara atomik menautkannya ke akun orang tua yang membuat.
	// Memerlukan ID orang tua yang melakukan permintaan.
	// Mengembalikan ID pengguna anak baru atau error jika terjadi kesalahan.
	CreateChildAccount(ctx context.Context, parentID int, input *models.CreateChildInput) (int, error)

	// // DeleteUser menangani logika bisnis untuk menghapus akun pengguna.
	// // Mungkin memerlukan validasi hak akses (misal, hanya admin atau pengguna sendiri).
	// DeleteUser(ctx context.Context, userID int, requestedByID int) error // Contoh untuk implementasi di masa depan
}

// ====================================================================================
// Task Service
// ====================================================================================

// TaskService: Kontrak untuk operasi terkait logika bisnis Tugas (assignment, verifikasi).
type TaskService interface {
	// VerifyTask menangani logika bisnis saat orang tua memverifikasi tugas yang telah disubmit oleh anak.
	// Memastikan pembaruan status tugas dan transaksi penambahan poin (jika disetujui)
	// dilakukan secara atomik (dalam satu transaksi database).
	// Memerlukan ID UserTask, ID orang tua (untuk validasi), dan status baru (Approved/Rejected).
	// Mengembalikan error jika terjadi kesalahan, validasi gagal, atau operasi database gagal.
	VerifyTask(ctx context.Context, userTaskID int, parentID int, newStatus models.UserTaskStatus) error

	// AssignTask menangani logika bisnis untuk menugaskan tugas kepada anak.
	// Memerlukan ID orang tua (pemberi tugas), ID anak (penerima), dan ID tugas yang akan diberikan.
	// Mengembalikan ID UserTask yang baru dibuat atau error jika terjadi kesalahan (misal, relasi tidak valid, tugas sudah aktif).
	// AssignTask(ctx context.Context, parentID int, childID int, taskID int) (int, error) // Contoh metode lain
}

// ====================================================================================
// Reward Service
// ====================================================================================

// RewardService: Kontrak untuk operasi terkait logika bisnis Hadiah (klaim, review).
type RewardService interface {
	// ClaimReward menangani logika bisnis saat anak mengklaim hadiah yang tersedia.
	// Memeriksa apakah poin anak mencukupi, membuat catatan klaim (UserReward),
	// dan mengurangi poin anak secara atomik (dalam satu transaksi database).
	// Memerlukan ID anak yang mengklaim dan ID hadiah yang diklaim.
	// Mengembalikan ID klaim (UserReward) yang baru dibuat atau error jika terjadi kesalahan
	// (misal, poin tidak cukup, hadiah tidak valid, operasi database gagal).
	ClaimReward(ctx context.Context, childID int, rewardID int) (int, error)

	// ReviewClaim menangani logika bisnis saat orang tua meninjau (menyetujui/menolak)
	// klaim hadiah yang dibuat oleh anak.
	// Memastikan pembaruan status klaim dilakukan secara atomik. Jika klaim ditolak,
	// mungkin perlu ada logika pengembalian poin (tergantung aturan bisnis).
	// Memerlukan ID klaim, ID orang tua (untuk validasi), dan status baru (Approved/Rejected).
	// Mengembalikan error jika terjadi kesalahan, validasi gagal, atau operasi database gagal.
	ReviewClaim(ctx context.Context, claimID int, parentID int, newStatus models.UserRewardStatus) error
}

// ====================================================================================
// Invitation Service
// ====================================================================================

// InvitationService: Kontrak untuk operasi terkait pengelolaan kode undangan
// yang digunakan untuk menautkan akun orang tua baru ke anak yang sudah ada.
type InvitationService interface {
	// GenerateAndStoreCode membuat kode undangan unik untuk anak tertentu,
	// yang diinisiasi oleh orang tua yang sudah terhubung dengan anak tersebut.
	// Menyimpan kode ke database dengan waktu kedaluwarsa.
	// Memerlukan ID orang tua yang meminta (untuk validasi izin) dan ID anak.
	// Mengembalikan string kode undangan yang dihasilkan atau error jika terjadi kesalahan.
	GenerateAndStoreCode(ctx context.Context, requestingParentID int, childID int) (string, error)

	// AcceptInvitation memungkinkan orang tua yang sudah login (joiningParentID)
	// untuk menggunakan kode undangan guna membuat relasi dengan anak yang terkait kode tersebut.
	// Operasi ini harus atomik (menggunakan transaksi database) untuk memastikan
	// kode ditandai sebagai terpakai dan relasi ditambahkan dalam satu kesatuan.
	// Mengembalikan error jika kode tidak valid, kedaluwarsa, sudah digunakan,
	// relasi sudah ada, atau operasi database gagal.
	AcceptInvitation(ctx context.Context, joiningParentID int, code string) error
}

// ====================================================================================
// (Optional) Point Service
// ====================================================================================

// PointService: Kontrak untuk operasi terkait poin secara manual (jika diperlukan).
// Bisa digabungkan ke service lain atau dibiarkan terpisah.
// type PointService interface {
// 	// AdjustPointsManual memungkinkan Admin atau Orang Tua untuk menyesuaikan poin anak secara manual.
// 	// Memerlukan ID pengguna yang melakukan penyesuaian, ID anak, jumlah penyesuaian (bisa positif/negatif), dan catatan.
// 	AdjustPointsManual(ctx context.Context, adminOrParentID int, childID int, amount int, notes string) error
// }

// Anda dapat menambahkan interface service lain sesuai kebutuhan di masa mendatang,
// misalnya: RelationshipService, NotificationService, dll.
