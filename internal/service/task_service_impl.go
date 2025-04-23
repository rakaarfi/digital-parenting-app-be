// internal/service/task_service_impl.go
package service

import (
	"context"
	"errors" // Import errors
	"fmt"

	"github.com/jackc/pgx/v5" // Import pgx for ErrNoRows etc.
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	zlog "github.com/rs/zerolog/log"
)

// taskServiceImpl implements the TaskService interface.
type taskServiceImpl struct {
	pool         *pgxpool.Pool // Pool dibutuhkan untuk memulai transaksi
	userTaskRepo repository.UserTaskRepository
	// taskRepo     repository.TaskRepository // Mungkin tidak perlu jika userTaskRepo.VerifyTaskTx sudah cukup
	pointRepo   repository.PointTransactionRepository
	userRelRepo repository.UserRelationshipRepository // Dibutuhkan untuk cek relasi
}

// NewTaskService creates a new instance of TaskService.
func NewTaskService(
	pool *pgxpool.Pool,
	userTaskRepo repository.UserTaskRepository,
	pointRepo repository.PointTransactionRepository,
	userRelRepo repository.UserRelationshipRepository,
) TaskService {
	return &taskServiceImpl{
		pool:         pool,
		userTaskRepo: userTaskRepo,
		pointRepo:    pointRepo,
		userRelRepo:  userRelRepo,
	}
}

// VerifyTask implements the business logic for verifying a task, including transaction management.
func (s *taskServiceImpl) VerifyTask(ctx context.Context, userTaskID int, parentID int, newStatus models.UserTaskStatus) (err error) { // Gunakan named return error agar defer bisa modifikasi
	// --- 1. Mulai Transaksi Database ---
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		zlog.Error().Err(err).Msg("Service: Failed to begin transaction for task verification")
		return fmt.Errorf("internal server error: could not start operation") // Error generik ke handler
	}

	// --- 2. Defer untuk Rollback atau Commit ---
	defer func() {
		if p := recover(); p != nil {
			// Jika terjadi panic, rollback dan re-panic
			zlog.Error().Msgf("Service: Panic recovered during task verification: %v", p)
			_ = tx.Rollback(ctx) // Abaikan error rollback saat panic
			panic(p)             // Re-panic agar bisa ditangkap recover middleware
		} else if err != nil {
			// Jika ada error (dari langkah berikutnya), rollback
			zlog.Warn().Err(err).Int("user_task_id", userTaskID).Int("parent_id", parentID).Msg("Service: Rolling back transaction due to error during task verification")
			rbErr := tx.Rollback(ctx)
			if rbErr != nil {
				// Log error rollback tambahan jika ada
				zlog.Error().Err(rbErr).Msg("Service: Failed to rollback transaction")
			}
			// Jangan override error asli dengan error rollback
		} else {
			// Jika tidak ada error, commit transaksi
			err = tx.Commit(ctx)
			if err != nil {
				zlog.Error().Err(err).Int("user_task_id", userTaskID).Int("parent_id", parentID).Msg("Service: Failed to commit transaction for task verification")
				err = fmt.Errorf("internal server error: could not finalize operation") // Set error jika commit gagal
			} else {
				zlog.Info().Int("user_task_id", userTaskID).Msg("Service: Transaction committed successfully for task verification")
			}
		}
	}()

	// --- 3. Modifikasi Metode Repository untuk Menerima Transaksi ---
	// Kita perlu metode di repository yang menerima pgx.Tx sebagai argumen.
	// Mari kita asumsikan kita punya metode berikut (perlu dibuat di repo):
	// - userTaskRepo.GetTaskForVerificationTx(ctx, tx, userTaskID) -> (*models.UserTaskSimple, error) // Hanya data minimal yg perlu
	// - userRelRepo.IsParentOfTx(ctx, tx, parentID, childID) -> (bool, error)
	// - userTaskRepo.UpdateStatusTx(ctx, tx, userTaskID, newStatus, parentID, completedAt) -> error
	// - pointRepo.CreateTransactionTx(ctx, tx, *models.PointTransaction) -> error

	// --- 4. Logika Bisnis Inti dalam Transaksi ---

	// 4a. Dapatkan detail dasar UserTask (user_id, status, task_id, task_point)
	// Asumsi metode repo baru `GetTaskDetailsForVerificationTx` mengembalikan struct sederhana
	// Anda perlu membuat metode ini di user_task_repo.go
	var taskDetails *repository.TaskVerificationDetails // Deklarasi tipe dari paket repo
	taskDetails, err = s.userTaskRepo.GetTaskDetailsForVerificationTx(ctx, tx, userTaskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_task_id", userTaskID).Msg("Service: UserTask not found for verification")
			// err sudah di-set, defer akan rollback
			return fmt.Errorf("task assignment not found") // Error spesifik ke handler
		}
		zlog.Error().Err(err).Int("user_task_id", userTaskID).Msg("Service: Error fetching task details for verification")
		// err sudah di-set, defer akan rollback
		return fmt.Errorf("internal server error: could not retrieve task details")
	}

	// 4b. Validasi Status Saat Ini
	if taskDetails.CurrentStatus != models.UserTaskStatusSubmitted {
		zlog.Warn().Int("user_task_id", userTaskID).Str("current_status", string(taskDetails.CurrentStatus)).Msg("Service: Verify task failed: Task not in 'submitted' status")
		err = fmt.Errorf("cannot verify task: current status is '%s', expected 'submitted'", taskDetails.CurrentStatus)
		return err // Rollback
	}

	// 4c. Validasi Relasi Parent-Child (gunakan UserRelationshipRepository)
	// Asumsi metode repo baru `IsParentOfTx`
	// Anda perlu membuat metode ini di user_relationship_repo.go
	isParent, err := s.userRelRepo.IsParentOfTx(ctx, tx, parentID, taskDetails.ChildID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", taskDetails.ChildID).Msg("Service: Error checking parent-child relationship during task verification")
		err = fmt.Errorf("internal server error: could not verify relationship")
		return err // Rollback
	}
	if !isParent {
		zlog.Warn().Int("user_task_id", userTaskID).Int("parent_id", parentID).Int("child_id", taskDetails.ChildID).Msg("Service: Verify task failed: Requesting user is not the parent")
		err = fmt.Errorf("forbidden: you are not authorized to verify tasks for this child")
		return err // Rollback
	}

	// 4d. Update Status UserTask dalam Transaksi
	// Asumsi metode repo baru `UpdateStatusTx`
	// Anda perlu membuat metode ini di user_task_repo.go
	err = s.userTaskRepo.UpdateStatusTx(ctx, tx, userTaskID, newStatus, parentID)
	if err != nil {
		zlog.Error().Err(err).Int("user_task_id", userTaskID).Str("new_status", string(newStatus)).Msg("Service: Failed to update task status within transaction")
		err = fmt.Errorf("internal server error: could not update task status")
		return err // Rollback
	}

	// 4e. Jika Approved, Buat Transaksi Poin dalam Transaksi DB
	if newStatus == models.UserTaskStatusApproved {
		if taskDetails.TaskPoint > 0 {
			pointTx := &models.PointTransaction{
				UserID:            taskDetails.ChildID,
				ChangeAmount:      taskDetails.TaskPoint,
				TransactionType:   models.TransactionTypeCompletion,
				RelatedUserTaskID: userTaskID,
				CreatedByUserID:   parentID,
			}
			// Asumsi metode repo baru `CreateTransactionTx`
			// Anda perlu membuat metode ini di point_transaction_repo.go (buat file ini)
			err = s.pointRepo.CreateTransactionTx(ctx, tx, pointTx)
			if err != nil {
				zlog.Error().Err(err).Int("user_task_id", userTaskID).Msg("Service: Failed to create point transaction within DB transaction")
				err = fmt.Errorf("internal server error: could not record points")
				return err // Rollback
			}
			zlog.Info().Int("user_task_id", userTaskID).Int("points_added", taskDetails.TaskPoint).Int("child_id", taskDetails.ChildID).Msg("Service: Point transaction created within DB transaction")
		} else {
			zlog.Info().Int("user_task_id", userTaskID).Msg("Service: Task approved, but no points awarded (TaskPoint <= 0)")
		}
	}

	// Jika semua berhasil, err akan nil, dan defer akan Commit
	return nil // Sukses
}

// Implementasi metode TaskService lainnya...
