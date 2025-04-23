// internal/repository/point_transaction_repo.go
package repository

import (
	"context"
	"database/sql" // Untuk sql.NullInt64 (atau sql.NullInt32) dan sql.NullString
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type pointTransactionRepo struct {
	db *pgxpool.Pool
}

// NewPointTransactionRepository membuat instance baru PointTransactionRepository.
func NewPointTransactionRepository(db *pgxpool.Pool) PointTransactionRepository {
	return &pointTransactionRepo{db: db}
}

// CreateTransaction menyimpan record transaksi poin baru.
// Biasanya dipanggil dari dalam Service Layer yang mengelola transaksi DB.
func (r *pointTransactionRepo) CreateTransaction(ctx context.Context, txData *models.PointTransaction) error {
	// Kolom `updated_at` tidak perlu di query INSERT karena ada trigger
	query := `INSERT INTO point_transactions
                (user_id, change_amount, transaction_type, related_user_task_id, related_user_reward_id, created_by_user_id, notes)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`

	// Gunakan Nullable types untuk FK yang opsional
	var relatedTaskID sql.NullInt64 // Gunakan NullInt64 jika ID bisa besar, atau NullInt32
	if txData.RelatedUserTaskID != 0 {
		relatedTaskID = sql.NullInt64{Int64: int64(txData.RelatedUserTaskID), Valid: true}
	}

	var relatedRewardID sql.NullInt64
	if txData.RelatedUserRewardID != 0 {
		relatedRewardID = sql.NullInt64{Int64: int64(txData.RelatedUserRewardID), Valid: true}
	}

	_, err := r.db.Exec(ctx, query,
		txData.UserID,
		txData.ChangeAmount,
		txData.TransactionType,
		relatedTaskID,
		relatedRewardID,
		txData.CreatedByUserID,
		txData.Notes,
	)

	if err != nil {
		// Handle FK violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Interface("transaction_data", txData).Msg("Foreign key violation on point transaction creation")
			// Perlu identifikasi FK mana yang gagal jika ingin pesan lebih spesifik
			return fmt.Errorf("invalid user, creator, task, or reward ID provided for point transaction")
		}
		// Error umum
		zlog.Error().Err(err).Interface("transaction_data", txData).Msg("Error creating point transaction")
		return fmt.Errorf("error creating point transaction: %w", err)
	}

	zlog.Info().Int("user_id", txData.UserID).Int("change", txData.ChangeAmount).Str("type", string(txData.TransactionType)).Msg("Point transaction created successfully")
	return nil
}

// GetTransactionsByUserID mengambil riwayat transaksi poin untuk user tertentu (paginated).
func (r *pointTransactionRepo) GetTransactionsByUserID(ctx context.Context, userID int, page, limit int) ([]models.PointTransaction, int, error) {
	// 1. Hitung total transaksi untuk user ini
	countQuery := `SELECT COUNT(*) FROM point_transactions WHERE user_id = $1`
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&totalCount)
	if err != nil {
		zlog.Error().Err(err).Int("user_id", userID).Msg("Error counting point transactions for user")
		return nil, 0, fmt.Errorf("error counting transactions for user %d: %w", userID, err)
	}

	if totalCount == 0 {
		return []models.PointTransaction{}, 0, nil
	}

	// 2. Hitung Offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 3. Query data transaksi dengan pagination
	query := `SELECT
                id, user_id, change_amount, transaction_type,
                related_user_task_id, related_user_reward_id,
                created_by_user_id, notes, created_at, updated_at
              FROM point_transactions
              WHERE user_id = $1
              ORDER BY created_at DESC -- Tampilkan riwayat terbaru dulu
              LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		zlog.Error().Err(err).Int("user_id", userID).Msg("Error querying paginated point transactions for user")
		return nil, totalCount, fmt.Errorf("error getting transactions for user %d: %w", userID, err)
	}
	defer rows.Close()

	// 4. Scan hasil
	transactions := []models.PointTransaction{}
	for rows.Next() {
		var tx models.PointTransaction
		var relatedTaskID sql.NullInt64
		var relatedRewardID sql.NullInt64
		var notes sql.NullString

		scanErr := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.ChangeAmount,
			&tx.TransactionType,
			&relatedTaskID,
			&relatedRewardID,
			&tx.CreatedByUserID,
			&notes,
			&tx.CreatedAt,
			&tx.UpdatedAt, // Pastikan ada di model dan tabel
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("user_id", userID).Msg("Error scanning point transaction row")
			return transactions, totalCount, fmt.Errorf("error scanning transaction data: %w", scanErr)
		}

		// Assign nullable fields
		if relatedTaskID.Valid {
			tx.RelatedUserTaskID = int(relatedTaskID.Int64)
		} else {
			tx.RelatedUserTaskID = 0 // Atau sesuai default model
		}
		if relatedRewardID.Valid {
			tx.RelatedUserRewardID = int(relatedRewardID.Int64)
		} else {
			tx.RelatedUserRewardID = 0 // Atau sesuai default model
		}
		if notes.Valid {
			tx.Notes = notes.String
		} else {
			tx.Notes = ""
		}

		transactions = append(transactions, tx)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		zlog.Error().Err(rowsErr).Int("user_id", userID).Msg("Error iterating paginated point transaction rows")
		return transactions, totalCount, fmt.Errorf("error iterating transaction data: %w", rowsErr)
	}

	return transactions, totalCount, nil
}

// CalculateTotalPointsByUserID menghitung total poin saat ini untuk user tertentu.
func (r *pointTransactionRepo) CalculateTotalPointsByUserID(ctx context.Context, userID int) (int, error) {
	query := `SELECT COALESCE(SUM(change_amount), 0) FROM point_transactions WHERE user_id = $1`
	var totalPoints int
	err := r.db.QueryRow(ctx, query, userID).Scan(&totalPoints)
	if err != nil {
		// Error ini seharusnya jarang terjadi kecuali masalah koneksi atau query salah
		zlog.Error().Err(err).Int("user_id", userID).Msg("Error calculating total points for user")
		// Jika user belum ada transaksi, SUM akan NULL, tapi COALESCE(..., 0) menanganinya.
		// Jadi, jika error bukan ErrNoRows (yang seharusnya tidak terjadi di SUM), ini error serius.
		return 0, fmt.Errorf("error calculating points for user %d: %w", userID, err)
	}

	zlog.Debug().Int("user_id", userID).Int("total_points", totalPoints).Msg("Calculated total points")
	return totalPoints, nil
}

// --- Metode Tx untuk Service Layer ---

// CreateTransactionTx menyimpan transaksi poin dalam konteks transaksi DB yang lebih besar.
func (r *pointTransactionRepo) CreateTransactionTx(ctx context.Context, tx pgx.Tx, txData *models.PointTransaction) error {
	query := `INSERT INTO point_transactions
                (user_id, change_amount, transaction_type, related_user_task_id, related_user_reward_id, created_by_user_id, notes)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`

	var relatedTaskID sql.NullInt64
	if txData.RelatedUserTaskID != 0 {
		relatedTaskID = sql.NullInt64{Int64: int64(txData.RelatedUserTaskID), Valid: true}
	}
	var relatedRewardID sql.NullInt64
	if txData.RelatedUserRewardID != 0 {
		relatedRewardID = sql.NullInt64{Int64: int64(txData.RelatedUserRewardID), Valid: true}
	}

	_, err := tx.Exec(ctx, query, // Gunakan tx.Exec bukan r.db.Exec
		txData.UserID,
		txData.ChangeAmount,
		txData.TransactionType,
		relatedTaskID,
		relatedRewardID,
		txData.CreatedByUserID,
		txData.Notes,
	)

	if err != nil {
		// Handle FK violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Interface("transaction_data", txData).Msg("RepoTx: Foreign key violation on point transaction creation")
			return fmt.Errorf("invalid user, creator, task, or reward ID provided for point transaction")
		}
		// Error umum
		zlog.Error().Err(err).Interface("transaction_data", txData).Msg("RepoTx: Error creating point transaction")
		return fmt.Errorf("repoTx error creating point transaction: %w", err)
	}
	return nil // Sukses dalam konteks transaksi ini
}

// CalculateTotalPointsByUserIDTx menghitung total poin dalam transaksi (mungkin perlu locking).
func (r *pointTransactionRepo) CalculateTotalPointsByUserIDTx(ctx context.Context, tx pgx.Tx, userID int) (int, error) {
	// Tambahkan 'FOR UPDATE' pada tabel point_transactions jika perlu mencegah race condition
	// saat membaca total poin dalam sebuah transaksi yang mungkin akan mengubahnya.
	// Namun, ini bisa menyebabkan contention. Alternatifnya adalah mengandalkan isolation level DB.
	// Untuk perhitungan sederhana, 'FOR UPDATE' mungkin tidak perlu jika isolasi READ COMMITTED cukup.
	// Jika Anda menggunakan SELECT SUM(...) lalu INSERT/UPDATE berdasarkan hasil SUM itu,
	// Anda MUNGKIN perlu locking yang lebih cermat atau menggunakan CTE dengan locking.
	// Untuk 'ClaimReward', mungkin cukup lock baris reward dan user (jika ada tabel user balance).
	// Karena kita pakai ledger, lock mungkin tidak sekrusial itu untuk SUM saja.
	query := `SELECT COALESCE(SUM(change_amount), 0) FROM point_transactions WHERE user_id = $1`
	var totalPoints int
	err := tx.QueryRow(ctx, query, userID).Scan(&totalPoints)
	if err != nil {
		zlog.Error().Err(err).Int("user_id", userID).Msg("RepoTx: Error calculating total points for user")
		return 0, fmt.Errorf("repoTx error calculating points for user %d: %w", userID, err)
	}
	return totalPoints, nil
}
