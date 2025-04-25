// internal/repository/user_reward_repo.go
package repository

import (
	"context"
	"database/sql" // Untuk nullable fields
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type userRewardRepo struct {
	db *pgxpool.Pool
	// Mungkin perlu UserRelationshipRepo jika validasi parent dilakukan di sini
	userRelRepo UserRelationshipRepository
}

// NewUserRewardRepository membuat instance baru UserRewardRepository.
func NewUserRewardRepository(db *pgxpool.Pool, userRelRepo UserRelationshipRepository) UserRewardRepository {
	return &userRewardRepo{
		db:          db,
		userRelRepo: userRelRepo,
	}
}

// CreateClaim membuat record klaim reward baru dengan status awal 'pending'.
// pointsDeducted diambil dari service layer setelah validasi poin.
func (r *userRewardRepo) CreateClaim(ctx context.Context, userID, rewardID, pointsDeducted int) (int, error) {
	query := `INSERT INTO user_rewards (user_id, reward_id, points_deducted, status, claimed_at)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var claimID int
	initialStatus := models.UserRewardStatusPending
	claimedAt := time.Now()

	err := r.db.QueryRow(ctx, query,
		userID,         // ID Anak yang claim
		rewardID,       // ID Reward yang di-claim
		pointsDeducted, // Poin yang akan dikurangi (sudah divalidasi service)
		initialStatus,  // Status awal pending
		claimedAt,      // Waktu klaim
	).Scan(&claimID)

	if err != nil {
		// Handle FK violation (user_id atau reward_id tidak valid)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("user_id", userID).Int("reward_id", rewardID).Msg("Foreign key violation on reward claim creation")
			return 0, fmt.Errorf("invalid user or reward ID provided")
		}
		// Error umum
		zlog.Error().Err(err).Int("user_id", userID).Int("reward_id", rewardID).Msg("Error creating reward claim")
		return 0, fmt.Errorf("error creating reward claim: %w", err)
	}

	zlog.Info().Int("claim_id", claimID).Int("user_id", userID).Int("reward_id", rewardID).Msg("Reward claim created successfully")
	return claimID, nil
}

// GetUserRewardByID mengambil detail klaim reward, termasuk data Reward dan User (Anak).
func (r *userRewardRepo) GetUserRewardByID(ctx context.Context, id int) (*models.UserReward, error) {
	query := `SELECT
                ur.id, ur.user_id, ur.reward_id, ur.points_deducted, ur.claimed_at, ur.status,
                ur.reviewed_by_user_id, ur.reviewed_at, ur.created_at, ur.updated_at,
                -- Reward details
                rw.id as rewardid, rw.reward_name, rw.reward_point, rw.reward_description,
                rw.created_by_user_id as reward_creator_id, rw.created_at as reward_created_at, rw.updated_at as reward_updated_at,
                -- User (Child) details
                u.id as userid, u.username, u.email, u.first_name, u.last_name, u.role_id,
                u.created_at as user_created_at, u.updated_at as user_updated_at,
                -- User Role details
				r.id as roleid, r.name as rolename
			FROM user_rewards ur
			JOIN rewards rw ON ur.reward_id = rw.id
			JOIN users u ON ur.user_id = u.id
			JOIN roles r ON u.role_id = r.id
			WHERE ur.id = $1`

	ur := &models.UserReward{
		Reward: &models.Reward{},
		User:   &models.User{Role: &models.Role{}},
	}

	// Nullable fields
	var reviewedByUserID sql.NullInt32
	var reviewedAt sql.NullTime
	var rewardDescription sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		// UserReward fields
		&ur.ID, &ur.UserID, &ur.RewardID, &ur.PointsDeducted, &ur.ClaimedAt, &ur.Status,
		&reviewedByUserID, &reviewedAt, &ur.CreatedAt, &ur.UpdatedAt,
		// Reward fields
		&ur.Reward.ID, &ur.Reward.RewardName, &ur.Reward.RewardPoint, &rewardDescription,
		&ur.Reward.CreatedByUserID, &ur.Reward.CreatedAt, &ur.Reward.UpdatedAt,
		// User fields
		&ur.User.ID, &ur.User.Username, &ur.User.Email, &ur.User.FirstName, &ur.User.LastName, &ur.User.RoleID,
		&ur.User.CreatedAt, &ur.User.UpdatedAt,
		// Role fields
		&ur.User.Role.ID, &ur.User.Role.Name,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_reward_id", id).Msg("UserReward claim not found")
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("user_reward_id", id).Msg("Error getting UserReward by ID")
		return nil, fmt.Errorf("error getting user reward claim %d: %w", id, err)
	}

	// Assign nullable fields
	if reviewedByUserID.Valid {
		ur.ReviewedByUserID = int(reviewedByUserID.Int32)
	} else {
		ur.ReviewedByUserID = 0 // Asumsi 0 = belum direview
	}
	if reviewedAt.Valid {
		ur.ReviewedAt = &reviewedAt.Time
	} else {
		ur.ReviewedAt = nil
	}
	if rewardDescription.Valid {
		ur.Reward.RewardDescription = rewardDescription.String
	} else {
		ur.Reward.RewardDescription = ""
	}

	return ur, nil
}

// Helper untuk scan baris UserReward (termasuk data Reward)
func scanUserRewardRow(rows pgx.Rows, ur *models.UserReward) error {
	var reviewedByUserID sql.NullInt32
	var reviewedAt sql.NullTime
	var rewardDescription sql.NullString

	err := rows.Scan(
		// UserReward fields
		&ur.ID, &ur.UserID, &ur.RewardID, &ur.PointsDeducted, &ur.ClaimedAt, &ur.Status,
		&reviewedByUserID, &reviewedAt, &ur.CreatedAt, &ur.UpdatedAt,
		// Reward fields
		&ur.Reward.ID, &ur.Reward.RewardName, &ur.Reward.RewardPoint, &rewardDescription,
		&ur.Reward.CreatedByUserID, &ur.Reward.CreatedAt, &ur.Reward.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Assign nullable fields
	if reviewedByUserID.Valid {
		ur.ReviewedByUserID = int(reviewedByUserID.Int32)
	} else {
		ur.ReviewedByUserID = 0
	}
	if reviewedAt.Valid {
		ur.ReviewedAt = &reviewedAt.Time
	} else {
		ur.ReviewedAt = nil
	}
	if rewardDescription.Valid {
		ur.Reward.RewardDescription = rewardDescription.String
	} else {
		ur.Reward.RewardDescription = ""
	}

	return nil
}

// GetClaimsByChildID mengambil daftar klaim reward oleh anak tertentu, dengan filter status dan pagination.
func (r *userRewardRepo) GetClaimsByChildID(ctx context.Context, childID int, statusFilter string, page, limit int) ([]models.UserReward, int, error) {
	// 1. Hitung total
	countArgs := []interface{}{childID}
	countQuery := `SELECT COUNT(*) FROM user_rewards ur WHERE ur.user_id = $1`
	countArgIdx := 2 // HAPUS
	if statusFilter != "" {
		countQuery += fmt.Sprintf(" AND ur.status = $%d", countArgIdx) // HAPUS
		// countQuery += " AND ur.status = $2"
		countArgs = append(countArgs, statusFilter)
		countArgIdx++ // HAPUS
	}

	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Str("status", statusFilter).Msg("Error counting claims by child ID")
		return nil, 0, fmt.Errorf("error counting claims for child %d: %w", childID, err)
	}

	if totalCount == 0 {
		return []models.UserReward{}, 0, nil
	}

	// 2. Query data dengan JOIN ke rewards dan pagination
	queryArgs := []interface{}{childID}
	query := `SELECT
                ur.id, ur.user_id, ur.reward_id, ur.points_deducted, ur.claimed_at, ur.status,
                ur.reviewed_by_user_id, ur.reviewed_at, ur.created_at, ur.updated_at,
                rw.id as rewardid, rw.reward_name, rw.reward_point, rw.reward_description,
                rw.created_by_user_id as reward_creator_id, rw.created_at as reward_created_at, rw.updated_at as reward_updated_at
             FROM user_rewards ur
             JOIN rewards rw ON ur.reward_id = rw.id
             WHERE ur.user_id = $1`
	queryArgIdx := 2
	if statusFilter != "" {
		query += fmt.Sprintf(" AND ur.status = $%d", queryArgIdx)
		queryArgs = append(queryArgs, statusFilter)
		queryArgIdx++
	}

	query += fmt.Sprintf(" ORDER BY ur.claimed_at DESC LIMIT $%d OFFSET $%d", queryArgIdx, queryArgIdx+1)
	queryArgs = append(queryArgs, limit)

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	queryArgs = append(queryArgs, offset)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Str("status", statusFilter).Msg("Error querying paginated claims by child ID")
		return nil, totalCount, fmt.Errorf("error getting claims for child %d: %w", childID, err)
	}
	defer rows.Close()

	// 3. Scan hasil
	claims := []models.UserReward{}
	for rows.Next() {
		var ur models.UserReward
		ur.Reward = &models.Reward{}            // Inisialisasi Reward
		scanErr := scanUserRewardRow(rows, &ur) // Gunakan helper scan
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("child_id", childID).Msg("Error scanning user reward claim row (paginated)")
			return claims, totalCount, fmt.Errorf("error scanning claim data: %w", scanErr)
		}
		claims = append(claims, ur)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		zlog.Error().Err(rowsErr).Int("child_id", childID).Msg("Error iterating paginated claim rows")
		return claims, totalCount, fmt.Errorf("error iterating claim data: %w", rowsErr)
	}

	return claims, totalCount, nil
}

// GetPendingClaimsByParentID mengambil daftar klaim 'pending' dari semua anak yang terhubung ke parent tertentu.
func (r *userRewardRepo) GetPendingClaimsByParentID(ctx context.Context, parentID int, page, limit int) ([]models.UserReward, int, error) {
	// 1. Query untuk mendapatkan ID anak dari parent
	childIDsQuery := `SELECT child_id FROM user_relationship WHERE parent_id = $1`
	rowsChildren, err := r.db.Query(ctx, childIDsQuery, parentID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Error getting child IDs for parent (pending claims)")
		return nil, 0, fmt.Errorf("error finding children for parent %d: %w", parentID, err)
	}

	childIDs := []int{}
	for rowsChildren.Next() {
		var cID int
		if err := rowsChildren.Scan(&cID); err != nil {
			rowsChildren.Close()
			zlog.Warn().Err(err).Int("parent_id", parentID).Msg("Error scanning child ID for parent (pending claims)")
			return nil, 0, fmt.Errorf("error processing child data for parent: %w", err)
		}
		childIDs = append(childIDs, cID)
	}
	rowsChildren.Close()

	if len(childIDs) == 0 {
		zlog.Info().Int("parent_id", parentID).Msg("Parent has no associated children, no pending claims to fetch.")
		return []models.UserReward{}, 0, nil
	}

	// 2. Hitung total klaim 'pending' dari anak-anak ini
	countQuery := `SELECT COUNT(*) FROM user_rewards ur
                   WHERE ur.user_id = ANY($1::int[]) AND ur.status = $2`
	var totalCount int
	err = r.db.QueryRow(ctx, countQuery, childIDs, models.UserRewardStatusPending).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Error counting pending claims for parent's children")
		return nil, 0, fmt.Errorf("error counting pending claims: %w", err)
	}

	if totalCount == 0 {
		return []models.UserReward{}, 0, nil
	}

	// 3. Hitung Offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 4. Query klaim 'pending' dari anak-anak ini dengan JOIN dan pagination
	query := `SELECT
                ur.id, ur.user_id, ur.reward_id, ur.points_deducted, ur.claimed_at, ur.status,
                ur.reviewed_by_user_id, ur.reviewed_at, ur.created_at, ur.updated_at,
                rw.id as rewardid, rw.reward_name, rw.reward_point, rw.reward_description,
                rw.created_by_user_id as reward_creator_id, rw.created_at as reward_created_at, rw.updated_at as reward_updated_at
             FROM user_rewards ur
             JOIN rewards rw ON ur.reward_id = rw.id
             WHERE ur.user_id = ANY($1::int[]) AND ur.status = $2
             ORDER BY ur.claimed_at ASC -- Tampilkan yang paling lama pending dulu
             LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, childIDs, models.UserRewardStatusPending, limit, offset)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Error querying pending claims for parent's children")
		return nil, totalCount, fmt.Errorf("error getting pending claims: %w", err)
	}
	defer rows.Close()

	// 5. Scan hasil
	claims := []models.UserReward{}
	for rows.Next() {
		var ur models.UserReward
		ur.Reward = &models.Reward{}
		scanErr := scanUserRewardRow(rows, &ur)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("parent_id", parentID).Msg("Error scanning pending claim row")
			return claims, totalCount, fmt.Errorf("error scanning pending claim data: %w", scanErr)
		}
		claims = append(claims, ur)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		zlog.Error().Err(rowsErr).Int("parent_id", parentID).Msg("Error iterating pending claim rows")
		return claims, totalCount, fmt.Errorf("error iterating pending claim data: %w", rowsErr)
	}

	return claims, totalCount, nil
}

// UpdateClaimStatus mengubah status klaim reward oleh parent.
// Validasi: Status harus 'pending', dan reviewerID harus parent dari anak yang claim.
// Pengurangan poin dilakukan oleh Service Layer setelah ini berhasil jika status 'approved'.
func (r *userRewardRepo) UpdateClaimStatus(ctx context.Context, id int, newStatus models.UserRewardStatus, reviewerID int) error {
	// Langkah 1: Dapatkan user_id (anak) dan status saat ini dari klaim
	getClaimQuery := `SELECT user_id, status FROM user_rewards WHERE id = $1`
	var childID int
	var currentStatus models.UserRewardStatus
	err := r.db.QueryRow(ctx, getClaimQuery, id).Scan(&childID, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_reward_id", id).Msg("Update claim status failed: Claim not found")
			return pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("user_reward_id", id).Msg("Error getting claim details before status update")
		return fmt.Errorf("error retrieving claim details for update: %w", err)
	}

	// Langkah 2: Validasi status saat ini
	if currentStatus != models.UserRewardStatusPending {
		zlog.Warn().Int("user_reward_id", id).Str("current_status", string(currentStatus)).Msg("Update claim status failed: Claim not in 'pending' status")
		return fmt.Errorf("cannot update claim status: current status is '%s', expected 'pending'", currentStatus)
	}

	// Langkah 3: Validasi relasi parent-child (perlu UserRelationshipRepo)
	// isParent, err := r.userRelRepo.IsParentOf(ctx, reviewerID, childID)
	// if err != nil {
	//     zlog.Error().Err(err).Int("reviewer_id", reviewerID).Int("child_id", childID).Msg("Error checking parent-child relationship during claim review")
	//     return fmt.Errorf("error verifying parent-child relationship: %w", err)
	// }
	// if !isParent {
	//     zlog.Warn().Int("user_reward_id", id).Int("reviewer_id", reviewerID).Int("child_id", childID).Msg("Update claim status failed: Requesting user is not the parent")
	//     return fmt.Errorf("forbidden: you are not authorized to review claims for this child")
	// }
	// ---- CATATAN: Validasi relasi parent sebaiknya dilakukan di Service Layer atau Handler ----
	// ---- Repository ini hanya fokus update jika ID & status sesuai ----
	// TODO: Add parent validation in Service Layer
	// Langkah 4: Lakukan Update Status
	query := `UPDATE user_rewards
              SET status = $1, reviewed_by_user_id = $2, reviewed_at = $3
              WHERE id = $4 AND status = $5` // Cek status 'pending' lagi untuk atomicity
	reviewedAt := time.Now()

	tag, err := r.db.Exec(ctx, query, newStatus, reviewerID, reviewedAt, id, models.UserRewardStatusPending)
	if err != nil {
		zlog.Error().Err(err).Int("user_reward_id", id).Int("reviewer_id", reviewerID).Str("new_status", string(newStatus)).Msg("Error updating user_reward claim status")
		return fmt.Errorf("error updating claim status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Bisa karena status sudah berubah oleh proses lain (concurrency)
		zlog.Warn().Int("user_reward_id", id).Msg("Failed to update claim status, likely due to status change or concurrency.")
		// Query ulang status untuk error message yang lebih baik
		var latestStatus models.UserRewardStatus
		errStatus := r.db.QueryRow(ctx, `SELECT status FROM user_rewards WHERE id = $1`, id).Scan(&latestStatus)
		if errStatus != nil {
			return fmt.Errorf("failed to update claim status and could not retrieve current status")
		}
		return fmt.Errorf("failed to update claim status: current status is already '%s'", latestStatus)
	}

	zlog.Info().Int("user_reward_id", id).Int("reviewer_id", reviewerID).Str("new_status", string(newStatus)).Msg("User reward claim status updated successfully")
	return nil
}

// --- Metode Tx untuk Service Layer ---

// CreateClaimTx membuat klaim dalam transaksi.
// Pengecekan poin cukup dilakukan oleh service sebelum memanggil ini.
func (r *userRewardRepo) CreateClaimTx(ctx context.Context, tx pgx.Tx, userID, rewardID, pointsDeducted int) (int, error) {
	query := `INSERT INTO user_rewards (user_id, reward_id, points_deducted, status, claimed_at)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var claimID int
	initialStatus := models.UserRewardStatusPending
	claimedAt := time.Now()

	err := tx.QueryRow(ctx, query, userID, rewardID, pointsDeducted, initialStatus, claimedAt).Scan(&claimID)
	if err != nil {
		// Handle FK violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("user_id", userID).Int("reward_id", rewardID).Msg("RepoTx: Foreign key violation on reward claim creation")
			return 0, fmt.Errorf("invalid user or reward ID provided")
		}
		zlog.Error().Err(err).Int("user_id", userID).Int("reward_id", rewardID).Msg("RepoTx: Error creating reward claim")
		return 0, fmt.Errorf("repoTx error creating reward claim: %w", err)
	}
	return claimID, nil
}

// UpdateClaimStatusTx (Opsional, jika service perlu update status dalam transaksi yang lebih besar)
func (r *userRewardRepo) UpdateClaimStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserRewardStatus, reviewerID int) error {
	// Mirip dengan UpdateClaimStatus non-Tx, tapi menggunakan tx.Exec
	// Validasi status 'pending' dan relasi parent sebaiknya tetap di service.
	query := `UPDATE user_rewards
              SET status = $1, reviewed_by_user_id = $2, reviewed_at = $3
              WHERE id = $4 AND status = $5`
	reviewedAt := time.Now()
	zlog.Debug().Str("query", query).Int("id", id).Str("newStatus", string(newStatus)).Int("reviewerID", reviewerID).Time("reviewedAt", reviewedAt).Str("expectedOldStatus", string(models.UserRewardStatusPending)).Msg("RepoTx: Executing UpdateClaimStatusTx") // Log detail
	tag, err := tx.Exec(ctx, query, newStatus, reviewerID, reviewedAt, id, models.UserRewardStatusPending)
	if err != nil {
		// ... (handle error) ...
		return fmt.Errorf("repoTx error updating claim status: %w", err)
	}
	zlog.Debug().Int64("rowsAffected", tag.RowsAffected()).Int("claimID", id).Msg("RepoTx: Result of UpdateClaimStatusTx execution") // Log hasil
	if tag.RowsAffected() == 0 {
		// ... (handle concurrency/status change) ...
		// Query ulang status jika perlu
		return fmt.Errorf("repoTx failed to update claim status: status may have changed") // atau pgx.ErrNoRows?
	}
	return nil
}

// GetClaimDetailsForReviewTx mengambil detail minimal klaim untuk proses review dalam transaksi.
func (r *userRewardRepo) GetClaimDetailsForReviewTx(ctx context.Context, tx pgx.Tx, claimID int) (*ClaimReviewDetails, error) {
	query := `SELECT ur.user_id, ur.status, ur.points_deducted, rw.created_by_user_id -- Tambahkan creator reward
				FROM user_rewards ur
				JOIN rewards rw ON ur.reward_id = rw.id -- Perlu JOIN ke rewards
				WHERE ur.id = $1 FOR UPDATE`
	details := &ClaimReviewDetails{}
	err := tx.QueryRow(ctx, query, claimID).Scan(
		&details.ChildID,
		&details.CurrentStatus,
		&details.PointsDeducted,
		&details.RewardCreatorID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("claim_id", claimID).Msg("RepoTx: Error getting claim details for review")
		return nil, fmt.Errorf("repoTx error getting claim details for review %d: %w", claimID, err)
	}
	return details, nil
}
