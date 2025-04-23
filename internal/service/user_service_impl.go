// internal/service/user_service_impl.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings" // Untuk cek error repo

	"github.com/jackc/pgx/v5" // Untuk pgx.ErrNoRows
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	// "github.com/jackc/pgx/v5/pgconn" // Mungkin tidak perlu di service ini
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils" // Perlu hash
	"github.com/rs/zerolog/log"
	zlog "github.com/rs/zerolog/log"
)

// Definisikan error spesifik service jika perlu
var ErrIncorrectPassword = errors.New("incorrect old password provided")
var ErrUserProfileUpdateFailed = errors.New("failed to update user profile")
var ErrPasswordChangeFailed = errors.New("failed to change password")

type userServiceImpl struct {
	pool        *pgxpool.Pool // Tambahkan Pool untuk transaksi
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository             // Tambahkan RoleRepo
	userRelRepo repository.UserRelationshipRepository // Tambahkan UserRelRepo
	// Tidak perlu pool jika operasi ini tidak butuh transaksi DB
}

// NewUserService creates a new instance of UserService.
func NewUserService(
	pool *pgxpool.Pool, // Terima pool
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository, // Terima RoleRepo
	userRelRepo repository.UserRelationshipRepository, // Terima UserRelRepo
) UserService {
	return &userServiceImpl{
		pool:        pool, // Simpan pool
		userRepo:    userRepo,
		roleRepo:    roleRepo,    // Simpan RoleRepo
		userRelRepo: userRelRepo, // Simpan UserRelRepo
	}
}

// GetUserProfile retrieves user details.
func (s *userServiceImpl) GetUserProfile(ctx context.Context, userID int) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID) // Asumsi GetUserByID memuat role dan tidak memuat password
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("user_id", userID).Msg("Service: User not found for GetUserProfile")
			return nil, ErrUserNotFound // Gunakan error service
		}
		zlog.Error().Err(err).Int("user_id", userID).Msg("Service: Error getting user profile from repository")
		return nil, fmt.Errorf("internal server error retrieving profile: %w", err)
	}
	// Password seharusnya tidak ada di user object dari GetUserByID yang sudah diperbaiki
	// Jika masih ada, pastikan query repo tidak SELECT password
	user.Password = "" // Pastikan password dikosongkan sebelum dikembalikan
	return user, nil
}

// UpdateUserProfile handles updating profile data.
func (s *userServiceImpl) UpdateUserProfile(ctx context.Context, userID int, input *models.UpdateProfileInput) error {
	err := s.userRepo.UpdateUserProfile(ctx, userID, input)
	if err != nil {
		// Cek error spesifik dari repo
		if errors.Is(err, pgx.ErrNoRows) { // User tidak ditemukan
			zlog.Warn().Int("user_id", userID).Msg("Service: User not found during profile update attempt")
			return ErrUserNotFound
		}
		if strings.Contains(err.Error(), "already exists") { // Unique constraint
			zlog.Warn().Err(err).Int("user_id", userID).Msg("Service: Unique constraint violation during user profile update")
			return ErrUsernameOrEmailExists // Gunakan error dari AuthService atau definisikan di sini
		}
		// Error lain
		zlog.Error().Err(err).Int("user_id", userID).Msg("Service: Failed to update user profile in repository")
		return fmt.Errorf("%w: database error", ErrUserProfileUpdateFailed)
	}
	zlog.Info().Int("user_id", userID).Msg("Service: User profile updated successfully")
	return nil
}

// ChangePassword handles changing the user's password.
func (s *userServiceImpl) ChangePassword(ctx context.Context, userID int, input *models.UpdatePasswordInput) error {
	// 1. Dapatkan data user saat ini (termasuk hash password lama)
	currentUser, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Error().Int("user_id", userID).Msg("Service: User not found during password change (inconsistency?)")
			return ErrUserNotFound
		}
		zlog.Error().Err(err).Int("user_id", userID).Msg("Service: Failed to get current user data for password change")
		return fmt.Errorf("%w: failed to retrieve user data", ErrPasswordChangeFailed)
	}

	// 2. Verifikasi password lama
	if !utils.CheckPasswordHash(input.OldPassword, currentUser.Password) {
		zlog.Warn().Int("user_id", userID).Msg("Service: Incorrect old password provided during password change")
		return ErrIncorrectPassword
	}

	// 3. Hash password baru
	newHashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		zlog.Error().Err(err).Int("user_id", userID).Msg("Service: Failed to hash new password")
		return fmt.Errorf("%w: password processing error", ErrPasswordChangeFailed)
	}

	// 4. Update password di repository
	err = s.userRepo.UpdateUserPassword(ctx, userID, newHashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // Seharusnya tidak terjadi
			zlog.Error().Int("user_id", userID).Msg("Service: User disappeared during password update?")
			return ErrUserNotFound
		}
		zlog.Error().Err(err).Int("user_id", userID).Msg("Service: Failed to update password in repository")
		return fmt.Errorf("%w: database error", ErrPasswordChangeFailed)
	}

	zlog.Info().Int("user_id", userID).Msg("Service: User password changed successfully")
	return nil
}

// CreateChildAccount implements creating a child user and linking to parent.
func (s *userServiceImpl) CreateChildAccount(ctx context.Context, parentID int, input *models.CreateChildInput) (childID int, err error) {
	// --- 1. Mulai Transaksi ---
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Service: Failed to begin transaction for CreateChildAccount")
		return 0, fmt.Errorf("internal server error: could not start operation")
	}
	defer func() { // Defer commit/rollback
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			log.Warn().Err(err).Int("parent_id", parentID).Msg("Service: Rolling back transaction due to error during CreateChildAccount")
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				log.Error().Err(err).Int("parent_id", parentID).Msg("Service: Failed to commit transaction for CreateChildAccount")
				err = fmt.Errorf("internal server error: could not finalize child creation")
			} else {
				log.Info().Int("child_id", childID).Int("parent_id", parentID).Msg("Service: Transaction committed for CreateChildAccount")
			}
		}
	}()

	// --- 2. Dapatkan Role ID untuk 'Child' ---
	// Asumsi ada metode GetRoleByName atau kita tahu ID-nya (misal 2)
	// Hardcode ID lebih cepat tapi kurang fleksibel jika ID berubah.
	// Mari kita asumsikan ID 2 adalah 'Child' untuk contoh ini.
	childRoleID := 2 // Ganti jika ID berbeda
	// Anda bisa menambahkan validasi `s.roleRepo.GetRoleByID(ctx, childRoleID)` jika ingin lebih aman

	// --- 3. Hash Password Anak ---
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		log.Error().Err(err).Msg("Service: Failed to hash child password during creation")
		err = fmt.Errorf("%w: password processing error", ErrRegistrationFailed) // Reuse error
		return 0, err                                                            // Rollback
	}

	// --- 4. Buat User Anak dalam Transaksi ---
	// Perlu metode CreateUserTx di UserRepository
	// Ubah input CreateChildInput menjadi RegisterUserInput (karena repo menerima itu)
	registerInput := &models.RegisterUserInput{
		Username:  input.Username,
		Password:  input.Password, // Password asli tidak disimpan, hanya untuk struct
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		RoleID:    childRoleID,
	}
	childID, err = s.userRepo.CreateUserTx(ctx, tx, registerInput, hashedPassword) // Panggil metode Tx
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			log.Warn().Str("username", input.Username).Str("email", input.Email).Msg("Service: Child username or email conflict")
			err = ErrUsernameOrEmailExists
			return 0, err // Rollback
		}
		log.Error().Err(err).Str("username", input.Username).Msg("Service: Error creating child user in repository Tx")
		if strings.Contains(err.Error(), "username already taken") || strings.Contains(err.Error(), "email already taken") {
			err = ErrUsernameOrEmailExists
			return 0, err // Rollback
		}
		err = fmt.Errorf("%w: database error creating child", ErrRegistrationFailed)
		return 0, err // Rollback
	}

	// --- 5. Buat Relasi Parent-Child dalam Transaksi ---
	// Perlu metode AddRelationshipTx di UserRelationshipRepository
	err = s.userRelRepo.AddRelationshipTx(ctx, tx, parentID, childID) // Panggil metode Tx
	if err != nil {
		// Handle error relasi sudah ada (seharusnya tidak terjadi jika user baru) atau FK violation
		log.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Service: Error adding relationship in repository Tx")
		err = fmt.Errorf("%w: database error linking parent/child", ErrRegistrationFailed)
		return 0, err // Rollback
	}

	// Jika semua berhasil, err = nil, defer akan commit
	log.Info().Int("child_id", childID).Int("parent_id", parentID).Msg("Service: Child account created and linked successfully")
	return childID, nil
}
