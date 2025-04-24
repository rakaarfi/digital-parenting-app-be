// internal/service/auth_service_impl.go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings" // Import strings

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils" // Perlu utils untuk hash & jwt
	zlog "github.com/rs/zerolog/log"
)

// Definisikan error spesifik jika perlu
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrRoleNotFound = errors.New("role not found")
var ErrRegistrationFailed = errors.New("failed to register user")
var ErrLoginFailed = errors.New("failed to login")
var ErrUsernameOrEmailExists = errors.New("username or email already exists")
var ErrDisallowedRoleRegistration = errors.New("registration for this role type is not allowed through this endpoint")

type authServiceImpl struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	// Tidak perlu pool jika tidak ada transaksi DB di sini
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(userRepo repository.UserRepository, roleRepo repository.RoleRepository) AuthService {
	return &authServiceImpl{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// RegisterUser implements the registration logic.
func (s *authServiceImpl) RegisterUser(ctx context.Context, input *models.RegisterUserInput) (int, error) {
	// --- VALIDASI ROLE YANG DIIZINKAN UNTUK REGISTRASI PUBLIK ---
	// Asumsi ID 1 = Parent, ID 3 = Admin (sesuaikan jika berbeda)
	// Role 'Child' (ID 2) tidak boleh mendaftar sendiri.
	if input.RoleID != 1 && input.RoleID != 3 { // Hanya izinkan Role ID 1 (Parent) dan 3 (Admin)
		zlog.Warn().Int("role_id", input.RoleID).Msg("Service: Attempt to register with disallowed role via public endpoint")
		// Kembalikan error yang jelas
		return 0, ErrDisallowedRoleRegistration
	}

	// 1. Validasi Role ID (panggil repo)
	_, err := s.roleRepo.GetRoleByID(ctx, input.RoleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("role_id", input.RoleID).Msg("Service: Role ID provided during registration not found in DB")
			return 0, ErrRoleNotFound // Kembalikan error spesifik service
		}
		zlog.Error().Err(err).Int("role_id", input.RoleID).Msg("Service: Error validating role during registration")
		return 0, fmt.Errorf("%w: could not validate role", ErrRegistrationFailed) // Bungkus error generik
	}

	// 2. Hash Password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		zlog.Error().Err(err).Msg("Service: Failed to hash password during registration")
		return 0, fmt.Errorf("%w: password processing error", ErrRegistrationFailed)
	}
	// zlog.Debug().Str("username", input.Username).Msg("Service: Password hashed") // Log di service jika perlu

	// 3. Create User di DB (panggil repo)
	userID, err := s.userRepo.CreateUser(ctx, input, hashedPassword)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			zlog.Warn().Str("username", input.Username).Str("email", input.Email).Msg("Service: Username or email conflict during registration")
			return 0, ErrUsernameOrEmailExists // Error spesifik
		}
		zlog.Error().Err(err).Str("username", input.Username).Msg("Service: Error creating user in repository")
		// Cek apakah error asli adalah "username already taken" dari repo, jika iya, gunakan ErrUsernameOrEmailExists
		if strings.Contains(err.Error(), "username already taken") { // Perbaikan dari repo sebelumnya
			return 0, ErrUsernameOrEmailExists
		}
		return 0, fmt.Errorf("%w: database error", ErrRegistrationFailed) // Error generik DB
	}

	zlog.Info().Int("userID", userID).Str("username", input.Username).Msg("Service: User registered successfully")
	return userID, nil // Sukses
}

// LoginUser implements the login logic.
func (s *authServiceImpl) LoginUser(ctx context.Context, input *models.LoginUserInput) (string, error) {
	// 1. Get User by Username (panggil repo)
	user, err := s.userRepo.GetUserByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Info().Str("username", input.Username).Msg("Service: User not found during login attempt")
			return "", ErrInvalidCredentials // Error spesifik tapi generik untuk keamanan
		}
		zlog.Error().Err(err).Str("username", input.Username).Msg("Service: Error fetching user during login")
		return "", fmt.Errorf("%w: database error retrieving user", ErrLoginFailed)
	}

	// 2. Check Password
	if !utils.CheckPasswordHash(input.Password, user.Password) {
		zlog.Info().Str("username", input.Username).Msg("Service: Invalid password provided during login attempt")
		return "", ErrInvalidCredentials // Error spesifik tapi generik
	}

	// 3. Check User Role (Pastikan role ter-load dari repo GetUserByUsername)
	if user.Role == nil || user.Role.Name == "" {
		zlog.Error().Int("user_id", user.ID).Str("username", user.Username).Msg("Service: User role data is missing after successful authentication")
		return "", fmt.Errorf("%w: user configuration error", ErrLoginFailed)
	}

	// 4. Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Username, user.Role.Name)
	if err != nil {
		zlog.Error().Err(err).Str("username", input.Username).Msg("Service: Error generating JWT")
		return "", fmt.Errorf("%w: token generation error", ErrLoginFailed)
	}

	zlog.Info().Str("username", input.Username).Msg("Service: User logged in successfully")
	return token, nil // Sukses, kembalikan token
}
