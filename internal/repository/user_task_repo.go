package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type userTaskRepo struct {
	db *pgxpool.Pool
	// Tambahkan dependensi lain jika perlu, misal UserRelationshipRepo untuk cek relasi di VerifyTask
	userRelRepo UserRelationshipRepository
}

// NewUserTaskRepository membuat instance baru dari UserTaskRepository.
// Membutuhkan UserRelationshipRepository untuk beberapa operasi validasi.
func NewUserTaskRepository(db *pgxpool.Pool, userRelRepo UserRelationshipRepository) UserTaskRepository {
	return &userTaskRepo{
		db:          db,
		userRelRepo: userRelRepo, // Simpan dependensi
	}
}

// --- Helper Functions ---

func buildUserTaskQueryWithChildID(baseQuery string, childID int, statusFilter string, page, limit int) (string, []interface{}) {
	args := []interface{}{childID} // Mulai dengan childID
	query := baseQuery
	argCount := 2 // Argumen selanjutnya mulai dari $2

	if statusFilter != "" {
		query += fmt.Sprintf(" AND ut.status = $%d", argCount)
		args = append(args, statusFilter)
		argCount++
	}

	// Selalu tambahkan ORDER BY sebelum LIMIT/OFFSET
	query += " ORDER BY ut.created_at DESC" // Atau assigned_at DESC? Tergantung kebutuhan
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit)

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	args = append(args, offset)

	return query, args
}

func buildUserTaskQueryWithParentID(baseQuery string, parentID int, statusFilter string, page, limit int) (string, []interface{}) {
	args := []interface{}{parentID} // Mulai dengan parentID
	query := baseQuery
	argCount := 2 // Argumen selanjutnya mulai dari $2

	if statusFilter != "" {
		query += fmt.Sprintf(" AND ut.status = $%d", argCount)
		args = append(args, statusFilter)
		argCount++
	}

	// Selalu tambahkan ORDER BY sebelum LIMIT/OFFSET
	query += " ORDER BY ut.created_at DESC" // Atau assigned_at DESC?
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit)

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	args = append(args, offset)

	return query, args
}

// scanUserTaskRow adalah helper untuk scan baris UserTask (termasuk data Task)
func scanUserTaskRow(rows pgx.Rows, ut *models.UserTask) error {
	// Variabel sementara untuk nullable fields
	var verifiedByUserID sql.NullInt32
	var submittedAt, verifiedAt, completedAt sql.NullTime
	// Juga untuk task_description yang nullable
	var taskDescription sql.NullString

	err := rows.Scan(
		// UserTask fields
		&ut.ID, &ut.UserID, &ut.TaskID, &ut.AssignedByUserID, &ut.Status,
		&ut.AssignedAt, &submittedAt, &verifiedByUserID, &verifiedAt, &completedAt,
		&ut.CreatedAt, &ut.UpdatedAt,
		// Task fields
		&ut.Task.ID, &ut.Task.TaskName, &ut.Task.TaskPoint, &taskDescription, &ut.Task.CreatedByUserID,
		&ut.Task.CreatedAt, &ut.Task.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Assign nullable fields jika valid
	if submittedAt.Valid {
		ut.SubmittedAt = &submittedAt.Time
	} else {
		ut.SubmittedAt = nil // Eksplisit set nil jika tidak valid
	}
	if verifiedByUserID.Valid {
		ut.VerifiedByUserID = int(verifiedByUserID.Int32)
	} else {
		// Perhatikan: VerifiedByUserID di model Anda adalah int, bukan *int.
		// Jika 0 dianggap sebagai "belum diverifikasi", ini OK.
		// Jika Anda ingin membedakan 0 vs NULL, ubah tipe di model menjadi *int.
		ut.VerifiedByUserID = 0 // Atau nilai default lain jika bukan 0
	}
	if verifiedAt.Valid {
		ut.VerifiedAt = &verifiedAt.Time
	} else {
		ut.VerifiedAt = nil
	}
	if completedAt.Valid {
		ut.CompletedAt = &completedAt.Time
	} else {
		ut.CompletedAt = nil
	}
	if taskDescription.Valid {
		ut.Task.TaskDescription = taskDescription.String
	} else {
		ut.Task.TaskDescription = "" // Default string kosong jika NULL
	}

	return nil
}

// --- Repository Methods ---

// AssignTask menugaskan sebuah task definition (taskID) kepada user (userID) oleh parent (assignedByID).
func (r *userTaskRepo) AssignTask(ctx context.Context, userID, taskID, assignedByID int) (int, error) {
	query := `INSERT INTO user_tasks (user_id, task_id, assigned_by_user_id, status, assigned_at, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id` // Tambahkan created_at, updated_at
	var userTaskID int
	assignedStatus := models.UserTaskStatusAssigned // Status awal
	assignedAt := time.Now()                        // Waktu penugasan

	err := r.db.QueryRow(ctx, query,
		userID,       // ID Anak
		taskID,       // ID Task Definition
		assignedByID, // ID Parent yang assign
		assignedStatus,
		assignedAt,
	).Scan(&userTaskID)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("user_id", userID).Int("task_id", taskID).Int("assigned_by_id", assignedByID).Msg("Foreign key violation on task assignment")
			return 0, fmt.Errorf("invalid user, task, or assigner ID provided")
		}
		zlog.Error().Err(err).Int("user_id", userID).Int("task_id", taskID).Msg("Error assigning task")
		return 0, fmt.Errorf("error assigning task: %w", err)
	}

	zlog.Info().Int("user_task_id", userTaskID).Int("user_id", userID).Int("task_id", taskID).Int("assigned_by_id", assignedByID).Msg("Task assigned successfully")
	return userTaskID, nil
}

// GetUserTaskByID mengambil detail penugasan tugas spesifik, termasuk data Task dan User (Anak).
func (r *userTaskRepo) GetUserTaskByID(ctx context.Context, id int) (*models.UserTask, error) {
	query := `SELECT
				ut.id, ut.user_id, ut.task_id, ut.assigned_by_user_id, ut.status,
				ut.assigned_at, ut.submitted_at, ut.verified_by_user_id, ut.verified_at, ut.completed_at,
				ut.created_at, ut.updated_at,
				-- Task details
				t.id as taskid, t.task_name, t.task_point, t.task_description, t.created_by_user_id as task_creator_id,
				t.created_at as task_created_at, t.updated_at as task_updated_at,
				-- User (Child) details
				u.id as userid, u.username, u.email, u.first_name, u.last_name, u.role_id,
				u.created_at as user_created_at, u.updated_at as user_updated_at,
				-- User Role details
				r.id as roleid, r.name as rolename
			 FROM user_tasks ut
			 JOIN tasks t ON ut.task_id = t.id
			 JOIN users u ON ut.user_id = u.id
			 JOIN roles r ON u.role_id = r.id
			 WHERE ut.id = $1`

	ut := &models.UserTask{
		Task: &models.Task{},
		User: &models.User{Role: &models.Role{}},
	}

	// Variabel sementara untuk nullable fields
	var verifiedByUserID sql.NullInt32 // Atau tipe yang sesuai
	var submittedAt, verifiedAt, completedAt sql.NullTime
	var taskDescription sql.NullString // Untuk task_description

	err := r.db.QueryRow(ctx, query, id).Scan(
		// UserTask fields
		&ut.ID, &ut.UserID, &ut.TaskID, &ut.AssignedByUserID, &ut.Status,
		&ut.AssignedAt, &submittedAt, &verifiedByUserID, &verifiedAt, &completedAt,
		&ut.CreatedAt, &ut.UpdatedAt, // Pastikan sudah ada di model dan tabel
		// Task fields
		&ut.Task.ID, &ut.Task.TaskName, &ut.Task.TaskPoint, &taskDescription, &ut.Task.CreatedByUserID,
		&ut.Task.CreatedAt, &ut.Task.UpdatedAt,
		// User (Child) fields
		&ut.User.ID, &ut.User.Username, &ut.User.Email, &ut.User.FirstName, &ut.User.LastName, &ut.User.RoleID,
		&ut.User.CreatedAt, &ut.User.UpdatedAt,
		// Role fields
		&ut.User.Role.ID, &ut.User.Role.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_task_id", id).Msg("UserTask not found")
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("user_task_id", id).Msg("Error getting UserTask by ID")
		return nil, fmt.Errorf("error getting user task %d: %w", id, err)
	}

	// Assign nullable fields
	if submittedAt.Valid {
		ut.SubmittedAt = &submittedAt.Time
	} else {
		ut.SubmittedAt = nil
	}
	if verifiedByUserID.Valid {
		ut.VerifiedByUserID = int(verifiedByUserID.Int32)
	} else {
		ut.VerifiedByUserID = 0 // Asumsi 0 = belum diverifikasi
	}
	if verifiedAt.Valid {
		ut.VerifiedAt = &verifiedAt.Time
	} else {
		ut.VerifiedAt = nil
	}
	if completedAt.Valid {
		ut.CompletedAt = &completedAt.Time
	} else {
		ut.CompletedAt = nil
	}
	if taskDescription.Valid {
		ut.Task.TaskDescription = taskDescription.String
	} else {
		ut.Task.TaskDescription = ""
	}

	return ut, nil
}

// GetTasksByChildID mengambil daftar tugas yang ditugaskan ke anak tertentu, dengan filter status dan pagination.
func (r *userTaskRepo) GetTasksByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserTask, int, error) {
	// 1. Hitung total count dengan filter
	countArgs := []interface{}{childID}
	countQuery := `SELECT COUNT(*) FROM user_tasks ut WHERE ut.user_id = $1`
	countArgIdx := 2 // HAPUS
	if statusFilter != "" {
		countQuery += fmt.Sprintf(" AND ut.status = $%d", countArgIdx) // HAPUS
		// countQuery += " AND ut.status = $2"
		countArgs = append(countArgs, statusFilter)
		countArgIdx++ // HAPUS
	}

	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Str("status", statusFilter).Msg("Error counting tasks by child ID")
		return nil, 0, fmt.Errorf("error counting tasks for child %d: %w", childID, err)
	}

	if totalCount == 0 {
		return []models.UserTask{}, 0, nil
	}

	// 2. Buat query utama dengan JOIN dan pagination
	baseQuery := `SELECT
					ut.id, ut.user_id, ut.task_id, ut.assigned_by_user_id, ut.status,
					ut.assigned_at, ut.submitted_at, ut.verified_by_user_id, ut.verified_at, ut.completed_at,
					ut.created_at, ut.updated_at,
					t.id as taskid, t.task_name, t.task_point, t.task_description, t.created_by_user_id as task_creator_id,
					t.created_at as task_created_at, t.updated_at as task_updated_at
				FROM user_tasks ut
				JOIN tasks t ON ut.task_id = t.id
				WHERE ut.user_id = $1` // Argumen pertama selalu childID

	finalQuery, args := buildUserTaskQueryWithChildID(baseQuery, childID, statusFilter, page, limit)

	rows, err := r.db.Query(ctx, finalQuery, args...)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Str("status", statusFilter).Msg("Error querying paginated tasks by child ID")
		return nil, totalCount, fmt.Errorf("error getting tasks for child %d: %w", childID, err)
	}
	defer rows.Close()

	// 3. Scan hasil
	userTasks := []models.UserTask{}
	for rows.Next() {
		var ut models.UserTask
		ut.Task = &models.Task{}              // Inisialisasi Task
		scanErr := scanUserTaskRow(rows, &ut) // Gunakan helper scan
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("child_id", childID).Msg("Error scanning user task row (paginated)")
			return userTasks, totalCount, fmt.Errorf("error scanning user task data: %w", scanErr)
		}
		userTasks = append(userTasks, ut)
	}

	if rowsErr := rows.Err(); rowsErr != nil { // Ganti nama variabel error
		zlog.Error().Err(rowsErr).Int("child_id", childID).Msg("Error iterating paginated user task rows")
		return userTasks, totalCount, fmt.Errorf("error iterating user task data: %w", rowsErr)
	}

	return userTasks, totalCount, nil
}

// GetTasksByParentID mengambil daftar tugas yang ditugaskan oleh parent tertentu (ke semua anaknya),
// dengan filter status dan pagination.
func (r *userTaskRepo) GetTasksByParentID(ctx context.Context, parentID int, statusFilter string, page, limit int) ([]models.UserTask, int, error) {
	// 1. Hitung total count dengan filter
	countArgs := []interface{}{parentID}
	countQuery := `SELECT COUNT(*) FROM user_tasks ut WHERE ut.assigned_by_user_id = $1`
	countArgIdx := 2 // HAPUS
	if statusFilter != "" {
		countQuery += fmt.Sprintf(" AND ut.status = $%d", countArgIdx) // HAPUS
		// countQuery += " AND ut.status = $2"
		countArgs = append(countArgs, statusFilter)
		countArgIdx++ // HAPUS
	}

	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Str("status", statusFilter).Msg("Error counting tasks by parent ID")
		return nil, 0, fmt.Errorf("error counting tasks for parent %d: %w", parentID, err)
	}

	if totalCount == 0 {
		return []models.UserTask{}, 0, nil
	}

	// 2. Buat query utama dengan JOIN dan pagination
	baseQuery := `SELECT
					ut.id, ut.user_id, ut.task_id, ut.assigned_by_user_id, ut.status,
					ut.assigned_at, ut.submitted_at, ut.verified_by_user_id, ut.verified_at, ut.completed_at,
					ut.created_at, ut.updated_at,
					t.id as taskid, t.task_name, t.task_point, t.task_description, t.created_by_user_id as task_creator_id,
					t.created_at as task_created_at, t.updated_at as task_updated_at
				FROM user_tasks ut
				JOIN tasks t ON ut.task_id = t.id
				WHERE ut.assigned_by_user_id = $1` // Argumen pertama selalu parentID

	finalQuery, args := buildUserTaskQueryWithParentID(baseQuery, parentID, statusFilter, page, limit)

	rows, err := r.db.Query(ctx, finalQuery, args...)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Str("status", statusFilter).Msg("Error querying paginated tasks by parent ID")
		return nil, totalCount, fmt.Errorf("error getting tasks for parent %d: %w", parentID, err)
	}
	defer rows.Close()

	// 3. Scan hasil
	userTasks := []models.UserTask{}
	for rows.Next() {
		var ut models.UserTask
		ut.Task = &models.Task{}              // Inisialisasi Task
		scanErr := scanUserTaskRow(rows, &ut) // Gunakan helper scan
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("parent_id", parentID).Msg("Error scanning user task row for parent (paginated)")
			return userTasks, totalCount, fmt.Errorf("error scanning user task data for parent: %w", scanErr)
		}
		userTasks = append(userTasks, ut)
	}

	if rowsErr := rows.Err(); rowsErr != nil { // Ganti nama variabel error
		zlog.Error().Err(rowsErr).Int("parent_id", parentID).Msg("Error iterating paginated user task rows for parent")
		return userTasks, totalCount, fmt.Errorf("error iterating user task data for parent: %w", rowsErr)
	}

	return userTasks, totalCount, nil
}

// SubmitTask mengubah status UserTask menjadi 'submitted' oleh anak.
// Melakukan validasi ownership (task milik childID) dan status saat ini ('assigned').
func (r *userTaskRepo) SubmitTask(ctx context.Context, id int, childID int) error {
	// Tambahkan logika untuk memeriksa status saat ini sebelum update
	getTaskQuery := `SELECT status FROM user_tasks WHERE id = $1 AND user_id = $2`
	var currentStatus models.UserTaskStatus
	err := r.db.QueryRow(ctx, getTaskQuery, id, childID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_task_id", id).Int("child_id", childID).Msg("Submit task failed: Task not found or does not belong to the child")
			return fmt.Errorf("task submission failed: task not found or not assigned to you")
		}
		zlog.Error().Err(err).Int("user_task_id", id).Int("child_id", childID).Msg("Error checking current status before task submission")
		return fmt.Errorf("error submitting task %d: %w", id, err)
	}

	if currentStatus != models.UserTaskStatusAssigned {
		zlog.Warn().Int("user_task_id", id).Int("child_id", childID).Str("current_status", string(currentStatus)).Msg("Submit task failed: Task not in 'assigned' status")
		return fmt.Errorf("task submission failed: task status is already '%s'", currentStatus)
	}

	// Lanjutkan dengan update jika status 'assigned'
	query := `UPDATE user_tasks
              SET status = $1, submitted_at = $2
              WHERE id = $3` // Tidak perlu cek user_id dan status lagi karena sudah dicek
	newStatus := models.UserTaskStatusSubmitted
	submittedAt := time.Now()

	tag, err := r.db.Exec(ctx, query, newStatus, submittedAt, id)
	if err != nil {
		zlog.Error().Err(err).Int("user_task_id", id).Int("child_id", childID).Msg("Error submitting task during update")
		return fmt.Errorf("error submitting task %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		// Ini seharusnya tidak terjadi jika pengecekan status di atas lolos
		zlog.Error().Int("user_task_id", id).Msg("Failed to submit task status despite passing initial checks (concurrency issue?)")
		return fmt.Errorf("failed to submit task, please try again")
	}

	zlog.Info().Int("user_task_id", id).Int("child_id", childID).Msg("Task submitted successfully by child")
	return nil
}

// VerifyTask memverifikasi task yang sudah disubmit ('submitted') oleh parent.
// Mengubah status menjadi 'approved' atau 'rejected'.
// Memvalidasi bahwa parentID adalah parent dari anak yang mengerjakan task.
// Mengembalikan data UserTask yang sudah diupdate (terutama untuk dapatkan poin jika approved).
func (r *userTaskRepo) VerifyTask(ctx context.Context, id int, parentID int, newStatus models.UserTaskStatus) (*models.UserTask, error) {
	// --- LANGKAH 1: Get UserTask untuk validasi ---
	getTaskQuery := `SELECT user_id, status FROM user_tasks WHERE id = $1`
	var childID int
	var currentStatus models.UserTaskStatus
	err := r.db.QueryRow(ctx, getTaskQuery, id).Scan(&childID, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_task_id", id).Msg("Verify task failed: UserTask not found")
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("user_task_id", id).Msg("Error getting UserTask details before verification")
		return nil, fmt.Errorf("error retrieving task details for verification: %w", err)
	}

	// --- LANGKAH 2: Validasi Status Saat Ini ---
	if currentStatus != models.UserTaskStatusSubmitted {
		zlog.Warn().Int("user_task_id", id).Str("current_status", string(currentStatus)).Msg("Verify task failed: Task not in 'submitted' status")
		return nil, fmt.Errorf("cannot verify task: current status is '%s', expected 'submitted'", currentStatus)
	}

	// --- LANGKAH 3: Validasi Relasi Parent-Child ---
	isParent, err := r.userRelRepo.IsParentOf(ctx, parentID, childID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Error checking parent-child relationship during task verification")
		return nil, fmt.Errorf("error verifying parent-child relationship: %w", err)
	}
	if !isParent {
		zlog.Warn().Int("user_task_id", id).Int("parent_id", parentID).Int("child_id", childID).Msg("Verify task failed: Requesting user is not the parent")
		return nil, fmt.Errorf("forbidden: you are not authorized to verify tasks for this child")
	}

	// --- LANGKAH 4: Lakukan Update Status ---
	updateQuery := `UPDATE user_tasks
					SET status = $1, verified_by_user_id = $2, verified_at = $3, completed_at = $4
					WHERE id = $5 AND status = $6` // Tambahkan cek status lagi untuk atomicity
	verifiedAt := time.Now()
	var completedAt *time.Time
	if newStatus == models.UserTaskStatusApproved {
		ca := verifiedAt
		completedAt = &ca
	}

	tag, err := r.db.Exec(ctx, updateQuery, newStatus, parentID, verifiedAt, completedAt, id, models.UserTaskStatusSubmitted) // Cek status 'submitted'
	if err != nil {
		zlog.Error().Err(err).Int("user_task_id", id).Int("parent_id", parentID).Str("new_status", string(newStatus)).Msg("Error updating user_task status during verification")
		return nil, fmt.Errorf("error updating task status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Bisa karena status sudah berubah oleh proses lain (concurrency) atau ID tidak valid (tapi sudah dicek)
		zlog.Warn().Int("user_task_id", id).Msg("Failed to update task status, likely due to status change or concurrency.")
		// Query ulang untuk mendapatkan status terbaru
		var latestStatus models.UserTaskStatus
		errStatus := r.db.QueryRow(ctx, `SELECT status FROM user_tasks WHERE id = $1`, id).Scan(&latestStatus)
		if errStatus != nil {
			return nil, fmt.Errorf("failed to update task status and could not retrieve current status")
		}
		return nil, fmt.Errorf("failed to update task status: current status is already '%s'", latestStatus) // Beri info status terbaru
	}

	// --- LANGKAH 5: Ambil Data UserTask yang Sudah Diupdate (Termasuk Task) ---
	updatedUserTask, err := r.GetUserTaskByID(ctx, id)
	if err != nil {
		zlog.Error().Err(err).Int("user_task_id", id).Msg("Failed to retrieve updated UserTask data after verification")
		return nil, fmt.Errorf("task status updated, but failed to retrieve updated details: %w", err)
	}

	zlog.Info().Int("user_task_id", id).Int("parent_id", parentID).Str("new_status", string(newStatus)).Msg("Task verified successfully by parent")
	return updatedUserTask, nil
}

// UpdateUserTaskStatus (Implementasi Internal atau untuk Admin)
// CATATAN: Jika ini hanya internal, hapus dari interface UserTaskRepository.
// Jika untuk Admin, perlu validasi role admin di handler yang memanggilnya.
func (r *userTaskRepo) UpdateUserTaskStatus(ctx context.Context, id int, newStatus models.UserTaskStatus, verifierID *int) error {
	query := `UPDATE user_tasks SET status = $1, verified_by_user_id = $2, verified_at = $3, completed_at = $4 WHERE id = $5`
	now := time.Now()
	var completedAt *time.Time
	var verifiedAt *time.Time

	// Tentukan logika timestamp berdasarkan status baru dan verifierID
	var sqlVerifierID sql.NullInt32
	if verifierID != nil {
		sqlVerifierID = sql.NullInt32{Int32: int32(*verifierID), Valid: true}
		verifiedAt = &now
		if newStatus == models.UserTaskStatusApproved {
			completedAt = &now
		}
	} else {
		// Jika verifierID nil, mungkin status diubah oleh sistem atau anak?
		// Set verified_by_user_id ke NULL.
		sqlVerifierID = sql.NullInt32{Valid: false}
		// Jangan set verifiedAt atau completedAt jika tidak ada verifier? Tergantung logika.
	}

	tag, err := r.db.Exec(ctx, query, newStatus, sqlVerifierID, verifiedAt, completedAt, id)
	if err != nil {
		return fmt.Errorf("error updating status for user_task %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Metode untuk Service Layer (Menggunakan Transaksi) ---

// GetTaskDetailsForVerificationTx mengambil detail minimal untuk verifikasi dalam transaksi.
func (r *userTaskRepo) GetTaskDetailsForVerificationTx(ctx context.Context, tx pgx.Tx, userTaskID int) (*TaskVerificationDetails, error) {
	query := `SELECT ut.user_id, ut.status, t.task_point
              FROM user_tasks ut
              JOIN tasks t ON ut.task_id = t.id
              WHERE ut.id = $1 FOR UPDATE` // Tambahkan FOR UPDATE untuk locking
	details := &TaskVerificationDetails{}
	err := tx.QueryRow(ctx, query, userTaskID).Scan(&details.ChildID, &details.CurrentStatus, &details.TaskPoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("user_task_id", userTaskID).Msg("RepoTx: Error getting task details for verification")
		return nil, fmt.Errorf("repoTx error getting task details for verification: %w", err)
	}
	return details, nil // Jangan bungkus ErrNoRows di sini
}

// UpdateStatusTx mengupdate status dalam transaksi.
func (r *userTaskRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserTaskStatus, verifierID int) error {
	query := `UPDATE user_tasks SET status = $1, verified_by_user_id = $2, verified_at = $3, completed_at = $4
			  WHERE id = $5 AND status = $6` // Pastikan status masih submitted
	now := time.Now()
	var completedAt *time.Time
	if newStatus == models.UserTaskStatusApproved {
		ca := now
		completedAt = &ca
	}
	tag, err := tx.Exec(ctx, query, newStatus, verifierID, now, completedAt, id, models.UserTaskStatusSubmitted) // Cek status submitted
	if err != nil {
		zlog.Error().Err(err).Int("user_task_id", id).Msg("RepoTx: Error updating task status")
		return fmt.Errorf("repoTx error updating status for user_task %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		zlog.Warn().Int("user_task_id", id).Msg("RepoTx: Failed to update task status, likely due to status change or concurrency.")
		// Query ulang status untuk error message yang lebih baik?
		var latestStatus models.UserTaskStatus
		errStatus := tx.QueryRow(ctx, `SELECT status FROM user_tasks WHERE id = $1`, id).Scan(&latestStatus)
		if errStatus != nil {
			// Jika query status juga gagal, kembalikan error NoRows/Concurrency umum
			return fmt.Errorf("task status update failed, possibly due to prior change") // Error NoRows bisa ambigu
		}
		// Jika status ditemukan, berikan info status terbaru
		return fmt.Errorf("failed to update task status: current status is already '%s'", latestStatus)
	}
	return nil
}
