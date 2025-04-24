// internal/service/invitation_service_impl.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	// "github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	zlog "github.com/rs/zerolog/log"
)

// Definisikan error spesifik service
var ErrCodeGenerationFailed = errors.New("failed to generate a unique invitation code")
var ErrInvalidInvitationCode = errors.New("invalid, expired, or already used invitation code")
var ErrUserNotParent = errors.New("user accepting invitation is not a parent")
var ErrAlreadyParent = errors.New("user is already a parent of this child")
var ErrCannotInviteSelf = errors.New("cannot accept an invitation for your own child's code") // Seharusnya tidak terjadi jika alur benar
var ErrInvitationFailed = errors.New("failed to process invitation")

const (
	invitationCodeLength     = 10                 // Panjang kode undangan (bisa disesuaikan)
	invitationCodeValidity   = 7 * 24 * time.Hour // Masa berlaku kode (misal: 7 hari)
	maxCodeGenerationRetries = 5                  // Maksimum percobaan jika terjadi collision kode
)

type invitationServiceImpl struct {
	pool        *pgxpool.Pool // Untuk transaksi
	invRepo     repository.InvitationCodeRepository
	userRelRepo repository.UserRelationshipRepository
	userRepo    repository.UserRepository // Untuk cek role user
	// RoleRepo mungkin tidak perlu jika kita bisa cek nama role dari UserRepo
}

// NewInvitationService creates a new instance of InvitationService.
func NewInvitationService(
	pool *pgxpool.Pool,
	invRepo repository.InvitationCodeRepository,
	userRelRepo repository.UserRelationshipRepository,
	userRepo repository.UserRepository,
) InvitationService {
	return &invitationServiceImpl{
		pool:        pool,
		invRepo:     invRepo,
		userRelRepo: userRelRepo,
		userRepo:    userRepo,
	}
}

// GenerateAndStoreCode creates and stores a unique invitation code.
func (s *invitationServiceImpl) GenerateAndStoreCode(ctx context.Context, requestingParentID int, childID int) (string, error) {
	log := zlog.With().Int("requestingParentID", requestingParentID).Int("childID", childID).Logger()

	// 1. Validasi: Pastikan requestingParentID adalah parent dari childID
	isParent, err := s.userRelRepo.IsParentOf(ctx, requestingParentID, childID) // Cek non-Tx OK di sini
	if err != nil {
		log.Error().Err(err).Msg("Service: Error checking parent-child relationship for generating code")
		return "", fmt.Errorf("internal server error: could not verify relationship")
	}
	if !isParent {
		log.Warn().Msg("Service: Attempt to generate invitation code by non-parent")
		return "", fmt.Errorf("forbidden: you are not authorized to generate codes for this child")
	}

	// 2. Generate kode unik dengan retry mechanism
	var code string
	var createErr error
	expiresAt := time.Now().Add(invitationCodeValidity)

	for i := 0; i < maxCodeGenerationRetries; i++ {
		code, err = utils.GenerateRandomString(invitationCodeLength)
		if err != nil {
			log.Error().Err(err).Msg("Service: Failed to generate random string for invitation code")
			return "", ErrCodeGenerationFailed // Error saat generate random
		}

		createErr = s.invRepo.CreateCode(ctx, code, childID, requestingParentID, expiresAt)
		if createErr == nil {
			// Kode berhasil dibuat dan disimpan, keluar loop
			log.Info().Str("code", code).Msg("Service: Invitation code generated and stored successfully")
			return code, nil
		}

		// Cek apakah error karena kode duplikat (unique constraint)
		if pgErr, ok := createErr.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "invitation_codes_code_key") {
				log.Warn().Int("attempt", i+1).Msg("Service: Generated invitation code collision, retrying...")
				continue // Coba generate lagi
			}
		}

		// Error lain saat menyimpan kode (misal FK violation, DB error), hentikan retry
		log.Error().Err(createErr).Str("code", code).Msg("Service: Failed to store generated invitation code")
		break // Keluar loop jika error bukan collision
	}

	// Jika loop selesai karena error non-collision atau max retries tercapai
	return "", fmt.Errorf("%w: %w", ErrCodeGenerationFailed, createErr) // Kembalikan error code generation gagal
}

// AcceptInvitation validates a code and links the joining parent to the child. Uses transaction.
func (s *invitationServiceImpl) AcceptInvitation(ctx context.Context, joiningParentID int, code string) (err error) {
	log := zlog.With().Int("joiningParentID", joiningParentID).Str("code", code).Logger()

	// --- 1. Mulai Transaksi ---
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Service: Failed to begin transaction for AcceptInvitation")
		return fmt.Errorf("internal server error: could not start operation")
	}
	defer func() { // Defer commit/rollback
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			log.Warn().Err(err).Msg("Service: Rolling back transaction due to error during AcceptInvitation")
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Service: Failed to commit transaction for AcceptInvitation")
				err = fmt.Errorf("internal server error: could not finalize invitation acceptance")
			} else {
				log.Info().Msg("Service: Transaction committed successfully for AcceptInvitation")
			}
		}
	}()

	// --- 2. Cari Kode Undangan Aktif ---
	invCode, err := s.invRepo.FindActiveCode(ctx, code) // Repo ini tidak perlu Tx karena hanya read
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Warn().Msg("Service: Invalid, expired, or used invitation code provided")
			err = ErrInvalidInvitationCode // Error spesifik service
			return err                     // Rollback
		}
		log.Error().Err(err).Msg("Service: Error finding active invitation code")
		err = fmt.Errorf("%w: database error finding code", ErrInvitationFailed)
		return err // Rollback
	}

	// --- 3. Validasi User yang Bergabung ---
	// a. Dapatkan detail user yang join
	joiningUser, err := s.userRepo.GetUserByID(ctx, joiningParentID) // Repo ini tidak perlu Tx
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error().Msg("Service: Joining user ID from JWT not found in DB (inconsistency!)")
			err = fmt.Errorf("%w: joining user not found", ErrInvitationFailed)
			return err // Rollback
		}
		log.Error().Err(err).Msg("Service: Error fetching joining user details")
		err = fmt.Errorf("%w: error fetching user details", ErrInvitationFailed)
		return err // Rollback
	}
	// b. Pastikan role-nya 'Parent'
	if joiningUser.Role == nil || !strings.EqualFold(joiningUser.Role.Name, "Parent") {
		log.Warn().Int("user_id", joiningParentID).Str("role", joiningUser.Role.Name).Msg("Service: User attempting to accept invitation is not a Parent")
		err = ErrUserNotParent
		return err // Rollback
	}
	// c. Pastikan joining parent bukan parent yang membuat kode untuk anaknya sendiri
	if joiningParentID == invCode.CreatedByParentID {
		log.Warn().Msg("Service: Parent attempted to accept invitation code they generated")
		// Ini seharusnya tidak terjadi jika alur UI benar, tapi validasi tetap penting
		err = fmt.Errorf("cannot accept an invitation generated by yourself for your child")
		return err // Rollback
	}

	// --- 4. Cek Apakah Relasi Sudah Ada (dalam Transaksi) ---
	isAlreadyParent, err := s.userRelRepo.IsParentOfTx(ctx, tx, joiningParentID, invCode.ChildID)
	if err != nil {
		log.Error().Err(err).Msg("Service: Error checking existing relationship within transaction")
		err = fmt.Errorf("%w: error checking existing relationship", ErrInvitationFailed)
		return err // Rollback
	}
	if isAlreadyParent {
		log.Warn().Msg("Service: Joining parent is already linked to the child")
		err = ErrAlreadyParent
		return err // Rollback
	}

	// --- 5. Tambahkan Relasi Parent-Child Baru (dalam Transaksi) ---
	err = s.userRelRepo.AddRelationshipTx(ctx, tx, joiningParentID, invCode.ChildID)
	if err != nil {
		// Handle error spesifik dari repo jika ada (misal FK violation - seharusnya tidak terjadi)
		log.Error().Err(err).Msg("Service: Failed to add relationship within transaction")
		err = fmt.Errorf("%w: database error adding relationship", ErrInvitationFailed)
		return err // Rollback
	}

	// --- 6. Tandai Kode Sudah Digunakan (dalam Transaksi) ---
	err = s.invRepo.MarkCodeAsUsedTx(ctx, tx, code)
	if err != nil {
		// Handle error spesifik dari repo (misal, kode sudah tidak aktif lagi karena race condition)
		log.Error().Err(err).Msg("Service: Failed to mark invitation code as used within transaction")
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "not currently active") {
			err = ErrInvalidInvitationCode // Jika kode sudah tidak aktif
		} else {
			err = fmt.Errorf("%w: database error updating code status", ErrInvitationFailed)
		}
		return err // Rollback
	}

	// Jika semua berhasil, err = nil, defer akan commit
	log.Info().Int("child_id", invCode.ChildID).Msg("Service: Parent successfully joined child via invitation code")
	return nil // Sukses
}
