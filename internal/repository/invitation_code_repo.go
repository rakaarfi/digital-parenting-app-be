// internal/repository/invitation_code_repo.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type invitationCodeRepo struct {
	db *pgxpool.Pool
}

// NewInvitationCodeRepository creates a new instance of InvitationCodeRepository.
func NewInvitationCodeRepository(db *pgxpool.Pool) InvitationCodeRepository {
	return &invitationCodeRepo{db: db}
}

// CreateCode stores a new invitation code in the database.
func (r *invitationCodeRepo) CreateCode(ctx context.Context, code string, childID int, parentID int, expiresAt time.Time) error {
	query := `INSERT INTO invitation_codes
                (code, child_id, created_by_parent_id, status, expires_at)
              VALUES ($1, $2, $3, $4, $5)`
	initialStatus := models.InvitationStatusActive

	_, err := r.db.Exec(ctx, query, code, childID, parentID, initialStatus, expiresAt)
	if err != nil {
		// Handle unique constraint violation for 'code'
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			// Periksa constraint name jika perlu untuk lebih spesifik
			if pgErr.ConstraintName == "invitation_codes_code_key" { // Asumsi nama constraint
				zlog.Warn().Err(err).Str("code", code).Msg("Attempted to create duplicate invitation code")
				// Ini seharusnya jarang terjadi jika generator kode bagus, tapi bisa return error spesifik
				return fmt.Errorf("generated invitation code conflict, please try again")
			}
		}
		// Handle FK violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Err(err).Int("child_id", childID).Int("parent_id", parentID).Msg("Foreign key violation on invitation code creation")
			return fmt.Errorf("invalid child or parent ID provided for invitation")
		}
		// Error umum
		zlog.Error().Err(err).Str("code", code).Msg("Error creating invitation code")
		return fmt.Errorf("error creating invitation code: %w", err)
	}
	zlog.Info().Str("code", code).Int("child_id", childID).Int("parent_id", parentID).Msg("Invitation code created successfully")
	return nil
}

// FindActiveCode retrieves an invitation code only if it's active and not expired.
func (r *invitationCodeRepo) FindActiveCode(ctx context.Context, code string) (*models.InvitationCode, error) {
	query := `SELECT
                id, code, child_id, created_by_parent_id, status, expires_at, created_at, updated_at
              FROM invitation_codes
              WHERE code = $1
                AND status = $2
                AND expires_at > NOW()` // Cek status 'active' dan belum expired
	invCode := &models.InvitationCode{}
	activeStatus := models.InvitationStatusActive

	err := r.db.QueryRow(ctx, query, code, activeStatus).Scan(
		&invCode.ID,
		&invCode.Code,
		&invCode.ChildID,
		&invCode.CreatedByParentID,
		&invCode.Status,
		&invCode.ExpiresAt,
		&invCode.CreatedAt,
		&invCode.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Kode tidak ditemukan, atau tidak aktif, atau sudah expired
			zlog.Warn().Str("code", code).Msg("Active invitation code not found or expired/used")
			return nil, pgx.ErrNoRows // Kembalikan ErrNoRows agar service tahu
		}
		// Error query lain
		zlog.Error().Err(err).Str("code", code).Msg("Error finding active invitation code")
		return nil, fmt.Errorf("error finding invitation code '%s': %w", code, err)
	}

	return invCode, nil
}

// MarkCodeAsUsedTx updates the status of a code to 'used' within a transaction.
func (r *invitationCodeRepo) MarkCodeAsUsedTx(ctx context.Context, tx pgx.Tx, code string) error {
	query := `UPDATE invitation_codes
              SET status = $1
              WHERE code = $2 AND status = $3` // Pastikan masih active sebelum diubah
	usedStatus := models.InvitationStatusUsed
	activeStatus := models.InvitationStatusActive

	tag, err := tx.Exec(ctx, query, usedStatus, code, activeStatus)
	if err != nil {
		zlog.Error().Err(err).Str("code", code).Msg("RepoTx: Error marking invitation code as used")
		return fmt.Errorf("repoTx error updating invitation code status for '%s': %w", code, err)
	}

	if tag.RowsAffected() == 0 {
		// Gagal update, kemungkinan kode tidak ditemukan atau statusnya sudah bukan 'active' lagi (race condition?)
		zlog.Warn().Str("code", code).Msg("RepoTx: Failed to mark invitation code as used (not found or not active)")
		// Query ulang untuk cek status? Atau kembalikan error spesifik?
		// Mengembalikan ErrNoRows mungkin membingungkan, lebih baik error spesifik.
		var exists bool
		checkErr := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invitation_codes WHERE code = $1)`, code).Scan(&exists)
		if checkErr != nil { // Error saat cek ulang
            return fmt.Errorf("failed to mark code '%s' as used and could not verify existence", code)
        }
        if !exists {
            return pgx.ErrNoRows // Kode memang tidak ada
        }
		// Kode ada tapi status bukan active
		return fmt.Errorf("failed to mark code '%s' as used: code is not currently active", code)
	}

	zlog.Info().Str("code", code).Msg("RepoTx: Invitation code marked as used successfully")
	return nil
}

// DeleteExpiredCodes removes expired invitation codes.
func (r *invitationCodeRepo) DeleteExpiredCodes(ctx context.Context) (int64, error) {
	query := `DELETE FROM invitation_codes WHERE expires_at <= NOW()`
	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		zlog.Error().Err(err).Msg("Error deleting expired invitation codes")
		return 0, fmt.Errorf("error deleting expired codes: %w", err)
	}

	rowsAffected := tag.RowsAffected()
	zlog.Info().Int64("deleted_count", rowsAffected).Msg("Expired invitation codes deleted")
	return rowsAffected, nil
}