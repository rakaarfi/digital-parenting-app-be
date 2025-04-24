// internal/repository/reward_repo.go
package repository

import (
	"context"
	"database/sql" // Untuk sql.NullString
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type rewardRepo struct {
	db *pgxpool.Pool
}

// NewRewardRepository membuat instance baru dari RewardRepository.
func NewRewardRepository(db *pgxpool.Pool) RewardRepository {
	return &rewardRepo{db: db}
}

// CreateReward membuat definisi reward baru.
func (r *rewardRepo) CreateReward(ctx context.Context, reward *models.Reward) (int, error) {
	query := `INSERT INTO rewards (reward_name, reward_point, reward_description, created_by_user_id)
              VALUES ($1, $2, $3, $4) RETURNING id`
	var rewardID int
	err := r.db.QueryRow(ctx, query,
		reward.RewardName,
		reward.RewardPoint,
		reward.RewardDescription,
		reward.CreatedByUserID, // ID Parent pembuat
	).Scan(&rewardID)

	if err != nil {
		// Handle FK violation jika created_by_user_id tidak valid
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("creator_id", reward.CreatedByUserID).Msg("Foreign key violation on reward creation (creator not found?)")
			return 0, fmt.Errorf("creator user with ID %d does not exist", reward.CreatedByUserID)
		}
		// Error umum
		zlog.Error().Err(err).Str("reward_name", reward.RewardName).Int("creator_id", reward.CreatedByUserID).Msg("Error creating reward definition")
		return 0, fmt.Errorf("error creating reward definition: %w", err)
	}

	zlog.Info().Int("reward_id", rewardID).Str("reward_name", reward.RewardName).Int("creator_id", reward.CreatedByUserID).Msg("Reward definition created successfully")
	return rewardID, nil
}

// GetRewardByID (Family Visibility Implementation)
// GetRewardByID hanya mengambil berdasarkan ID, tanpa validasi ownership/family.
// Validasi akses dilakukan di Handler/Service.
func (r *rewardRepo) GetRewardByID(ctx context.Context, id int) (*models.Reward, error) {
	query := `
        SELECT
            id, reward_name, reward_point, reward_description,
            created_by_user_id, created_at, updated_at
        FROM rewards
        WHERE id = $1
    `
	reward := &models.Reward{}
	var description sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&reward.ID,
		&reward.RewardName,
		&reward.RewardPoint,
		&description,
		&reward.CreatedByUserID,
		&reward.CreatedAt,
		&reward.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("reward_id", id).Msg("Reward definition not found by ID")
			// Kembalikan ErrNoRows agar handler tahu tidak ditemukan / tidak berhak
			return nil, pgx.ErrNoRows
		}
		// Error umum
		zlog.Error().Err(err).Int("reward_id", id).Msg("Error getting reward definition by ID")
		return nil, fmt.Errorf("error getting reward definition %d: %w", id, err)
	}

	if description.Valid {
		reward.RewardDescription = description.String
	}

	return reward, nil
}

// GetRewardsByCreatorID mengambil daftar reward (paginated) yang dibuat oleh parent tertentu.
func (r *rewardRepo) GetRewardsByCreatorID(ctx context.Context, creatorID int, page, limit int) ([]models.Reward, int, error) {
	// 1. Hitung total
	countQuery := `SELECT COUNT(*) FROM rewards WHERE created_by_user_id = $1`
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, creatorID).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("creator_id", creatorID).Msg("Error counting rewards by creator ID")
		return nil, 0, fmt.Errorf("error counting rewards for creator %d: %w", creatorID, err)
	}

	if totalCount == 0 {
		return []models.Reward{}, 0, nil
	}

	// 2. Hitung Offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 3. Query data
	query := `SELECT id, reward_name, reward_point, reward_description, created_by_user_id, created_at, updated_at
              FROM rewards
              WHERE created_by_user_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, creatorID, limit, offset)
	if err != nil {
		zlog.Error().Err(err).Int("creator_id", creatorID).Msg("Error querying paginated rewards by creator ID")
		return nil, totalCount, fmt.Errorf("error getting rewards for creator %d: %w", creatorID, err)
	}
	defer rows.Close()

	// 4. Scan hasil
	rewards := []models.Reward{}
	for rows.Next() {
		var reward models.Reward
		var description sql.NullString
		scanErr := rows.Scan(
			&reward.ID,
			&reward.RewardName,
			&reward.RewardPoint,
			&description,
			&reward.CreatedByUserID,
			&reward.CreatedAt,
			&reward.UpdatedAt,
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("creator_id", creatorID).Msg("Error scanning reward row (paginated)")
			return rewards, totalCount, fmt.Errorf("error scanning reward data: %w", scanErr)
		}
		if description.Valid {
			reward.RewardDescription = description.String
		}
		rewards = append(rewards, reward)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		zlog.Error().Err(rowsErr).Int("creator_id", creatorID).Msg("Error iterating paginated reward rows")
		return rewards, totalCount, fmt.Errorf("error iterating reward data: %w", rowsErr)
	}

	return rewards, totalCount, nil
}

// GetAvailableRewardsForChild mengambil daftar reward (paginated) yang dibuat oleh SEMUA parent dari anak tersebut.
func (r *rewardRepo) GetAvailableRewardsForChild(ctx context.Context, childID int, page, limit int) ([]models.Reward, int, error) {
	// Query untuk parent ID dari anak
	parentIDsQuery := `SELECT parent_id FROM user_relationship WHERE child_id = $1`
	rowsParents, err := r.db.Query(ctx, parentIDsQuery, childID)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Error getting parent IDs for child")
		return nil, 0, fmt.Errorf("error finding parents for child %d: %w", childID, err)
	}

	parentIDs := []int{} // Slice untuk menampung ID parent
	for rowsParents.Next() {
		var pID int
		if err := rowsParents.Scan(&pID); err != nil {
			rowsParents.Close() // Tutup rows jika ada error scan
			zlog.Warn().Err(err).Int("child_id", childID).Msg("Error scanning parent ID")
			return nil, 0, fmt.Errorf("error processing parent data: %w", err)
		}
		parentIDs = append(parentIDs, pID)
	}
	rowsParents.Close() // Tutup rows setelah selesai

	if len(parentIDs) == 0 {
		// Anak tidak punya parent terdaftar? Atau error sebelumnya?
		zlog.Warn().Int("child_id", childID).Msg("Child has no associated parents found")
		return []models.Reward{}, 0, nil // Kembalikan kosong
	}

	// Konversi slice int ke format yang bisa dipakai di query (misal: ANY($1::int[]))
	// atau bangun query IN (...) secara dinamis (lebih rumit & hati-hati SQL Injection jika tidak pakai parameter)
	// Menggunakan ANY($1::int[]) lebih aman dan efisien dengan pgx.

	// 1. Hitung total reward dari semua parent anak ini
	countQuery := `SELECT COUNT(*) FROM rewards WHERE created_by_user_id = ANY($1::int[])`
	var totalCount int
	err = r.db.QueryRow(ctx, countQuery, parentIDs).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Interface("parent_ids", parentIDs).Msg("Error counting available rewards for child")
		return nil, 0, fmt.Errorf("error counting available rewards for child %d: %w", childID, err)
	}

	if totalCount == 0 {
		return []models.Reward{}, 0, nil
	}

	// 2. Hitung Offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 3. Query reward dari semua parent anak ini dengan pagination
	query := `SELECT id, reward_name, reward_point, reward_description, created_by_user_id, created_at, updated_at
              FROM rewards
              WHERE created_by_user_id = ANY($1::int[])
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, parentIDs, limit, offset)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Error querying available rewards for child")
		return nil, totalCount, fmt.Errorf("error getting available rewards for child %d: %w", childID, err)
	}
	defer rows.Close()

	// 4. Scan hasil
	rewards := []models.Reward{}
	for rows.Next() {
		var reward models.Reward
		var description sql.NullString
		scanErr := rows.Scan(
			&reward.ID,
			&reward.RewardName,
			&reward.RewardPoint,
			&description,
			&reward.CreatedByUserID,
			&reward.CreatedAt,
			&reward.UpdatedAt,
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("child_id", childID).Msg("Error scanning available reward row")
			return rewards, totalCount, fmt.Errorf("error scanning available reward data: %w", scanErr)
		}
		if description.Valid {
			reward.RewardDescription = description.String
		}
		rewards = append(rewards, reward)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		zlog.Error().Err(rowsErr).Int("child_id", childID).Msg("Error iterating available reward rows")
		return rewards, totalCount, fmt.Errorf("error iterating available reward data: %w", rowsErr)
	}

	return rewards, totalCount, nil
}

// UpdateReward memperbarui detail definisi reward.
// Hanya pembuat asli (Strict Ownership) yang bisa mengedit.
func (r *rewardRepo) UpdateReward(ctx context.Context, reward *models.Reward, parentID int) error {
	query := `UPDATE rewards
              SET reward_name = $1, reward_point = $2, reward_description = $3
              WHERE id = $4 AND created_by_user_id = $5` // Validasi ID dan kepemilikan

	tag, err := r.db.Exec(ctx, query,
		reward.RewardName,
		reward.RewardPoint,
		reward.RewardDescription,
		reward.ID, // ID reward yang diupdate
		parentID,  // ID parent yang melakukan request (harus == created_by_user_id)
	)

	if err != nil {
		zlog.Error().Err(err).Int("reward_id", reward.ID).Int("requesting_parent_id", parentID).Msg("Error updating reward definition")
		return fmt.Errorf("error updating reward definition %d: %w", reward.ID, err)
	}

	if tag.RowsAffected() == 0 {
		zlog.Warn().Int("reward_id", reward.ID).Int("requesting_parent_id", parentID).Msg("Reward definition not found or update access denied")
		return pgx.ErrNoRows // Kembalikan ErrNoRows
	}

	zlog.Info().Int("reward_id", reward.ID).Int("requesting_parent_id", parentID).Msg("Reward definition updated successfully")
	return nil
}

// DeleteReward menghapus definisi reward.
// Hanya pembuat asli (Strict Ownership) yang bisa menghapus.
// Perlu penanganan FK dari user_rewards (kita set ON DELETE RESTRICT).
func (r *rewardRepo) DeleteReward(ctx context.Context, id int, parentID int) error {
	// Validasi kepemilikan sebelum menghapus
	query := `DELETE FROM rewards WHERE id = $1 AND created_by_user_id = $2`
	tag, err := r.db.Exec(ctx, query, id, parentID)

	if err != nil {
		// Handle FK violation (user_rewards merujuk ke sini)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("reward_id", id).Msg("Attempted to delete reward definition that is still referenced by user_rewards")
			return fmt.Errorf("cannot delete reward definition: it has been claimed or is pending approval by users")
		}
		// Error umum
		zlog.Error().Err(err).Int("reward_id", id).Int("requesting_parent_id", parentID).Msg("Error deleting reward definition")
		return fmt.Errorf("error deleting reward definition %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		zlog.Warn().Int("reward_id", id).Int("requesting_parent_id", parentID).Msg("Reward definition not found or delete access denied")
		return pgx.ErrNoRows
	}

	zlog.Info().Int("reward_id", id).Int("requesting_parent_id", parentID).Msg("Reward definition deleted successfully")
	return nil
}

// --- Metode Tx untuk Service Layer (jika diperlukan) ---

// GetRewardDetailsTx mengambil detail minimal reward dalam transaksi.
// Implementasi ini SAMA dengan GetRewardByID non-Tx karena tidak ada validasi kepemilikan parent lagi di sini,
// validasi kepemilikan dilakukan di service layer atau handler yang memanggil (berdasarkan relasi anak-parent).
func (r *rewardRepo) GetRewardDetailsTx(ctx context.Context, tx pgx.Tx, rewardID int) (*RewardDetails, error) {
	// Tambahkan created_by_user_id ke SELECT
	query := `SELECT id, reward_point, created_by_user_id
              FROM rewards WHERE id = $1 FOR UPDATE` // Lock baris
	details := &RewardDetails{}
	// Tambahkan &details.CreatedByUserID ke Scan
	err := tx.QueryRow(ctx, query, rewardID).Scan(
        &details.ID,
        &details.RequiredPoints,
        &details.CreatedByUserID,
    )
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		zlog.Error().Err(err).Int("reward_id", rewardID).Msg("RepoTx: Error getting reward details")
		return nil, fmt.Errorf("repoTx error getting reward details %d: %w", rewardID, err)
	}
	return details, nil
}
