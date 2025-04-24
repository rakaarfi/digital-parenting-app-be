// internal/api/v1/handlers/auth_handler.go
package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	zlog "github.com/rs/zerolog/log"
)

type AuthHandler struct {
	AuthService service.AuthService // Ganti repo dengan service
	Validate    *validator.Validate
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
		Validate:    validator.New(),
	}
}

// Register godoc
// @Summary Register New User
// @Description Creates a new user account.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param register body models.RegisterUserInput true "User Registration Details"
// @Success 201 {object} models.Response{data=map[string]int} "User registered successfully, returns user ID"
// @Failure 400 {object} models.Response{data=map[string]string} "Validation failed or invalid request body"
// @Failure 409 {object} models.Response "Username or Email already exists"
// @Failure 500 {object} models.Response "Internal server error during registration"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	input := new(models.RegisterUserInput)

	// Parse body (tetap di handler)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body during registration")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid request body",
		})
	}

	// Validate input (tetap di handler)
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Validation failed during registration")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Validation failed", Data: errorDetails,
		})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	userID, err := h.AuthService.RegisterUser(ctx, input) // Panggil service
	if err != nil {
		// --- Tangani Error dari Service ---
		if errors.Is(err, service.ErrUsernameOrEmailExists) {
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false, Message: service.ErrUsernameOrEmailExists.Error(),
			})
		}
		if errors.Is(err, service.ErrRoleNotFound) {
			// Berikan pesan yang lebih spesifik ke user jika perlu
			return c.Status(fiber.StatusBadRequest).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("Role with ID %d not found", input.RoleID),
			})
		}
		if errors.Is(err, service.ErrDisallowedRoleRegistration) {
            // Kembalikan 400 Bad Request atau 403 Forbidden
            return c.Status(fiber.StatusBadRequest).JSON(models.Response{
                Success: false, Message: service.ErrDisallowedRoleRegistration.Error(),
            })
        }
		// Handle error registrasi generik atau internal lainnya
		zlog.Error().Err(err).Str("username", input.Username).Msg("Handler: Error returned from AuthService.RegisterUser")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to register user", // Pesan generik ke user
		})
	}

	// Sukses
	zlog.Info().Int("userID", userID).Str("username", input.Username).Msg("Handler: User registered successfully via service")
	return c.Status(fiber.StatusCreated).JSON(models.Response{
		Success: true,
		Message: "User registered successfully",
		Data:    fiber.Map{"user_id": userID},
	})
}

// Login godoc
// @Summary User Login
// @Description Authenticates a user and returns a JWT token upon successful login.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param login body models.LoginUserInput true "Login Credentials"
// @Success 200 {object} models.Response{data=map[string]string} "Login successful, returns JWT token"
// @Failure 400 {object} models.Response{data=map[string]string} "Validation failed or invalid request body"
// @Failure 401 {object} models.Response "Invalid username or password"
// @Failure 500 {object} models.Response "Internal server error during login"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	input := new(models.LoginUserInput)

	// Parse body
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body during login")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid request body",
		})
	}

	// Validate input
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Validation failed during login")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Validation failed", Data: errorDetails,
		})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	token, err := h.AuthService.LoginUser(ctx, input) // Panggil service
	if err != nil {
		// --- Tangani Error dari Service ---
		if errors.Is(err, service.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.Response{
				Success: false, Message: service.ErrInvalidCredentials.Error(), // Pesan "Invalid username or password"
			})
		}
		// Handle error login generik atau internal lainnya
		zlog.Error().Err(err).Str("username", input.Username).Msg("Handler: Error returned from AuthService.LoginUser")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Login process failed", // Pesan generik ke user
		})
	}

	// Sukses
	zlog.Info().Str("username", input.Username).Msg("Handler: User logged in successfully via service")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true,
		Message: "Login successful",
		Data:    fiber.Map{"token": token},
	})
}
