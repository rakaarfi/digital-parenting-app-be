// internal/api/v1/handlers/user_handler.go
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	zlog "github.com/rs/zerolog/log"
)

type UserHandler struct {
	UserService service.UserService
	Validate    *validator.Validate
}

func NewUserHandler(
	userService service.UserService,
) *UserHandler {
	return &UserHandler{
		UserService: userService,
		Validate:    validator.New(),
	}
}

// UpdateMyProfile godoc
// @Summary Update my profile
// @Description Update the profile for the current user.
// @Tags User - Profile Management
// @Accept json
// @Produce json
// @Param update_profile_input body models.UpdateProfileInput true "Profile update information"
// @Success 200 {object} models.Response "Profile updated successfully"
// @Failure 400 {object} models.Response "Validation failed or invalid request body"
// @Failure 401 {object} models.Response "Unauthorized: Invalid token"
// @Failure 404 {object} models.Response "User not found"
// @Failure 409 {object} models.Response "Username or Email already exists"
// @Failure 500 {object} models.Response "Internal server error during profile update"
// @Security ApiKeyAuth
// @Router /user/profile [patch]
func (h *UserHandler) UpdateMyProfile(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract userID from JWT for profile update")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{ // 401 lebih cocok
			Success: false, Message: "Unauthorized: Invalid token",
		})
	}

	input := new(models.UpdateProfileInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body during profile update")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid request body",
		})
	}

	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("user_id", userID).Msg("Handler: Update profile validation failed")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Validation failed", Data: errorDetails,
		})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	err = h.UserService.UpdateUserProfile(ctx, userID, input) // Panggil service
	if err != nil {
		// --- Tangani Error dari Service ---
		if errors.Is(err, service.ErrUserNotFound) { // Cek error service
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: "User not found", // Pesan generik
			})
		}
		if errors.Is(err, service.ErrUsernameOrEmailExists) { // Cek error service
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false, Message: service.ErrUsernameOrEmailExists.Error(),
			})
		}
		// Error internal lain
		zlog.Error().Err(err).Int("user_id", userID).Msg("Handler: Error returned from UserService.UpdateUserProfile")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to update profile",
		})
	}

	// Sukses
	zlog.Info().Int("user_id", userID).Msg("Handler: User profile updated successfully via service")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Profile updated successfully",
	})
}

// UpdateMyPassword godoc
// @Summary Update My Password
// @Description Updates the current user's password.
// @Tags User - Profile Management
// @Accept json
// @Produce json
// @Param update_password body models.UpdatePasswordInput true "Password Update Details"
// @Success 200 {object} models.Response "Password updated successfully"
// @Failure 400 {object} models.Response "Validation failed or invalid request body"
// @Failure 401 {object} models.Response "Unauthorized: Invalid token or incorrect old password"
// @Failure 404 {object} models.Response "User not found"
// @Failure 500 {object} models.Response "Internal server error during password update"
// @Security ApiKeyAuth
// @Router /user/password [patch]
func (h *UserHandler) UpdateMyPassword(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract userID from JWT for password update")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{
			Success: false, Message: "Unauthorized: Invalid token",
		})
	}

	input := new(models.UpdatePasswordInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body during password update")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid request body",
		})
	}

	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("user_id", userID).Msg("Handler: Update password validation failed")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Validation failed", Data: errorDetails,
		})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	err = h.UserService.ChangePassword(ctx, userID, input) // Panggil service
	if err != nil {
		// --- Tangani Error dari Service ---
		if errors.Is(err, service.ErrIncorrectPassword) { // Cek error service
			return c.Status(fiber.StatusUnauthorized).JSON(models.Response{ // 401 cocok
				Success: false, Message: service.ErrIncorrectPassword.Error(),
			})
		}
		if errors.Is(err, service.ErrUserNotFound) { // Cek error service
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: "User not found",
			})
		}
		// Error internal lain
		zlog.Error().Err(err).Int("user_id", userID).Msg("Handler: Error returned from UserService.ChangePassword")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to change password",
		})
	}

	// Sukses
	zlog.Info().Int("user_id", userID).Msg("Handler: User password updated successfully via service")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Password updated successfully",
	})
}

// GetMyProfile godoc
// @Summary Get my profile
// @Description Get the profile for the current user.
// @Tags User - Profile Management
// @Produce json
// @Success 200 {object} models.Response{data=map[string]interface{}} "Profile data for current user"
// @Failure 401 {object} models.Response "Unauthorized: Invalid token"
// @Failure 404 {object} models.Response "User profile not found"
// @Failure 500 {object} models.Response "Internal server error during profile retrieval"
// @Security ApiKeyAuth
// @Router /user/profile [get]
func (h *UserHandler) GetMyProfile(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract userID from JWT for get profile")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{
			Success: false, Message: "Unauthorized: Invalid token",
		})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	userProfile, err := h.UserService.GetUserProfile(ctx, userID) // Panggil service
	if err != nil {
		// --- Tangani Error dari Service ---
		if errors.Is(err, service.ErrUserNotFound) { // Cek error service
			// Ini aneh tapi tangani saja
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: "User profile not found",
			})
		}
		// Error internal lain
		zlog.Error().Err(err).Int("user_id", userID).Msg("Handler: Error returned from UserService.GetUserProfile")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to retrieve profile",
		})
	}

	// Sukses
	zlog.Info().Int("user_id", userID).Msg("Handler: User profile retrieved successfully via service")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Profile retrieved successfully", Data: userProfile,
	})
}
