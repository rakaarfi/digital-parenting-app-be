// internal/api/v1/handlers/admin_handler.go
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	zlog "github.com/rs/zerolog/log"
)

type AdminHandler struct {
	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository
	Validate *validator.Validate
}

func NewAdminHandler(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
) *AdminHandler {
	return &AdminHandler{
		UserRepo: userRepo,
		RoleRepo: roleRepo,
		Validate: validator.New(),
	}
}

// -------------------------------------------------------------------------
// User Management
// -------------------------------------------------------------------------

// GetAllUsers godoc
// @Summary Get All Users (Admin)
// @Description Retrieves a paginated list of all users. Requires Admin role.
// @Tags Admin - Users Management
// @Accept json
// @Produce json
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10) maximum(100)
// @Success 200 {object} map[string]interface{} "Successfully retrieved users with pagination metadata"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized (Invalid or missing token)"
// @Failure 403 {object} models.Response "Forbidden (User is not an Admin)"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /admin/users [get]
func (h *AdminHandler) GetAllUsers(c *fiber.Ctx) error {
	// --- 1. Baca dan Validasi Parameter Pagination ---
	pagination := utils.ParsePaginationParams(c)

	// --- 2. Panggil Repository dengan Parameter Pagination ---
	ctx := c.Context()
	users, totalCount, err := h.UserRepo.GetAllUsers(ctx, pagination.Page, pagination.Limit)
	if err != nil {
		// Error sudah di-log di repo, tapi log di handler juga baik untuk konteks request
		zlog.Error().Err(err).Int("page", pagination.Page).Int("limit", pagination.Limit).Msg("Failed to get users from repository (paginated)")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to retrieve users",
		})
	}

	// --- 3. Siapkan Response dengan Metadata ---
	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Users retrieved successfully", users, meta)

	zlog.Info().
		Int("page", pagination.Page).
		Int("limit", pagination.Limit).
		Int("returned_count", len(users)).
		Int("total_count", totalCount).
		Msg("Successfully retrieved paginated users for admin request")

	return c.Status(http.StatusOK).JSON(response)
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Retrieves a user by its ID.
// @Tags Admin - Users Management
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} models.Response{data=models.User} "User retrieved successfully"
// @Failure 400 {object} models.Response "Invalid User ID parameter"
// @Failure 404 {object} models.Response "User not found"
// @Failure 500 {object} models.Response "Internal server error during user retrieval"
// @Security ApiKeyAuth
// @Router /admin/users/{userId} [get]
func (h *AdminHandler) GetUserByID(c *fiber.Ctx) error {
	userIdStr := c.Params("userId")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", userIdStr).Msg("Invalid User ID parameter")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid User ID parameter",
		})
	}

	adminUserId, _ := utils.ExtractUserIDFromJWT(c) // Abaikan error sementara jika hanya untuk log

	ctx := c.Context()
	user, err := h.UserRepo.GetUserByID(ctx, userId)
	if err != nil {
		// --- CEK NOT FOUND ---
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("requested_user_id", userId).Msg("Admin requested non-existent user")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("User with ID %d not found", userId),
			})
		}
		zlog.Error().Err(err).Int("user_id", userId).Msg("Failed to get user from repository")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to retrieve user",
		})
	}
	// Logging sukses
	zlog.Info().Int("user_id", userId).Int("admin_id", adminUserId).Msg("Successfully retrieved user for admin request")
	// Logging sukses
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "User retrieved successfully", Data: user,
	})
}

// UpdateUser godoc
// @Summary Update user
// @Description Updates an existing user by its ID.
// @Tags Admin - Users Management
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param update_user body models.AdminUpdateUserInput true "User details"
// @Success 200 {object} models.Response "User updated successfully"
// @Failure 400 {object} models.Response "Validation failed or invalid request body"
// @Failure 404 {object} models.Response "User not found"
// @Failure 500 {object} models.Response "Internal server error during user update"
// @Security ApiKeyAuth
// @Router /admin/users/{userId} [patch]
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	// 1. Dapatkan ID user target dari URL
	targetUserIdStr := c.Params("userId")
	targetUserId, err := strconv.Atoi(targetUserIdStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", targetUserIdStr).Msg("Invalid User ID parameter for update")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid User ID parameter",
		})
	}

	// 2. Dapatkan ID admin yang sedang login (opsional, tapi bisa berguna untuk log)
	adminUserId, _ := utils.ExtractUserIDFromJWT(c) // Abaikan error sementara jika hanya untuk log

	// 3. Parse & Validasi Input Body (Gunakan struct input baru)
	input := new(models.AdminUpdateUserInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Error().Err(err).Msg("Error parsing update user request body")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Failed to parse request body",
		})
	}

	// 4. Validasi data input menggunakan validator
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Msg("Update user validation failed")
		// Berikan detail error validasi jika perlu (hati-hati info sensitif)
		errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Validation failed",
			Data:    errorDetails, // Jika menggunakan helper
			// Atau hapus Data:
			// Data: nil,
		})
	}

	ctx := c.Context()

	// 5. Validasi Role ID HANYA JIKA disediakan di input dan bukan 0
	if input.RoleID != 0 { // Hanya validasi jika RoleID diisi (bukan nilai default 0)
		_, errRole := h.RoleRepo.GetRoleByID(ctx, input.RoleID)
		if errRole != nil {
			if errors.Is(errRole, pgx.ErrNoRows) {
				zlog.Warn().Err(errRole).Int("target_user_id", targetUserId).Int("provided_role_id", input.RoleID).Msg("Handler: Invalid Role ID provided during user update")
				return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Role ID provided"}) // Pesan lebih jelas
			}
			zlog.Error().Err(errRole).Int("target_user_id", targetUserId).Int("provided_role_id", input.RoleID).Msg("Handler: Error checking Role ID during user update")
			return c.Status(fiber.StatusInternalServerError).JSON(models.Response{Success: false, Message: "Failed to validate role"})
		}
		// Jika validasi lolos, RoleID di input akan digunakan oleh repo
	}
	// else {
	// Jika RoleID tidak disediakan (atau 0), repo UpdateUserByID TIDAK akan mengubah role user yang ada.
    // Pastikan query UPDATE di repo hanya mengupdate role jika input.RoleID != 0?
    // Tidak perlu, query UPDATE Anda sudah benar akan mengupdate ke nilai RoleID yang ada di `input`,
    // tapi jika RoleID = 0 dan kita tidak validasi, mungkin menyebabkan error FK jika ID 0 tidak ada.
    // Jadi, pengecekan `if input.RoleID != 0` sebelum validasi GetRoleByID sudah cukup.
    // Repo `UpdateUserByID` akan menerima `input` apa adanya.
	// }

	// 6. Panggil repository untuk update user
	err = h.UserRepo.UpdateUserByID(ctx, targetUserId, input)
	if err != nil {
		// Cek apakah error karena user tidak ditemukan
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("target_user_id", targetUserId).Msg("Attempted to update non-existent user")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("User with ID %d not found", targetUserId),
			})
		}
		// Cek apakah error karena unique constraint
		if strings.Contains(err.Error(), "already exists") {
			zlog.Warn().Err(err).Int("target_user_id", targetUserId).Msg("Unique constraint violation during user update by admin")
			errorDetails := utils.FormatValidationErrors(err)           // Gunakan helper (opsional)
			return c.Status(fiber.StatusConflict).JSON(models.Response{ // 409 Conflict
				Success: false,
				Message: "Unique constraint violation during user update by admin",
				Data:    errorDetails, // Jika menggunakan helper
				// Atau hapus Data:
				// Data: nil,
			})
		}

		// Error lain saat update
		zlog.Error().Err(err).Int("target_user_id", targetUserId).Msg("Failed to update user by admin")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to update user",
		})
	}

	// 7. Kirim response sukses
	zlog.Info().Int("admin_id", adminUserId).Int("updated_user_id", targetUserId).Msg("Admin successfully updated user")
	// Pertimbangkan untuk mengembalikan data user yang sudah diupdate (ambil lagi dari DB)
	// atau cukup pesan sukses
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: fmt.Sprintf("User with ID %d updated successfully", targetUserId),
	})
}

// DeleteUser godoc
// @Summary Delete User (Admin)
// @Description Deletes a specific user by ID. Requires Admin role. Admin cannot delete themselves.
// @Tags Admin - Users Management
// @Accept json
// @Produce json
// @Param userId path int true "User ID to delete"
// @Success 200 {object} models.Response "User deleted successfully"
// @Failure 400 {object} models.Response "Invalid User ID parameter"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not Admin or attempting self-delete)"
// @Failure 404 {object} models.Response "User not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /admin/users/{userId} [delete]
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	// 1. Dapatkan ID user yang akan dihapus dari parameter URL
	targetUserIdStr := c.Params("userId") // Sesuaikan nama param dengan route nanti
	targetUserId, err := strconv.Atoi(targetUserIdStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", targetUserIdStr).Msg("Invalid User ID parameter for deletion")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid User ID parameter",
		})
	}

	// 2. Dapatkan ID admin yang sedang login dari JWT (PENTING: untuk mencegah hapus diri sendiri)
	adminUserId, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to extract admin user ID from JWT")
		// Ini seharusnya tidak terjadi jika middleware auth bekerja, tapi handle untuk keamanan
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to identify requesting admin",
		})
	}

	// 3. Validasi: Admin tidak boleh menghapus dirinya sendiri
	if targetUserId == adminUserId {
		zlog.Warn().Int("admin_id", adminUserId).Msg("Admin attempted to delete themselves")
		return c.Status(fiber.StatusForbidden).JSON(models.Response{
			Success: false, Message: "Admin cannot delete their own account",
		})
	}

	// 4. Panggil repository untuk menghapus user
	ctx := c.Context()
	err = h.UserRepo.DeleteUserByID(ctx, targetUserId)
	if err != nil {
		// Cek apakah error karena user tidak ditemukan
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("target_user_id", targetUserId).Msg("Attempted to delete non-existent user")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("User with ID %d not found", targetUserId),
			})
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23502") {
			zlog.Warn().Err(err).Int("target_user_id", targetUserId).Msg("Cannot delete user due to existing references")
			return c.Status(fiber.StatusConflict).JSON(models.Response{ // 409 Conflict
				Success: false,
				Message: "Cannot delete user: User has existing related records (tasks, rewards, points, etc.).",
			})
		}

		// Error lain saat menghapus
		zlog.Error().Err(err).Int("target_user_id", targetUserId).Msg("Failed to delete user")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to delete user",
		})
	}

	// 5. Kirim response sukses
	zlog.Info().Int("admin_id", adminUserId).Int("deleted_user_id", targetUserId).Msg("Admin successfully deleted user")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: fmt.Sprintf("User with ID %d deleted successfully", targetUserId),
	})
}

// -------------------------------------------------------------------------
// Role Management
// -------------------------------------------------------------------------

// CreateRole godoc
// @Summary Create new role
// @Description Creates a new role and returns the ID of the created role.
// @Tags Admin - Roles Management
// @Accept json
// @Produce json
// @Param create_role body models.Role true "Role details"
// @Success 201 {object} models.Response{data=int} "Role created successfully, returns role ID"
// @Failure 400 {object} models.Response "Validation failed or invalid request body"
// @Failure 409 {object} models.Response "Role with same name already exists"
// @Failure 500 {object} models.Response "Internal server error during role creation"
// @Security ApiKeyAuth
// @Router /admin/roles [post]
func (h *AdminHandler) CreateRole(c *fiber.Ctx) error {
	input := new(models.Role) // Role hanya perlu Name saat create

	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Error parsing create role input")
		errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Invalid request body",
			Data:    errorDetails, // Jika menggunakan helper
			// Atau hapus Data:
			// Data: nil,
		})
	}

	// Validasi input Name (gunakan tag validate di models.Role)
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Msg("Create role validation failed")
		errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Validation failed: role name is required",
			Data:    errorDetails, // Jika menggunakan helper
			// Atau hapus Data:
			// Data: nil,
		})
	}

	ctx := c.Context()
	roleID, err := h.RoleRepo.CreateRole(ctx, input)
	if err != nil {
		// Handle error nama sudah ada
		if strings.Contains(err.Error(), "already exists") {
			zlog.Warn().Err(err).Str("role_name", input.Name).Msg("Attempted to create duplicate role name")
			errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false,
				Message: "Role with same name already exists",
				Data:    errorDetails, // Jika menggunakan helper
				// Atau hapus Data:
				// Data: nil,
			})
		}
		// Error lain
		zlog.Error().Err(err).Str("role_name", input.Name).Msg("Failed to create role")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to create role",
		})
	}

	zlog.Info().Int("role_id", roleID).Str("role_name", input.Name).Msg("Role created successfully")
	return c.Status(fiber.StatusCreated).JSON(models.Response{
		Success: true, Message: "Role created successfully", Data: fiber.Map{"role_id": roleID},
	})
}

// GetAllRoles godoc
// @Summary Get all roles
// @Description Retrieves all available roles and their respective IDs.
// @Tags Admin - Roles Management
// @Accept json
// @Produce json
// @Success 200 {object} models.Response{data=[]models.Role} "Roles retrieved successfully"
// @Failure 500 {object} models.Response "Internal server error during role retrieval"
// @Security ApiKeyAuth
// @Router /admin/roles [get]
func (h *AdminHandler) GetAllRoles(c *fiber.Ctx) error {
	ctx := c.Context()
	roles, err := h.RoleRepo.GetAllRoles(ctx)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to get all roles from repository")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to retrieve roles",
		})
	}

	zlog.Info().Int("role_count", len(roles)).Msg("Successfully retrieved all roles")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Roles retrieved successfully", Data: roles,
	})
}

// GetRoleByID godoc
// @Summary Get role by ID
// @Description Retrieves a role by its ID.
// @Tags Admin - Roles Management
// @Accept json
// @Produce json
// @Param roleId path int true "Role ID"
// @Success 200 {object} models.Response{data=models.Role} "Role retrieved successfully"
// @Failure 400 {object} models.Response "Invalid Role ID parameter"
// @Failure 404 {object} models.Response "Role not found"
// @Failure 500 {object} models.Response "Internal server error during role retrieval"
// @Security ApiKeyAuth
// @Router /admin/roles/{roleId} [get]
func (h *AdminHandler) GetRoleByID(c *fiber.Ctx) error {
	roleIDStr := c.Params("roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", roleIDStr).Msg("Invalid Role ID parameter")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid Role ID parameter",
		})
	}

	ctx := c.Context()
	role, err := h.RoleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("role_id", roleID).Msg("Role not found")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("Role with ID %d not found", roleID),
			})
		}
		zlog.Error().Err(err).Int("role_id", roleID).Msg("Failed to get role by ID")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to retrieve role",
		})
	}

	zlog.Info().Int("role_id", roleID).Msg("Role retrieved successfully")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true,
		Message: "Role retrieved successfully",
		Data:    role,
	})
}

// UpdateRole godoc
// @Summary Update role
// @Description Updates an existing role by its ID.
// @Tags Admin - Roles Management
// @Accept json
// @Produce json
// @Param roleId path int true "Role ID"
// @Param update_role body models.Role true "Role details"
// @Success 200 {object} models.Response "Role updated successfully"
// @Failure 400 {object} models.Response "Validation failed or invalid request body"
// @Failure 404 {object} models.Response "Role not found"
// @Failure 500 {object} models.Response "Internal server error during role update"
// @Security ApiKeyAuth
// @Router /admin/roles/{roleId} [patch]
func (h *AdminHandler) UpdateRole(c *fiber.Ctx) error {
	roleIDStr := c.Params("roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", roleIDStr).Msg("Invalid Role ID parameter")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Invalid Role ID parameter",
		})
	}

	input := new(models.Role) // Hanya perlu Name di body
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Error parsing update role input")
		errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Invalid request body",
			Data:    errorDetails, // Jika menggunakan helper
			// Atau hapus Data:
			// Data: nil,
		})
	}

	// Validasi input Name
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("role_id", roleID).Msg("Update role validation failed")
		errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: "Validation failed: role name is required",
			Data:    errorDetails, // Jika menggunakan helper
			// Atau hapus Data:
			// Data: nil,
		})
	}

	// Set ID dari URL dan panggil repo
	input.ID = roleID
	ctx := c.Context()
	err = h.RoleRepo.UpdateRole(ctx, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("role_id", roleID).Msg("Attempted to update non-existent role")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("Role with ID %d not found", roleID),
			})
		}
		if strings.Contains(err.Error(), "already exists") {
			zlog.Warn().Err(err).Int("role_id", roleID).Str("role_name", input.Name).Msg("Role name conflict during update")
			errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false,
				Message: "Role name already exists, please choose a different name",
				Data:    errorDetails, // Jika menggunakan helper
				// Atau hapus Data:
				// Data: nil,
			})
		}
		zlog.Error().Err(err).Int("role_id", roleID).Msg("Failed to update role")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to update role",
		})
	}

	zlog.Info().Int("role_id", roleID).Str("new_name", input.Name).Msg("Role updated successfully")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Role updated successfully",
	})
}

// DeleteRole godoc
// @Summary Delete role
// @Description Deletes an existing role by its ID. Cannot delete base roles (Admin/Child).
// @Tags Admin - Roles Management
// @Accept json
// @Produce json
// @Param roleId path int true "Role ID"
// @Success 200 {object} models.Response "Role deleted successfully"
// @Failure 400 {object} models.Response "Invalid Role ID parameter"
// @Failure 403 {object} models.Response "Cannot delete base roles (Admin/Child)"
// @Failure 404 {object} models.Response "Role not found"
// @Failure 500 {object} models.Response "Internal server error during role deletion"
// @Security ApiKeyAuth
// @Router /admin/roles/{roleId} [delete]
func (h *AdminHandler) DeleteRole(c *fiber.Ctx) error {
	roleIDStr := c.Params("roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", roleIDStr).Msg("Invalid Role ID parameter")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false, Message: "Invalid Role ID parameter",
		})
	}

	// Hindari menghapus role dasar (opsional tapi aman)
	if roleID == 1 || roleID == 2 { // Asumsi ID 1=Admin, 2=Child
		zlog.Warn().Int("role_id", roleID).Msg("Attempted to delete base role")
		return c.Status(fiber.StatusForbidden).JSON(models.Response{
			Success: false, Message: "Cannot delete base roles (Admin/Child)",
		})
	}

	ctx := c.Context()
	err = h.RoleRepo.DeleteRole(ctx, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Int("role_id", roleID).Msg("Attempted to delete non-existent role")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{
				Success: false, Message: fmt.Sprintf("Role with ID %d not found", roleID),
			})
		}
		// Handle error jika role masih digunakan
		if strings.Contains(err.Error(), "still assigned to this role") {
			zlog.Warn().Err(err).Int("role_id", roleID).Msg("Attempted to delete role still in use")
			errorDetails := utils.FormatValidationErrors(err) // Gunakan helper (opsional)
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false,
				Message: "Failed to delete role due to role still in use",
				Data:    errorDetails, // Jika menggunakan helper
				// Atau hapus Data:
				// Data: nil,
			})
		}
		zlog.Error().Err(err).Int("role_id", roleID).Msg("Failed to delete role")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to delete role",
		})
	}

	zlog.Info().Int("role_id", roleID).Msg("Role deleted successfully")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true, Message: "Role deleted successfully",
	})
}
