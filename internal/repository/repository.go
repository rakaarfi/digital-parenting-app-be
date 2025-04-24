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

// UserRepository: Kontrak untuk operasi data User.
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.RegisterUserInput, hashedPassword string) (int, error) // Buat user baru.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)                       // Cari user by username (termasuk role).
	GetUserByID(ctx context.Context, id int) (*models.User, error)                                      // Cari user by ID (termasuk role).
	DeleteUserByID(ctx context.Context, id int) error                                                   // Hapus user by ID.
	GetAllUsers(ctx context.Context, page, limit int) ([]models.User, int, error)                       // Dapatkan semua user (paginated, termasuk role).
	UpdateUserByID(ctx context.Context, id int, input *models.AdminUpdateUserInput) error               // Update user by ID (oleh Admin).
	UpdateUserPassword(ctx context.Context, id int, hashedPassword string) error                        // Update password user by ID (dengan hash).
	UpdateUserProfile(ctx context.Context, id int, input *models.UpdateProfileInput) error              // Update profil user by ID (oleh user sendiri).
	CreateUserTx(ctx context.Context, tx pgx.Tx, user *models.RegisterUserInput, hashedPassword string) (int, error)
}

// RoleRepository: Kontrak untuk operasi data Role.
type RoleRepository interface {
	CreateRole(ctx context.Context, role *models.Role) (int, error) // Buat role baru.
	GetRoleByID(ctx context.Context, id int) (*models.Role, error)  // Cari role by ID.
	GetAllRoles(ctx context.Context) ([]models.Role, error)         // Dapatkan semua role.
	UpdateRole(ctx context.Context, role *models.Role) error        // Update role by ID.
	DeleteRole(ctx context.Context, id int) error                   // Hapus role by ID (cek dependensi user).
}

// UserRelationshipRepository: Kontrak untuk relasi Parent-Child
type UserRelationshipRepository interface {
	AddRelationship(ctx context.Context, parentID int, childID int) error
	GetChildrenByParentID(ctx context.Context, parentID int) ([]models.User, error) // Mengembalikan data anak
	GetParentsByChildID(ctx context.Context, childID int) ([]models.User, error)    // Mengembalikan data parent
	IsParentOf(ctx context.Context, parentID int, childID int) (bool, error)        // Cek apakah relasi ada
	RemoveRelationship(ctx context.Context, parentID int, childID int) error
	HasSharedChild(ctx context.Context, parentID1 int, parentID2 int) (bool, error)
	IsParentOfTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) (bool, error)
	AddRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error
	GetParentIDsByChildIDTx(ctx context.Context, tx pgx.Tx, childID int) ([]int, error)
	HasSharedChildTx(ctx context.Context, tx pgx.Tx, parentID1 int, parentID2 int) (bool, error)
}

// TaskRepository: Kontrak untuk definisi Task
type TaskRepository interface {
	CreateTask(ctx context.Context, task *models.Task) (int, error)                                      // ID Task baru
	GetTaskByID(ctx context.Context, id int) (*models.Task, error)                         // ParentID untuk cek kepemilikan
	GetTasksByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Task, int, error) // Pagination
	UpdateTask(ctx context.Context, task *models.Task, parentID int) error                               // ParentID untuk cek kepemilikan
	DeleteTask(ctx context.Context, id int, parentID int) error                                          // ParentID untuk cek kepemilikan
}

// TaskVerificationDetails adalah struct helper untuk membawa data
// yang diperlukan saat verifikasi task dalam transaksi.
type TaskVerificationDetails struct {
	ChildID       int
	CurrentStatus models.UserTaskStatus
	TaskPoint     int
}

// UserTaskRepository: Kontrak untuk Task yang ditugaskan
type UserTaskRepository interface {
	AssignTask(ctx context.Context, userID, taskID, assignedByID int) (int, error)                                              // ID UserTask baru
	GetUserTaskByID(ctx context.Context, id int) (*models.UserTask, error)                                                      // Perlu JOIN data task & user?
	GetTasksByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserTask, int, error)   // Filter by status, pagination
	GetTasksByParentID(ctx context.Context, parentID int, statusFilter string, page, limit int) ([]models.UserTask, int, error) // View tugas anak-anaknya, pagination
	UpdateUserTaskStatus(ctx context.Context, id int, newStatus models.UserTaskStatus, verifierID *int) error                   // VerifierID opsional (untuk parent)
	SubmitTask(ctx context.Context, id int, childID int) error                                                                  // Child submit, cek ownership
	VerifyTask(ctx context.Context, id int, parentID int, newStatus models.UserTaskStatus) (*models.UserTask, error)            // Parent verify, cek relasi, return UserTask untuk dapatkan poin
	GetTaskDetailsForVerificationTx(ctx context.Context, tx pgx.Tx, userTaskID int) (*TaskVerificationDetails, error)           // Definisikan struct TaskVerificationDetails
	UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserTaskStatus, verifierID int) error
	CheckExistingActiveTask(ctx context.Context, userID, taskID int) (bool, error)
}

// RewardDetails adalah struct helper untuk membawa data yang diperlukan
// saat claim reward dalam transaksi.
type RewardDetails struct {
	ID             int
	RequiredPoints int
	CreatedByUserID int
	// Tambahkan field lain jika perlu
}

// RewardRepository: Kontrak untuk definisi Reward
type RewardRepository interface {
	CreateReward(ctx context.Context, reward *models.Reward) (int, error)
	GetRewardByID(ctx context.Context, id int) (*models.Reward, error) // Cek kepemilikan? Atau semua parent bisa lihat?
	GetRewardsByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Reward, int, error)
	GetAvailableRewardsForChild(ctx context.Context, childID int, page, limit int) ([]models.Reward, int, error) // Tampilkan reward dari parent si anak
	UpdateReward(ctx context.Context, reward *models.Reward, parentID int) error
	DeleteReward(ctx context.Context, id int, parentID int) error
	GetRewardDetailsTx(ctx context.Context, tx pgx.Tx, rewardID int) (*RewardDetails, error)
}

// ClaimReviewDetails membawa data minimal untuk proses review klaim.
type ClaimReviewDetails struct {
	ChildID        int
	CurrentStatus  models.UserRewardStatus
	PointsDeducted int
	RewardCreatorID int
}

// UserRewardRepository: Kontrak untuk Klaim Reward
type UserRewardRepository interface {
	CreateClaim(ctx context.Context, userID, rewardID, pointsDeducted int) (int, error) // ID UserReward baru
	GetUserRewardByID(ctx context.Context, id int) (*models.UserReward, error)          // Perlu JOIN data reward & user?
	GetClaimsByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserReward, int, error)
	GetPendingClaimsByParentID(ctx context.Context, parentID int, page, limit int) ([]models.UserReward, int, error) // Parent lihat klaim anak-anaknya
	UpdateClaimStatus(ctx context.Context, id int, newStatus models.UserRewardStatus, reviewerID int) error          // Parent review
	CreateClaimTx(ctx context.Context, tx pgx.Tx, userID, rewardID, pointsDeducted int) (int, error)
	UpdateClaimStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserRewardStatus, reviewerID int) error // Jika diperlukan service ReviewClaim
	GetClaimDetailsForReviewTx(ctx context.Context, tx pgx.Tx, claimID int) (*ClaimReviewDetails, error)
}

// PointTransactionRepository: Kontrak untuk Transaksi Poin
type PointTransactionRepository interface {
	CreateTransaction(ctx context.Context, txData *models.PointTransaction) error                                     // Nama metode asli mungkin lebih baik txData daripada tx
	GetTransactionsByUserID(ctx context.Context, userID int, page, limit int) ([]models.PointTransaction, int, error) // History poin anak
	CalculateTotalPointsByUserID(ctx context.Context, userID int) (int, error)                                        // Hitung total poin anak
	CreateTransactionTx(ctx context.Context, tx pgx.Tx, txData *models.PointTransaction) error
	CalculateTotalPointsByUserIDTx(ctx context.Context, tx pgx.Tx, userID int) (int, error)
}

// InvitationCodeRepository defines the contract for data operations related to invitation codes.
type InvitationCodeRepository interface {
	// CreateCode stores a new invitation code in the database.
	CreateCode(ctx context.Context, code string, childID int, parentID int, expiresAt time.Time) error

	// FindActiveCode retrieves an invitation code by its value, but only if it's currently active and not expired.
	// Returns the InvitationCode details or pgx.ErrNoRows if not found or not active/valid.
	FindActiveCode(ctx context.Context, code string) (*models.InvitationCode, error)

	// MarkCodeAsUsedTx updates the status of a specific code to 'used' within a database transaction.
	// Returns an error if the code cannot be found or updated.
	MarkCodeAsUsedTx(ctx context.Context, tx pgx.Tx, code string) error

	// DeleteExpiredCodes (Optional) removes invitation codes that have passed their expiration date.
	// Returns the number of rows deleted or an error.
	DeleteExpiredCodes(ctx context.Context) (int64, error)
}
