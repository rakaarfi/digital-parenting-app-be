// internal/repository/repository.go
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
)

// File ini mendefinisikan **interfaces** untuk Data Access Layer (DAL).
// Interface ini berfungsi sebagai **kontrak** yang menentukan operasi data apa saja
// yang harus bisa dilakukan oleh implementasi repository konkret (misal: *_repo.go).
// Penggunaan interface memungkinkan decoupling (pemisahan) antara lapisan handler/service
// dengan implementasi akses data spesifik (misal: PostgreSQL).

// ====================================================================================
// User Repository
// ====================================================================================

// UserRepository: Kontrak untuk operasi data terkait Pengguna (User).
type UserRepository interface {
	// CreateUser membuat pengguna baru dalam database.
	// Mengembalikan ID pengguna baru atau error jika terjadi kesalahan.
	CreateUser(ctx context.Context, user *models.RegisterUserInput, hashedPassword string) (int, error)

	// GetUserByUsername mencari pengguna berdasarkan username.
	// Mengembalikan data pengguna (termasuk role) atau error jika tidak ditemukan.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// GetUserByID mencari pengguna berdasarkan ID.
	// Mengembalikan data pengguna (termasuk role) atau error jika tidak ditemukan.
	GetUserByID(ctx context.Context, id int) (*models.User, error)

	// DeleteUserByID menghapus pengguna berdasarkan ID.
	// Mengembalikan error jika terjadi kesalahan.
	DeleteUserByID(ctx context.Context, id int) error

	// GetAllUsers mendapatkan daftar semua pengguna dengan paginasi.
	// Mengembalikan slice pengguna, total jumlah pengguna, dan error jika ada.
	GetAllUsers(ctx context.Context, page, limit int) ([]models.User, int, error)

	// UpdateUserByID memperbarui data pengguna berdasarkan ID (biasanya oleh Admin).
	// Mengembalikan error jika terjadi kesalahan.
	UpdateUserByID(ctx context.Context, id int, input *models.AdminUpdateUserInput) error

	// UpdateUserPassword memperbarui kata sandi pengguna berdasarkan ID.
	// Menggunakan kata sandi yang sudah di-hash. Mengembalikan error jika terjadi kesalahan.
	UpdateUserPassword(ctx context.Context, id int, hashedPassword string) error

	// UpdateUserProfile memperbarui profil pengguna berdasarkan ID (oleh pengguna sendiri).
	// Mengembalikan error jika terjadi kesalahan.
	UpdateUserProfile(ctx context.Context, id int, input *models.UpdateProfileInput) error

	// --- Metode Transaksional ---

	// CreateUserTx membuat pengguna baru dalam konteks transaksi database.
	// Mengembalikan ID pengguna baru atau error jika terjadi kesalahan.
	CreateUserTx(ctx context.Context, tx pgx.Tx, user *models.RegisterUserInput, hashedPassword string) (int, error)
}

// ====================================================================================
// Role Repository
// ====================================================================================

// RoleRepository: Kontrak untuk operasi data terkait Peran (Role).
type RoleRepository interface {
	// CreateRole membuat peran baru dalam database.
	// Mengembalikan ID peran baru atau error jika terjadi kesalahan.
	CreateRole(ctx context.Context, role *models.Role) (int, error)

	// GetRoleByID mencari peran berdasarkan ID.
	// Mengembalikan data peran atau error jika tidak ditemukan.
	GetRoleByID(ctx context.Context, id int) (*models.Role, error)

	// GetAllRoles mendapatkan daftar semua peran.
	// Mengembalikan slice peran atau error jika terjadi kesalahan.
	GetAllRoles(ctx context.Context) ([]models.Role, error)

	// UpdateRole memperbarui data peran berdasarkan ID.
	// Mengembalikan error jika terjadi kesalahan.
	UpdateRole(ctx context.Context, role *models.Role) error

	// DeleteRole menghapus peran berdasarkan ID.
	// Perlu memeriksa dependensi pengguna sebelum menghapus. Mengembalikan error jika terjadi kesalahan.
	DeleteRole(ctx context.Context, id int) error
}

// ====================================================================================
// User Relationship Repository
// ====================================================================================

// UserRelationshipRepository: Kontrak untuk operasi data terkait relasi Orang Tua-Anak.
type UserRelationshipRepository interface {
	// AddRelationship menambahkan relasi baru antara orang tua dan anak.
	// Mengembalikan error jika terjadi kesalahan.
	AddRelationship(ctx context.Context, parentID int, childID int) error

	// GetChildrenByParentID mendapatkan daftar anak dari seorang orang tua.
	// Mengembalikan slice data pengguna (anak) atau error jika terjadi kesalahan.
	GetChildrenByParentID(ctx context.Context, parentID int) ([]models.User, error)

	// GetParentsByChildID mendapatkan daftar orang tua dari seorang anak.
	// Mengembalikan slice data pengguna (orang tua) atau error jika terjadi kesalahan.
	GetParentsByChildID(ctx context.Context, childID int) ([]models.User, error)

	// IsParentOf memeriksa apakah relasi orang tua-anak tertentu ada.
	// Mengembalikan boolean (true jika ada) dan error jika terjadi kesalahan.
	IsParentOf(ctx context.Context, parentID int, childID int) (bool, error)

	// RemoveRelationship menghapus relasi antara orang tua dan anak.
	// Mengembalikan error jika terjadi kesalahan.
	RemoveRelationship(ctx context.Context, parentID int, childID int) error

	// HasSharedChild memeriksa apakah dua orang tua memiliki setidaknya satu anak yang sama.
	// Mengembalikan boolean (true jika ada) dan error jika terjadi kesalahan.
	HasSharedChild(ctx context.Context, parentID1 int, parentID2 int) (bool, error)

	// --- Metode Transaksional ---

	// IsParentOfTx memeriksa apakah relasi orang tua-anak ada dalam konteks transaksi.
	// Mengembalikan boolean (true jika ada) dan error jika terjadi kesalahan.
	IsParentOfTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) (bool, error)

	// AddRelationshipTx menambahkan relasi baru dalam konteks transaksi.
	// Mengembalikan error jika terjadi kesalahan.
	AddRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error

	// GetParentIDsByChildIDTx mendapatkan daftar ID orang tua dari seorang anak dalam konteks transaksi.
	// Mengembalikan slice ID orang tua atau error jika terjadi kesalahan.
	GetParentIDsByChildIDTx(ctx context.Context, tx pgx.Tx, childID int) ([]int, error)

	// HasSharedChildTx memeriksa apakah dua orang tua memiliki anak yang sama dalam konteks transaksi.
	// Mengembalikan boolean (true jika ada) dan error jika terjadi kesalahan.
	HasSharedChildTx(ctx context.Context, tx pgx.Tx, parentID1 int, parentID2 int) (bool, error)
}

// ====================================================================================
// Task Repository
// ====================================================================================

// TaskRepository: Kontrak untuk operasi data terkait definisi Tugas (Task).
type TaskRepository interface {
	// CreateTask membuat definisi tugas baru.
	// Mengembalikan ID tugas baru atau error jika terjadi kesalahan.
	CreateTask(ctx context.Context, task *models.Task) (int, error)

	// GetTaskByID mencari definisi tugas berdasarkan ID.
	// Memerlukan parentID untuk validasi kepemilikan. Mengembalikan data tugas atau error.
	GetTaskByID(ctx context.Context, id int) (*models.Task, error)

	// GetTasksByCreatorID mendapatkan daftar tugas yang dibuat oleh pengguna tertentu (orang tua) dengan paginasi.
	// Mengembalikan slice tugas, total jumlah tugas, dan error jika ada.
	GetTasksByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Task, int, error)

	// UpdateTask memperbarui definisi tugas.
	// Memerlukan parentID untuk validasi kepemilikan. Mengembalikan error jika terjadi kesalahan.
	UpdateTask(ctx context.Context, task *models.Task, parentID int) error

	// DeleteTask menghapus definisi tugas.
	// Memerlukan parentID untuk validasi kepemilikan. Mengembalikan error jika terjadi kesalahan.
	DeleteTask(ctx context.Context, id int, parentID int) error
}

// ====================================================================================
// User Task Repository
// ====================================================================================

// TaskVerificationDetails adalah struct helper untuk membawa data
// yang diperlukan saat verifikasi tugas dalam transaksi.
type TaskVerificationDetails struct {
	ChildID       int                 // ID Anak yang mengerjakan tugas
	CurrentStatus models.UserTaskStatus // Status tugas saat ini sebelum verifikasi
	TaskPoint     int                 // Jumlah poin yang terkait dengan tugas
}

// UserTaskRepository: Kontrak untuk operasi data terkait Tugas yang Ditugaskan kepada Pengguna (UserTask).
type UserTaskRepository interface {
	// AssignTask menugaskan sebuah tugas kepada pengguna (anak).
	// Mengembalikan ID UserTask baru atau error jika terjadi kesalahan.
	AssignTask(ctx context.Context, userID, taskID, assignedByID int) (int, error)

	// GetUserTaskByID mencari tugas yang ditugaskan berdasarkan ID UserTask.
	// Mengembalikan data UserTask (mungkin perlu JOIN dengan Task dan User) atau error.
	GetUserTaskByID(ctx context.Context, id int) (*models.UserTask, error)

	// GetTasksByChildID mendapatkan daftar tugas yang ditugaskan kepada anak tertentu, dengan filter status dan paginasi.
	// Mengembalikan slice UserTask, total jumlah, dan error jika ada.
	GetTasksByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserTask, int, error)

	// GetTasksByParentID mendapatkan daftar tugas yang ditugaskan kepada anak-anak dari orang tua tertentu, dengan filter status dan paginasi.
	// Mengembalikan slice UserTask, total jumlah, dan error jika ada.
	GetTasksByParentID(ctx context.Context, parentID int, statusFilter string, page, limit int) ([]models.UserTask, int, error)

	// UpdateUserTaskStatus memperbarui status tugas yang ditugaskan.
	// verifierID bersifat opsional (digunakan saat orang tua memverifikasi). Mengembalikan error jika terjadi kesalahan.
	UpdateUserTaskStatus(ctx context.Context, id int, newStatus models.UserTaskStatus, verifierID *int) error

	// SubmitTask menandai tugas sebagai selesai oleh anak.
	// Memerlukan childID untuk validasi kepemilikan. Mengembalikan error jika terjadi kesalahan.
	SubmitTask(ctx context.Context, id int, childID int) error

	// VerifyTask memverifikasi tugas yang telah disubmit oleh anak (dilakukan oleh orang tua).
	// Memerlukan parentID untuk validasi relasi. Mengembalikan data UserTask yang diperbarui atau error.
	VerifyTask(ctx context.Context, id int, parentID int, newStatus models.UserTaskStatus) (*models.UserTask, error)

	// CheckExistingActiveTask memeriksa apakah anak sudah memiliki tugas yang sama yang masih aktif (belum selesai/ditolak).
	// Mengembalikan boolean (true jika ada) dan error jika terjadi kesalahan.
	CheckExistingActiveTask(ctx context.Context, userID, taskID int) (bool, error)

	// --- Metode Transaksional ---

	// GetTaskDetailsForVerificationTx mendapatkan detail tugas yang diperlukan untuk proses verifikasi dalam konteks transaksi.
	// Mengembalikan detail verifikasi atau error.
	GetTaskDetailsForVerificationTx(ctx context.Context, tx pgx.Tx, userTaskID int) (*TaskVerificationDetails, error)

	// UpdateStatusTx memperbarui status tugas dalam konteks transaksi.
	// Memerlukan verifierID. Mengembalikan error jika terjadi kesalahan.
	UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserTaskStatus, verifierID int) error
}

// ====================================================================================
// Reward Repository
// ====================================================================================

// RewardDetails adalah struct helper untuk membawa data yang diperlukan
// saat klaim reward dalam transaksi.
type RewardDetails struct {
	ID              int // ID Reward
	RequiredPoints  int // Poin yang dibutuhkan untuk klaim
	CreatedByUserID int // ID Pengguna (Orang Tua) yang membuat reward
}

// RewardRepository: Kontrak untuk operasi data terkait definisi Hadiah (Reward).
type RewardRepository interface {
	// CreateReward membuat definisi hadiah baru.
	// Mengembalikan ID hadiah baru atau error jika terjadi kesalahan.
	CreateReward(ctx context.Context, reward *models.Reward) (int, error)

	// GetRewardByID mencari definisi hadiah berdasarkan ID.
	// Mengembalikan data hadiah atau error jika tidak ditemukan.
	GetRewardByID(ctx context.Context, id int) (*models.Reward, error)

	// GetRewardsByCreatorID mendapatkan daftar hadiah yang dibuat oleh pengguna tertentu (orang tua) dengan paginasi.
	// Mengembalikan slice hadiah, total jumlah hadiah, dan error jika ada.
	GetRewardsByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Reward, int, error)

	// GetAvailableRewardsForChild mendapatkan daftar hadiah yang tersedia untuk anak tertentu (dari orang tuanya) dengan paginasi.
	// Mengembalikan slice hadiah, total jumlah hadiah, dan error jika ada.
	GetAvailableRewardsForChild(ctx context.Context, childID int, page, limit int) ([]models.Reward, int, error)

	// UpdateReward memperbarui definisi hadiah.
	// Memerlukan parentID untuk validasi kepemilikan. Mengembalikan error jika terjadi kesalahan.
	UpdateReward(ctx context.Context, reward *models.Reward, parentID int) error

	// DeleteReward menghapus definisi hadiah.
	// Memerlukan parentID untuk validasi kepemilikan. Mengembalikan error jika terjadi kesalahan.
	DeleteReward(ctx context.Context, id int, parentID int) error

	// --- Metode Transaksional ---

	// GetRewardDetailsTx mendapatkan detail hadiah yang diperlukan untuk proses klaim dalam konteks transaksi.
	// Mengembalikan detail hadiah atau error.
	GetRewardDetailsTx(ctx context.Context, tx pgx.Tx, rewardID int) (*RewardDetails, error)
}

// ====================================================================================
// User Reward Repository
// ====================================================================================

// ClaimReviewDetails membawa data minimal untuk proses review klaim hadiah.
type ClaimReviewDetails struct {
	ChildID         int                   // ID Anak yang mengklaim
	CurrentStatus   models.UserRewardStatus // Status klaim saat ini sebelum direview
	PointsDeducted  int                   // Jumlah poin yang dikurangkan saat klaim dibuat
	RewardCreatorID int                   // ID Orang Tua yang membuat reward (untuk validasi reviewer)
}

// UserRewardRepository: Kontrak untuk operasi data terkait Klaim Hadiah oleh Pengguna (UserReward).
type UserRewardRepository interface {
	// CreateClaim membuat catatan klaim hadiah baru oleh anak.
	// Memerlukan pointsDeducted yang sudah dihitung sebelumnya. Mengembalikan ID UserReward baru atau error.
	CreateClaim(ctx context.Context, userID, rewardID, pointsDeducted int) (int, error)

	// GetUserRewardByID mencari klaim hadiah berdasarkan ID UserReward.
	// Mengembalikan data UserReward (mungkin perlu JOIN dengan Reward dan User) atau error.
	GetUserRewardByID(ctx context.Context, id int) (*models.UserReward, error)

	// GetClaimsByChildID mendapatkan daftar klaim hadiah oleh anak tertentu, dengan filter status dan paginasi.
	// Mengembalikan slice UserReward, total jumlah, dan error jika ada.
	GetClaimsByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserReward, int, error)

	// GetPendingClaimsByParentID mendapatkan daftar klaim hadiah yang menunggu persetujuan dari anak-anak orang tua tertentu, dengan paginasi.
	// Mengembalikan slice UserReward, total jumlah, dan error jika ada.
	GetPendingClaimsByParentID(ctx context.Context, parentID int, page, limit int) ([]models.UserReward, int, error)

	// UpdateClaimStatus memperbarui status klaim hadiah (disetujui/ditolak oleh orang tua).
	// Memerlukan reviewerID (parentID) untuk validasi. Mengembalikan error jika terjadi kesalahan.
	UpdateClaimStatus(ctx context.Context, id int, newStatus models.UserRewardStatus, reviewerID int) error

	// --- Metode Transaksional ---

	// CreateClaimTx membuat catatan klaim hadiah baru dalam konteks transaksi.
	// Mengembalikan ID UserReward baru atau error.
	CreateClaimTx(ctx context.Context, tx pgx.Tx, userID, rewardID, pointsDeducted int) (int, error)

	// UpdateClaimStatusTx memperbarui status klaim hadiah dalam konteks transaksi.
	// Mengembalikan error jika terjadi kesalahan.
	UpdateClaimStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserRewardStatus, reviewerID int) error

	// GetClaimDetailsForReviewTx mendapatkan detail klaim yang diperlukan untuk proses review dalam konteks transaksi.
	// Mengembalikan detail klaim atau error.
	GetClaimDetailsForReviewTx(ctx context.Context, tx pgx.Tx, claimID int) (*ClaimReviewDetails, error)
}

// ====================================================================================
// Point Transaction Repository
// ====================================================================================

// PointTransactionRepository: Kontrak untuk operasi data terkait Transaksi Poin.
type PointTransactionRepository interface {
	// CreateTransaction mencatat transaksi poin baru (penambahan/pengurangan).
	// Mengembalikan error jika terjadi kesalahan.
	CreateTransaction(ctx context.Context, txData *models.PointTransaction) error

	// GetTransactionsByUserID mendapatkan riwayat transaksi poin untuk pengguna tertentu (anak) dengan paginasi.
	// Mengembalikan slice transaksi, total jumlah transaksi, dan error jika ada.
	GetTransactionsByUserID(ctx context.Context, userID int, page, limit int) ([]models.PointTransaction, int, error)

	// CalculateTotalPointsByUserID menghitung total poin terkini untuk pengguna tertentu (anak).
	// Mengembalikan total poin atau error jika terjadi kesalahan.
	CalculateTotalPointsByUserID(ctx context.Context, userID int) (int, error)

	// --- Metode Transaksional ---

	// CreateTransactionTx mencatat transaksi poin baru dalam konteks transaksi.
	// Mengembalikan error jika terjadi kesalahan.
	CreateTransactionTx(ctx context.Context, tx pgx.Tx, txData *models.PointTransaction) error

	// CalculateTotalPointsByUserIDTx menghitung total poin terkini dalam konteks transaksi.
	// Mengembalikan total poin atau error jika terjadi kesalahan.
	CalculateTotalPointsByUserIDTx(ctx context.Context, tx pgx.Tx, userID int) (int, error)
}

// ====================================================================================
// Invitation Code Repository
// ====================================================================================

// InvitationCodeRepository: Kontrak untuk operasi data terkait Kode Undangan.
type InvitationCodeRepository interface {
	// CreateCode menyimpan kode undangan baru ke database.
	// Mengembalikan error jika terjadi kesalahan.
	CreateCode(ctx context.Context, code string, childID int, parentID int, expiresAt time.Time) error

	// FindActiveCode mencari kode undangan berdasarkan nilainya, hanya jika aktif dan belum kedaluwarsa.
	// Mengembalikan detail InvitationCode atau pgx.ErrNoRows jika tidak ditemukan/tidak valid.
	FindActiveCode(ctx context.Context, code string) (*models.InvitationCode, error)

	// DeleteExpiredCodes (Opsional) menghapus kode undangan yang sudah melewati tanggal kedaluwarsa.
	// Mengembalikan jumlah baris yang dihapus atau error.
	DeleteExpiredCodes(ctx context.Context) (int64, error)

	// --- Metode Transaksional ---

	// MarkCodeAsUsedTx menandai status kode tertentu menjadi 'used' dalam konteks transaksi database.
	// Mengembalikan error jika kode tidak ditemukan atau gagal diperbarui.
	MarkCodeAsUsedTx(ctx context.Context, tx pgx.Tx, code string) error
}
