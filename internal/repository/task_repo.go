package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type taskRepo struct {
	db *pgxpool.Pool
}

// NewTaskRepository membuat instance baru dari TaskRepository.
func NewTaskRepository(db *pgxpool.Pool) TaskRepository {
	return &taskRepo{db: db}
}

// CreateTask membuat definisi task baru di database.
func (r *taskRepo) CreateTask(ctx context.Context, task *models.Task) (int, error) {
	query := `INSERT INTO tasks (task_name, task_point, task_description, created_by_user_id)
              VALUES ($1, $2, $3, $4) RETURNING id`
	var taskID int
	err := r.db.QueryRow(ctx, query,
		task.TaskName,
		task.TaskPoint,
		task.TaskDescription,
		task.CreatedByUserID, // ID Parent yang membuat
	).Scan(&taskID)

	if err != nil {
		// Handle FK violation jika created_by_user_id tidak valid
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("creator_id", task.CreatedByUserID).Msg("Foreign key violation on task creation (creator not found?)")
			return 0, fmt.Errorf("creator user with ID %d does not exist", task.CreatedByUserID)
		}
		// Error umum lainnya
		zlog.Error().Err(err).Str("task_name", task.TaskName).Int("creator_id", task.CreatedByUserID).Msg("Error creating task definition")
		return 0, fmt.Errorf("error creating task definition: %w", err)
	}

	zlog.Info().Int("task_id", taskID).Str("task_name", task.TaskName).Int("creator_id", task.CreatedByUserID).Msg("Task definition created successfully")
	return taskID, nil
}

// GetTaskByID mengambil detail definisi task berdasarkan ID-nya.
// Memerlukan parentID untuk memvalidasi bahwa task tersebut dibuat oleh parent yang meminta.
func (r *taskRepo) GetTaskByID(ctx context.Context, id int, parentID int) (*models.Task, error) {
	query := `SELECT id, task_name, task_point, task_description, created_by_user_id, created_at, updated_at
              FROM tasks
              WHERE id = $1 AND created_by_user_id = $2` // Validasi kepemilikan di query
	task := &models.Task{}
	err := r.db.QueryRow(ctx, query, id, parentID).Scan(
		&task.ID,
		&task.TaskName,
		&task.TaskPoint,
		&task.TaskDescription,
		&task.CreatedByUserID,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("task_id", id).Int("requesting_parent_id", parentID).Msg("Task definition not found or access denied")
			// Kembalikan ErrNoRows agar handler tahu tidak ditemukan / tidak berhak
			return nil, pgx.ErrNoRows
		}
		// Error umum
		zlog.Error().Err(err).Int("task_id", id).Int("requesting_parent_id", parentID).Msg("Error getting task definition by ID")
		return nil, fmt.Errorf("error getting task definition %d: %w", id, err)
	}

	return task, nil
}

// GetTasksByCreatorID mengambil daftar definisi task (paginated) yang dibuat oleh parent tertentu.
func (r *taskRepo) GetTasksByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Task, int, error) {
	// 1. Hitung Total Task untuk creator ini
	countQuery := `SELECT COUNT(*) FROM tasks WHERE created_by_user_id = $1`
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, creatorID).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("creator_id", creatorID).Msg("Error counting tasks by creator ID")
		return nil, 0, fmt.Errorf("error counting tasks for creator %d: %w", creatorID, err)
	}

	if totalCount == 0 {
		return []models.Task{}, 0, nil // Kembalikan slice kosong jika tidak ada task
	}

	// 2. Hitung Offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 3. Query Task dengan Pagination
	query := `SELECT id, task_name, task_point, task_description, created_by_user_id, created_at, updated_at
			FROM tasks
			WHERE created_by_user_id = $1
			ORDER BY created_at DESC -- Urutkan dari terbaru
			LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, creatorID, limit, offset)
	if err != nil {
		zlog.Error().Err(err).Int("creator_id", creatorID).Msg("Error querying paginated tasks by creator ID")
		return nil, totalCount, fmt.Errorf("error getting tasks for creator %d: %w", creatorID, err)
	}
	defer rows.Close()

	// 4. Scan Hasil
	tasks := []models.Task{}
	for rows.Next() {
		var task models.Task
		scanErr := rows.Scan(
			&task.ID,
			&task.TaskName,
			&task.TaskPoint,
			&task.TaskDescription,
			&task.CreatedByUserID,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("creator_id", creatorID).Msg("Error scanning task row (paginated)")
			return tasks, totalCount, fmt.Errorf("error scanning task data: %w", scanErr)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		zlog.Error().Err(err).Int("creator_id", creatorID).Msg("Error iterating paginated task rows")
		return tasks, totalCount, fmt.Errorf("error iterating task data: %w", err)
	}

	return tasks, totalCount, nil
}

// UpdateTask memperbarui detail definisi task.
// Memerlukan parentID untuk memastikan hanya pembuat asli yang bisa mengedit.
func (r *taskRepo) UpdateTask(ctx context.Context, task *models.Task, parentID int) error {
	query := `UPDATE tasks
              SET task_name = $1, task_point = $2, task_description = $3
              WHERE id = $4 AND created_by_user_id = $5` // Validasi ID dan kepemilikan

	tag, err := r.db.Exec(ctx, query,
		task.TaskName,
		task.TaskPoint,
		task.TaskDescription,
		task.ID,  // ID task yang diupdate
		parentID, // ID parent yang melakukan request
	)

	if err != nil {
		// Error umum
		zlog.Error().Err(err).Int("task_id", task.ID).Int("requesting_parent_id", parentID).Msg("Error updating task definition")
		return fmt.Errorf("error updating task definition %d: %w", task.ID, err)
	}

	if tag.RowsAffected() == 0 {
		// Ini bisa berarti task tidak ditemukan ATAU parentID tidak cocok dengan created_by_user_id
		zlog.Warn().Int("task_id", task.ID).Int("requesting_parent_id", parentID).Msg("Task definition not found or update access denied")
		return pgx.ErrNoRows // Kembalikan ErrNoRows untuk ditangani handler
	}

	zlog.Info().Int("task_id", task.ID).Int("requesting_parent_id", parentID).Msg("Task definition updated successfully")
	return nil
}

// DeleteTask menghapus definisi task.
// Memerlukan parentID untuk memastikan hanya pembuat asli yang bisa menghapus.
// Penting: Ini hanya menghapus definisi. Penugasan di user_tasks yang merujuk ke sini
// akan gagal jika FK constraint-nya RESTRICT (seperti yang kita set).
func (r *taskRepo) DeleteTask(ctx context.Context, id int, parentID int) error {
	// Cek dulu apakah task ini sedang digunakan di user_tasks (jika constraint RESTRICT)
	// Ini opsional tapi bisa memberikan error yang lebih baik ke user daripada FK error
	checkUsageQuery := `SELECT COUNT(*) FROM user_tasks WHERE task_id = $1`
	var usageCount int
	errUsage := r.db.QueryRow(ctx, checkUsageQuery, id).Scan(&usageCount)
	if errUsage != nil {
		zlog.Error().Err(errUsage).Int("task_id", id).Msg("Error checking task usage before deletion")
		return fmt.Errorf("error checking task usage for task %d: %w", id, errUsage)
	}
	if usageCount > 0 {
		zlog.Warn().Int("task_id", id).Int("usage_count", usageCount).Msg("Attempted to delete task definition that is still in use")
		return fmt.Errorf("cannot delete task definition: task is currently assigned to %d assignment(s)", usageCount)
	}

	// Lanjutkan penghapusan dengan validasi kepemilikan
	query := `DELETE FROM tasks WHERE id = $1 AND created_by_user_id = $2`
	tag, err := r.db.Exec(ctx, query, id, parentID)

	if err != nil {
		// Error umum (termasuk FK violation jika user_tasks merujuk ke sini dan kita tidak cek di atas)
		// Jika FK error (kode 23503), berikan pesan spesifik.
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("task_id", id).Msg("Attempted to delete task definition that is still referenced by user_tasks")
			return fmt.Errorf("cannot delete task definition: it is currently assigned or has been completed by users")
		}
		zlog.Error().Err(err).Int("task_id", id).Int("requesting_parent_id", parentID).Msg("Error deleting task definition")
		return fmt.Errorf("error deleting task definition %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		// Task tidak ditemukan ATAU bukan milik parent ini
		zlog.Warn().Int("task_id", id).Int("requesting_parent_id", parentID).Msg("Task definition not found or delete access denied")
		return pgx.ErrNoRows
	}

	zlog.Info().Int("task_id", id).Int("requesting_parent_id", parentID).Msg("Task definition deleted successfully")
	return nil
}
