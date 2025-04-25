// internal/api/v1/handlers/child_handler.go
package handlers

import (
	"errors" // Import errors
	"fmt"
	"net/http"
	"strconv"
	"strings" // Import strings

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5" // Import pgx
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service" // Import service
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	"github.com/rs/zerolog/log"
)

type ChildHandler struct {
	// --- Repositories (untuk Get) ---
	UserTaskRepo   repository.UserTaskRepository
	RewardRepo     repository.RewardRepository
	UserRewardRepo repository.UserRewardRepository
	PointRepo      repository.PointTransactionRepository

	// --- Services (untuk operasi dengan logika/transaksi) ---
	RewardService service.RewardService // Untuk ClaimReward
	// TaskService service.TaskService // Belum perlu untuk SubmitTask? Tergantung repo

	// --- Lainnya ---
	// UserRepo repository.UserRepository // Mungkin tidak perlu jika info user dari JWT cukup
	Validate *validator.Validate
}

// Modifikasi constructor
func NewChildHandler(
	userTaskRepo repository.UserTaskRepository,
	rewardRepo repository.RewardRepository,
	userRewardRepo repository.UserRewardRepository,
	pointRepo repository.PointTransactionRepository,
	rewardService service.RewardService, // Inject RewardService
	// taskService service.TaskService,
) *ChildHandler {
	return &ChildHandler{
		UserTaskRepo:   userTaskRepo,
		RewardRepo:     rewardRepo,
		UserRewardRepo: userRewardRepo,
		PointRepo:      pointRepo,
		RewardService:  rewardService, // Simpan RewardService
		// TaskService:  taskService,
		Validate: validator.New(),
	}
}

// Helper function to validate task status (bisa diletakkan di sini atau di utils)
func isValidTaskStatus(status string) bool {
	switch models.UserTaskStatus(status) {
	case models.UserTaskStatusAssigned,
		models.UserTaskStatusSubmitted,
		models.UserTaskStatusApproved,
		models.UserTaskStatusRejected:
		return true
	default:
		return false
	}
}

// Helper function to validate reward claim status (bisa diletakkan di sini atau di utils)
func isValidClaimStatus(status string) bool {
	switch models.UserRewardStatus(status) {
	case models.UserRewardStatusPending,
		models.UserRewardStatusApproved,
		models.UserRewardStatusRejected:
		return true
	default:
		return false
	}
}

// --- Error Handling Helper (bisa dibuat generik untuk semua handler) ---
func handleChildError(c *fiber.Ctx, err error, operation string) error {
	log := log.With().Str("operation", operation).Logger()

	if errors.Is(err, pgx.ErrNoRows) {
		log.Warn().Msg("Resource not found")
		// Sesuaikan pesan berdasarkan operasi jika perlu
		message := "Resource not found"
		if operation == "SubmitMyTask" {
			message = "Task assignment not found or not yours"
		} else if operation == "ClaimReward" {
			message = "Reward not found"
		}
		return c.Status(fiber.StatusNotFound).JSON(models.Response{Success: false, Message: message})
	}
	if errors.Is(err, service.ErrInsufficientPoints) {
		log.Warn().Err(err).Msg("Insufficient points")
		return c.Status(fiber.StatusPaymentRequired).JSON(models.Response{Success: false, Message: err.Error()}) // 402
	}
	// Cek error forbidden/status salah dari repo/service
	if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "not assigned to you") || strings.Contains(err.Error(), "already submitted/completed") || strings.Contains(err.Error(), "task status is already") {
		log.Warn().Err(err).Msg("Forbidden or invalid state")
		return c.Status(fiber.StatusForbidden).JSON(models.Response{Success: false, Message: err.Error()}) // 403
	}

	// Error Internal Server
	log.Error().Err(err).Msg("Internal server error")
	return c.Status(fiber.StatusInternalServerError).JSON(models.Response{Success: false, Message: "An internal error occurred"})
}

// ==========================================================
// --- Task Viewing & Submission ---
// ==========================================================

// GetMyTasks godoc
// @Summary Get My Tasks
// @Description Retrieves tasks assigned to the logged-in child, optionally filtered by status.
// @Tags Child - Tasks
// @Produce json
// @Param status query string false "Filter by status (assigned, submitted, approved, rejected)" Enums(assigned, submitted, approved, rejected)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Tasks retrieved"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/tasks [get]
func (h *ChildHandler) GetMyTasks(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	statusFilter := c.Query("status")
	if statusFilter != "" && !isValidTaskStatus(statusFilter) {
		log.Warn().Str("status_filter", statusFilter).Int("child_id", childID).Msg("Handler: Invalid status filter value provided for GetMyTasks")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: fmt.Sprintf("Invalid status filter value: '%s'. Valid statuses are assigned, submitted, approved, rejected.", statusFilter),
		})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context() // Gunakan context

	tasks, totalCount, err := h.UserTaskRepo.GetTasksByChildID(ctx, childID, statusFilter, pagination.Page, pagination.Limit)
	if err != nil {
		// Gunakan helper error
		return handleChildError(c, err, "GetMyTasks")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Tasks retrieved successfully", tasks, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// SubmitMyTask godoc
// @Summary Submit My Task
// @Description Marks a specific assigned task as 'submitted' by the logged-in child.
// @Tags Child - Tasks
// @Produce json
// @Param userTaskId path int true "UserTask ID (the specific assignment)"
// @Success 200 {object} models.Response "Task submitted successfully"
// @Failure 400 {object} models.Response "Invalid UserTask ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not your task or task not assignable)"
// @Failure 404 {object} models.Response "Task assignment not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/tasks/{userTaskId}/submit [patch]
func (h *ChildHandler) SubmitMyTask(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	userTaskID, err := strconv.Atoi(c.Params("userTaskId"))
	if err != nil {
		log.Warn().Err(err).Str("param", c.Params("userTaskId")).Msg("Handler: Invalid UserTask ID parameter for SubmitMyTask")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid UserTask ID parameter"})
	}

	// Panggil repo untuk submit (Repo sudah cek ownership dan status 'assigned')
	ctx := c.Context()
	err = h.UserTaskRepo.SubmitTask(ctx, userTaskID, childID)
	if err != nil {
		// Gunakan helper error
		return handleChildError(c, err, "SubmitMyTask")
	}

	log.Info().Int("child_id", childID).Int("user_task_id", userTaskID).Msg("Handler: Task submitted successfully by child")
	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Task submitted successfully"})
}

// ==========================================================
// --- Points & Rewards ---
// ==========================================================

// GetMyPoints godoc
// @Summary Get My Points Balance
// @Description Retrieves the current points balance for the logged-in child.
// @Tags Child - Points & Rewards
// @Produce json
// @Success 200 {object} models.Response{data=map[string]int} "Points balance retrieved"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/points [get]
func (h *ChildHandler) GetMyPoints(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	ctx := c.Context()
	totalPoints, err := h.PointRepo.CalculateTotalPointsByUserID(ctx, childID)
	if err != nil {
		// Gunakan helper error
		return handleChildError(c, err, "GetMyPoints")
	}

	log.Info().Int("child_id", childID).Int("total_points", totalPoints).Msg("Handler: Retrieved points balance for child")
	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Points balance retrieved", Data: fiber.Map{"total_points": totalPoints}})
}

// GetAvailableRewards godoc
// @Summary Get Available Rewards
// @Description Retrieves rewards available for the logged-in child to claim.
// @Tags Child - Points & Rewards
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Available rewards retrieved"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/rewards [get]
func (h *ChildHandler) GetAvailableRewards(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	rewards, totalCount, err := h.RewardRepo.GetAvailableRewardsForChild(ctx, childID, pagination.Page, pagination.Limit)
	if err != nil {
		// Gunakan helper error
		return handleChildError(c, err, "GetAvailableRewards")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Available rewards retrieved successfully", rewards, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// ClaimReward godoc
// @Summary Claim a Reward
// @Description Submits a claim request for a specific reward by the logged-in child.
// @Tags Child - Points & Rewards
// @Produce json
// @Param rewardId path int true "Reward ID to claim"
// @Success 201 {object} models.Response{data=map[string]int} "Reward claim submitted"
// @Failure 400 {object} models.Response "Invalid Reward ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 402 {object} models.Response "Insufficient points" // Payment Required (402) bisa dipakai di sini
// @Failure 404 {object} models.Response "Reward not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/rewards/{rewardId}/claim [post]
func (h *ChildHandler) ClaimReward(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	rewardID, err := strconv.Atoi(c.Params("rewardId"))
	if err != nil {
		log.Warn().Err(err).Str("param", c.Params("rewardId")).Msg("Handler: Invalid Reward ID parameter for ClaimReward")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Reward ID parameter"})
	}

	// --- Panggil Service Layer ---
	ctx := c.Context()
	claimID, err := h.RewardService.ClaimReward(ctx, childID, rewardID) // Panggil service
	if err != nil {
		// Gunakan helper error
		return handleChildError(c, err, "ClaimReward")
	}

	// Sukses
	log.Info().Int("child_id", childID).Int("reward_id", rewardID).Int("claim_id", claimID).Msg("Handler: Reward claim submitted successfully via service")
	return c.Status(fiber.StatusCreated).JSON(models.Response{Success: true, Message: "Reward claim submitted for approval", Data: fiber.Map{"claim_id": claimID}})
}

// GetMyClaims godoc
// @Summary Get My Reward Claims History
// @Description Retrieves the history of reward claims made by the logged-in child (paginated).
// @Tags Child - Points & Rewards
// @Produce json
// @Param status query string false "Filter by status (pending, approved, rejected)" Enums(pending, approved, rejected)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Claims history retrieved"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/claims [get]
func (h *ChildHandler) GetMyClaims(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	statusFilter := c.Query("status")
	if statusFilter != "" && !isValidClaimStatus(statusFilter) {
		log.Warn().Str("status_filter", statusFilter).Int("child_id", childID).Msg("Handler: Invalid status filter value provided for GetMyClaims")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{
			Success: false,
			Message: fmt.Sprintf("Invalid status filter value: '%s'. Valid statuses are pending, approved, rejected.", statusFilter),
		})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	claims, totalCount, err := h.UserRewardRepo.GetClaimsByChildID(ctx, childID, statusFilter, pagination.Page, pagination.Limit)
	if err != nil {
		return handleChildError(c, err, "GetMyClaims")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Reward claims history retrieved successfully", claims, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// GetMyPointHistory godoc
// @Summary Get My Points History
// @Description Retrieves the points transaction history for the logged-in child (paginated).
// @Tags Child - Points & Rewards
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Points history retrieved"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /child/points/history [get]
func (h *ChildHandler) GetMyPointHistory(c *fiber.Ctx) error {
	childID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		log.Error().Err(err).Msg("Handler: Failed to extract childID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	transactions, totalCount, err := h.PointRepo.GetTransactionsByUserID(ctx, childID, pagination.Page, pagination.Limit)
	if err != nil {
		return handleChildError(c, err, "GetMyPointHistory")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Points transaction history retrieved successfully", transactions, meta)

	return c.Status(http.StatusOK).JSON(response)
}
