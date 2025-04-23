// internal/api/v1/handlers/parent_handler.go
package handlers

import (
	"errors" // Import errors
	"fmt"

	// Import fmt
	"net/http"
	"strconv"
	"strings" // Import strings

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5" // Import pgx for ErrNoRows
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service" // Import service
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	zlog "github.com/rs/zerolog/log"
)

type ParentHandler struct {
	// --- Repositories (untuk CRUD sederhana) ---
	UserRelRepo    repository.UserRelationshipRepository
	TaskRepo       repository.TaskRepository
	UserTaskRepo   repository.UserTaskRepository // Mungkin masih perlu untuk Get?
	RewardRepo     repository.RewardRepository
	UserRewardRepo repository.UserRewardRepository       // Mungkin masih perlu untuk Get?
	PointRepo      repository.PointTransactionRepository // Mungkin masih perlu untuk Get?
	UserRepo       repository.UserRepository

	// --- Services (untuk logika bisnis & transaksi) ---
	TaskService   service.TaskService
	RewardService service.RewardService
	UserService   service.UserService

	Validate *validator.Validate
}

// Modifikasi constructor untuk menerima service
func NewParentHandler(
	userRelRepo repository.UserRelationshipRepository,
	taskRepo repository.TaskRepository,
	userTaskRepo repository.UserTaskRepository,
	rewardRepo repository.RewardRepository,
	userRewardRepo repository.UserRewardRepository,
	pointRepo repository.PointTransactionRepository,
	userRepo repository.UserRepository,
	taskService service.TaskService, // Terima TaskService
	rewardService service.RewardService, // Terima RewardService
	userService service.UserService,
) *ParentHandler {
	return &ParentHandler{
		UserRelRepo:    userRelRepo,
		TaskRepo:       taskRepo,
		UserTaskRepo:   userTaskRepo,
		RewardRepo:     rewardRepo,
		UserRewardRepo: userRewardRepo,
		PointRepo:      pointRepo,
		UserRepo:       userRepo,
		TaskService:    taskService,   // Simpan TaskService
		RewardService:  rewardService, // Simpan RewardService
		UserService:    userService,
		Validate:       validator.New(),
	}
}

// --- Error Handling Helper ---
func handleParentError(c *fiber.Ctx, err error, operation string) error {
	log := zlog.With().Str("operation", operation).Logger() // Tambahkan konteks operasi ke log

	// Error spesifik service/repo
	if errors.Is(err, pgx.ErrNoRows) {
		log.Warn().Msg("Resource not found or access denied") // Pesan bisa lebih spesifik tergantung konteks operasi
		// Periksa operasi untuk pesan yang lebih baik jika perlu
		message := "Resource not found"
		if operation == "RemoveChild" {
			message = "Relationship not found"
		}
		return c.Status(fiber.StatusNotFound).JSON(models.Response{Success: false, Message: message}) // Kembalikan 404
	}
	if errors.Is(err, service.ErrInsufficientPoints) { // Jika error ini relevan di parent handler
		log.Warn().Err(err).Msg("Insufficient points")
		return c.Status(fiber.StatusPaymentRequired).JSON(models.Response{Success: false, Message: err.Error()})
	}
	// Cek error forbidden (misal, dari service atau validasi repo)
	if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "not authorized") {
		log.Warn().Err(err).Msg("Forbidden access attempt")
		return c.Status(fiber.StatusForbidden).JSON(models.Response{Success: false, Message: "Forbidden: You are not authorized for this action"})
	}
	// Cek error conflict (misal, unique constraint)
	if strings.Contains(err.Error(), "already exists") {
		log.Warn().Err(err).Msg("Conflict detected")
		return c.Status(fiber.StatusConflict).JSON(models.Response{Success: false, Message: err.Error()})
	}
	// Cek error bad request (misal, status tidak valid)
	if strings.Contains(err.Error(), "cannot") || strings.Contains(err.Error(), "invalid") {
		log.Warn().Err(err).Msg("Bad request detected")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: err.Error()})
	}

	// Error Internal Server
	log.Error().Err(err).Msg("Internal server error")
	return c.Status(fiber.StatusInternalServerError).JSON(models.Response{Success: false, Message: "An internal error occurred"})
}

// --- Child Management ---

// GetMyChildren godoc
// @Summary Get My Children
// @Description Retrieves a list of child user accounts associated with the logged-in parent account.
// @Tags Parent - Children
// @Produce json
// @Success 200 {object} models.Response "Children retrieved"
// @Failure 400 {object} models.Response "Invalid query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children [get]
func (h *ParentHandler) GetMyChildren(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		// Error extract JWT biasanya 500 atau 401 tergantung implementasi middleware/utils
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	ctx := c.Context() // Gunakan context
	children, err := h.UserRelRepo.GetChildrenByParentID(ctx, parentID)
	if err != nil {
		// Gunakan helper error
		return handleParentError(c, err, "GetMyChildren")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Children retrieved successfully", Data: children})
}

// AddChild godoc
// @Summary Add Child Relationship
// @Description Associates a child user account with the logged-in parent account using the child's username or email.
// @Tags Parent - Children
// @Accept json
// @Produce json
// @Param add_child_input body models.AddChildInput true "Child Identifier (Username or Email)"
// @Success 201 {object} models.Response "Child relationship added successfully"
// @Failure 400 {object} models.Response "Validation failed, invalid input, or attempting to add self"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 404 {object} models.Response "Child user not found with the provided identifier"
// @Failure 409 {object} models.Response "Relationship already exists or child is not a 'Child' role"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children [post]
func (h *ParentHandler) AddChild(c *fiber.Ctx) error {
	// 1. Dapatkan ID Parent dari JWT
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT for AddChild")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	// 2. Parse & Validasi Input Body
	input := new(models.AddChildInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for AddChild")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Msg("Handler: Validation failed for AddChild input")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	identifier := strings.TrimSpace(input.Identifier) // Bersihkan spasi

	// 3. Cari User Anak berdasarkan Identifier (Username atau Email)
	//    Kita perlu memanggil UserRepository.GetUserByUsername atau metode baru GetUserByIdentifier.
	//    Untuk contoh ini, kita asumsikan GetUserByUsername bisa handle ini (repo mungkin perlu modifikasi jika identifier adalah email)
	//    Atau, lebih baik panggil GetUserByUsername DAN GetUserByEmail jika berbeda.
	ctx := c.Context()
	childUser, err := h.UserRepo.GetUserByUsername(ctx, identifier) // Coba cari by username
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// TODO: Jika Anda ingin mendukung pencarian via email, tambahkan logika pencarian email di sini
			// childUser, err = h.UserRepo.GetUserByEmail(ctx, identifier) // Perlu metode repo baru
			// if errors.Is(err, pgx.ErrNoRows) {
			zlog.Warn().Str("identifier", identifier).Msg("Handler: Child user not found by identifier for AddChild")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{Success: false, Message: "Child user not found with the provided identifier"})
			// } else if err != nil {
			//  	return handleParentError(c, err, "AddChild - Find By Email")
			// }
		} else {
			// Error lain saat mencari user
			return handleParentError(c, err, "AddChild - Find By Username")
		}
	}

	// 4. Validasi Tambahan
	// a. Pastikan user yang ditemukan memiliki role 'Child'
	if childUser.Role == nil || !strings.EqualFold(childUser.Role.Name, "Child") {
		zlog.Warn().Int("found_user_id", childUser.ID).Str("role", childUser.Role.Name).Msg("Handler: Attempted to add a user who is not a Child")
		return c.Status(fiber.StatusConflict).JSON(models.Response{Success: false, Message: "The specified user is not a child account"})
	}
	// b. Pastikan parent tidak menambahkan dirinya sendiri
	if childUser.ID == parentID {
		zlog.Warn().Int("parent_id", parentID).Msg("Handler: Parent attempted to add themselves as a child")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Cannot add yourself as a child"})
	}

	// 5. Tambahkan Relasi (Panggil Repo)
	err = h.UserRelRepo.AddRelationship(ctx, parentID, childUser.ID)
	if err != nil {
		// handleParentError akan menangani conflict (already exists) dan error internal
		return handleParentError(c, err, "AddChild - Add Relationship")
	}

	// Sukses
	zlog.Info().Int("parent_id", parentID).Int("child_id", childUser.ID).Msg("Handler: Child relationship added successfully")
	return c.Status(fiber.StatusCreated).JSON(models.Response{Success: true, Message: "Child relationship added successfully"})
}

// RemoveChild godoc
// @Summary Remove Child Relationship
// @Description Removes the association between the logged-in parent and a specific child user.
// @Tags Parent - Children
// @Produce json
// @Param childId path int true "Child User ID to remove relationship with"
// @Success 200 {object} models.Response "Child relationship removed successfully"
// @Failure 400 {object} models.Response "Invalid Child ID parameter"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 404 {object} models.Response "Relationship not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children/{childId} [delete]
func (h *ParentHandler) RemoveChild(c *fiber.Ctx) error {
	// 1. Dapatkan ID Parent dari JWT
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT for RemoveChild")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	// 2. Dapatkan ID Child dari Parameter URL
	childIDStr := c.Params("childId")
	childID, err := strconv.Atoi(childIDStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", childIDStr).Msg("Handler: Invalid Child ID parameter for RemoveChild")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Child ID parameter"})
	}

	// 3. Validasi: Parent tidak bisa menghapus relasi dengan dirinya sendiri (meskipun tidak mungkin)
	if childID == parentID {
		zlog.Warn().Int("parent_id", parentID).Msg("Handler: Parent attempted to remove relationship with self")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Cannot remove relationship with yourself"})
	}

	// 4. Panggil Repository untuk Menghapus Relasi
	ctx := c.Context()
	err = h.UserRelRepo.RemoveRelationship(ctx, parentID, childID)
	if err != nil {
		// Gunakan helper error, handleParentError sudah bisa menangani pgx.ErrNoRows sebagai 404 Not Found
		return handleParentError(c, err, "RemoveChild")
	}

	// Sukses
	zlog.Info().Int("parent_id", parentID).Int("child_id", childID).Msg("Handler: Child relationship removed successfully")
	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Child relationship removed successfully"})
}

// --- Task Definition Management ---

// CreateTaskDefinition godoc
// @Summary Create a new Task Definition
// @Description Creates a new task definition that can be assigned to children.
// @Tags Parent - Task Management
// @Accept json
// @Produce json
// @Param input body models.CreateTaskInput true "Task Definition Input"
// @Success 201 {object} models.Response{data=map[string]int} "Task definition created successfully"
// @Failure 400 {object} models.Response "Invalid request body"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/tasks [post]
func (h *ParentHandler) CreateTaskDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	// Gunakan struct spesifik untuk input jika field Task lebih banyak dari yg diperlukan
	input := new(models.CreateTaskInput) // Atau struct input khusus
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for CreateTaskDefinition")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}

	// Validasi data task (misal nama tidak kosong, poin > 0)
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Msg("Handler: Validation failed for CreateTaskDefinition")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	// Buat objek models.Task dari input
	task := &models.Task{
		TaskName:        input.TaskName,
		TaskPoint:       input.TaskPoint,
		TaskDescription: input.TaskDescription,
		CreatedByUserID: parentID, // Set creator dari JWT
	}

	ctx := c.Context()
	taskID, err := h.TaskRepo.CreateTask(ctx, task)
	if err != nil {
		// Error FK violation (parentID tidak ada) seharusnya tidak terjadi jika JWT valid
		return handleParentError(c, err, "CreateTaskDefinition")
	}

	return c.Status(fiber.StatusCreated).JSON(models.Response{Success: true, Message: "Task definition created", Data: fiber.Map{"task_id": taskID}})
}

// GetMyTaskDefinitions godoc
// @Summary Get My Task Definitions
// @Description Retrieves a paginated list of task definitions created by the logged-in parent.
// @Tags Parent - Tasks
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Task definitions retrieved"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/tasks [get]
func (h *ParentHandler) GetMyTaskDefinitions(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	tasks, totalCount, err := h.TaskRepo.GetTasksByCreatorID(ctx, parentID, pagination.Page, pagination.Limit)
	if err != nil {
		return handleParentError(c, err, "GetMyTaskDefinitions")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Task definitions retrieved successfully", tasks, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// UpdateMyTaskDefinition godoc
// @Summary Update My Task Definition
// @Description Updates a task definition created by the logged-in parent.
// @Tags Parent - Tasks
// @Accept json
// @Produce json
// @Param taskId path int true "Task Definition ID"
// @Param task_input body models.Task true "Updated Task Details (Name, Point, Description)"
// @Success 200 {object} models.Response "Task definition updated"
// @Failure 400 {object} models.Response "Invalid input or Task ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not the owner of the task)"
// @Failure 404 {object} models.Response "Task definition not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/tasks/{taskId} [patch]
func (h *ParentHandler) UpdateMyTaskDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	taskID, err := strconv.Atoi(c.Params("taskId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Task ID parameter"})
	}

	input := new(models.UpdateTaskInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for UpdateTaskDefinition")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Int("task_id", taskID).Msg("Handler: Validation failed for UpdateTaskDefinition")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	task := &models.Task{
		ID:              taskID, // Set ID dari URL
		TaskName:        input.TaskName,
		TaskPoint:       input.TaskPoint,
		TaskDescription: input.TaskDescription,
	}

	ctx := c.Context()
	err = h.TaskRepo.UpdateTask(ctx, task, parentID) // Repo cek ownership
	if err != nil {
		// handleParentError akan menangani ErrNoRows (bisa jadi not found atau forbidden)
		return handleParentError(c, err, "UpdateMyTaskDefinition")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Task definition updated successfully"})
}

// DeleteMyTaskDefinition godoc
// @Summary Delete My Task Definition
// @Description Deletes a task definition created by the logged-in parent. Fails if task is assigned.
// @Tags Parent - Tasks
// @Produce json
// @Param taskId path int true "Task Definition ID"
// @Success 200 {object} models.Response "Task definition deleted"
// @Failure 400 {object} models.Response "Invalid Task ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not the owner)"
// @Failure 404 {object} models.Response "Task definition not found"
// @Failure 409 {object} models.Response "Conflict (Task is currently assigned)"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/tasks/{taskId} [delete]
func (h *ParentHandler) DeleteMyTaskDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	taskID, err := strconv.Atoi(c.Params("taskId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Task ID parameter"})
	}

	ctx := c.Context()
	err = h.TaskRepo.DeleteTask(ctx, taskID, parentID) // Repo cek ownership & FK
	if err != nil {
		// // Cek error spesifik dari repo untuk FK violation (karena RESTRICT)
		// if strings.Contains(err.Error(), "currently assigned") || strings.Contains(err.Error(), "still referenced") {
		// 	return c.Status(fiber.StatusConflict).JSON(models.Response{Success: false, Message: err.Error()})
		// }
		// handleParentError akan menangani ErrNoRows (not found/forbidden) dan error lain
		return handleParentError(c, err, "DeleteMyTaskDefinition")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Task definition deleted successfully"})
}

// --- Task Assignment & Verification ---

// AssignTaskToChild godoc
// @Summary Assign Task to Child
// @Description Assigns a task definition created by the logged-in parent to a specific child user.
// @Tags Parent - Tasks
// @Produce json
// @Param childId path int true "Child User ID to assign task to"
// @Param task_id body int true "Task Definition ID to assign"
// @Success 200 {object} models.Response "Task assigned to child successfully"
// @Failure 400 {object} models.Response "Invalid Child ID or Task ID parameter"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not the parent)"
// @Failure 404 {object} models.Response "Child user not found or task definition not found"
// @Failure 409 {object} models.Response "Conflict (Task is already assigned to child)"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children/{childId}/tasks [post]
func (h *ParentHandler) AssignTaskToChild(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	childID, err := strconv.Atoi(c.Params("childId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Child ID parameter"})
	}

	// 1. Validasi: Apakah user ini parent dari childId? (Langsung ke Repo)
	ctx := c.Context()
	isParent, err := h.UserRelRepo.IsParentOf(ctx, parentID, childID)
	if err != nil {
		// handleParentError bisa menangani error internal
		return handleParentError(c, err, "AssignTaskToChild - Check Relationship")
	}
	if !isParent {
		return c.Status(fiber.StatusForbidden).JSON(models.Response{Success: false, Message: "You are not authorized to assign tasks to this child"})
	}

	// 2. Parse input (task_id)
	var input struct {
		TaskID int `json:"task_id" validate:"required,gt=0"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	// 3. (Opsional) Validasi TaskID ada dan bisa diakses oleh parent ini?
	//    Jika menggunakan Family Visibility di GetTaskByID, ini bisa dicek
	// _, err = h.TaskRepo.GetTaskByID(ctx, input.TaskID, parentID)
	// if err != nil {
	//     return handleParentError(c, err, "AssignTaskToChild - Check Task") // Handle Not Found / Forbidden
	// }

	// 4. Assign task (Langsung ke Repo)
	userTaskID, err := h.UserTaskRepo.AssignTask(ctx, childID, input.TaskID, parentID)
	if err != nil {
		// Handle FK violation (misal task_id tidak ada)
		if strings.Contains(err.Error(), "invalid user, task, or assigner ID") {
			return c.Status(fiber.StatusNotFound).JSON(models.Response{Success: false, Message: "Task definition not found"})
		}
		return handleParentError(c, err, "AssignTaskToChild - Assign")
	}

	return c.Status(fiber.StatusCreated).JSON(models.Response{Success: true, Message: "Task assigned successfully", Data: fiber.Map{"user_task_id": userTaskID}})
}

// VerifySubmittedTask godoc
// @Summary Verify Submitted Task
// @Description Verifies a task submitted by a child, approving or rejecting it as a parent.
// @Tags Parent - Tasks
// @Produce json
// @Param userTaskId path int true "UserTask ID to verify"
// @Param status body string true "New status for the task (approved or rejected)"
// @Success 200 {object} models.Response "Task verified successfully"
// @Failure 400 {object} models.Response "Invalid UserTask ID or status"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not authorized to verify this task)"
// @Failure 404 {object} models.Response "Task not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/tasks/{userTaskId}/verify [patch]
func (h *ParentHandler) VerifySubmittedTask(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	userTaskID, err := strconv.Atoi(c.Params("userTaskId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid UserTask ID parameter"})
	}

	// 1. Parse input (new status)
	var input struct {
		Status string `json:"status" validate:"required,oneof=approved rejected"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}
	newStatus := models.UserTaskStatus(input.Status)

	// --- Panggil Service Layer ---
	ctx := c.Context()
	err = h.TaskService.VerifyTask(ctx, userTaskID, parentID, newStatus) // Gunakan TaskService
	if err != nil {
		// Gunakan helper untuk tangani error dari service
		return handleParentError(c, err, "VerifySubmittedTask")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Task status updated successfully"})
}

// GetTasksForChild godoc
// @Summary Get Tasks Assigned to a Specific Child
// @Description Retrieves a paginated list of tasks assigned to a specific child by the logged-in parent.
// @Tags Parent - Tasks
// @Produce json
// @Param childId path int true "Child User ID"
// @Param status query string false "Filter by status (assigned, submitted, approved, rejected)" Enums(assigned, submitted, approved, rejected)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Tasks retrieved"
// @Failure 400 {object} models.Response "Invalid Child ID or query parameters"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not parent of this child)"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children/{childId}/tasks [get]
func (h *ParentHandler) GetTasksForChild(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	childID, err := strconv.Atoi(c.Params("childId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Child ID parameter"})
	}

	// Validasi relasi parent-child
	ctx := c.Context()
	isParent, err := h.UserRelRepo.IsParentOf(ctx, parentID, childID)
	if err != nil {
		return handleParentError(c, err, "GetTasksForChild - Check Relationship")
	}
	if !isParent {
		return c.Status(fiber.StatusForbidden).JSON(models.Response{Success: false, Message: "You are not authorized to view tasks for this child"})
	}

	statusFilter := c.Query("status")
	pagination := utils.ParsePaginationParams(c)

	tasks, totalCount, err := h.UserTaskRepo.GetTasksByChildID(ctx, childID, statusFilter, pagination.Page, pagination.Limit)
	if err != nil {
		return handleParentError(c, err, "GetTasksForChild - Get Tasks")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Tasks retrieved successfully", tasks, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// --- Reward Definition Management ---

// CreateRewardDefinition godoc
// @Summary Create Reward Definition
// @Description Creates a new reward template.
// @Tags Parent - Rewards
// @Accept json
// @Produce json
// @Param reward_input body models.Reward true "Reward Details (Name, Point, Description)"
// @Success 201 {object} models.Response{data=map[string]int} "Reward definition created"
// @Failure 400 {object} models.Response "Validation failed"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/rewards [post]
func (h *ParentHandler) CreateRewardDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	input := new(models.CreateRewardInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for CreateRewardDefinition")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}

	if err := h.Validate.Struct(input); err != nil {
		errorDetails := utils.FormatValidationErrors(err)
		zlog.Warn().Err(err).Int("parent_id", parentID).Msg("Handler: Validation failed for CreateRewardDefinition")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	reward := &models.Reward{
		RewardName:        input.RewardName,
		RewardPoint:       input.RewardPoint,
		RewardDescription: input.RewardDescription,
		CreatedByUserID:   parentID, // Set creator dari JWT
	}

	ctx := c.Context()
	rewardID, err := h.RewardRepo.CreateReward(ctx, reward)
	if err != nil {
		return handleParentError(c, err, "CreateRewardDefinition")
	}

	return c.Status(fiber.StatusCreated).JSON(models.Response{Success: true, Message: "Reward definition created", Data: fiber.Map{"reward_id": rewardID}})
}

// GetMyRewardDefinitions godoc
// @Summary Get My Reward Definitions
// @Description Retrieves reward definitions created by the logged-in parent (paginated).
// @Tags Parent - Rewards
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Reward definitions retrieved"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/rewards [get]
func (h *ParentHandler) GetMyRewardDefinitions(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	rewards, totalCount, err := h.RewardRepo.GetRewardsByCreatorID(ctx, parentID, pagination.Page, pagination.Limit)
	if err != nil {
		return handleParentError(c, err, "GetMyRewardDefinitions")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Reward definitions retrieved successfully", rewards, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// UpdateMyRewardDefinition godoc
// @Summary Update My Reward Definition
// @Description Updates a reward definition created by the logged-in parent.
// @Tags Parent - Rewards
// @Accept json
// @Produce json
// @Param rewardId path int true "Reward Definition ID"
// @Param reward_input body models.Reward true "Updated Reward Details"
// @Success 200 {object} models.Response "Reward definition updated"
// @Failure 400 {object} models.Response "Invalid input or Reward ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not the owner)"
// @Failure 404 {object} models.Response "Reward definition not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/rewards/{rewardId} [patch]
func (h *ParentHandler) UpdateMyRewardDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	rewardID, err := strconv.Atoi(c.Params("rewardId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Reward ID parameter"})
	}

	input := new(models.UpdateRewardInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for UpdateMyRewardDefinition")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Int("reward_id", rewardID).Msg("Handler: Validation failed for UpdateMyRewardDefinition")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	reward := &models.Reward{
		ID:                rewardID, // Set ID dari URL
		RewardName:        input.RewardName,
		RewardPoint:       input.RewardPoint,
		RewardDescription: input.RewardDescription,
	}

	ctx := c.Context()
	err = h.RewardRepo.UpdateReward(ctx, reward, parentID)
	if err != nil {
		return handleParentError(c, err, "UpdateMyRewardDefinition")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Reward definition updated successfully"})
}

// DeleteMyRewardDefinition godoc
// @Summary Delete My Reward Definition
// @Description Deletes a reward definition created by the logged-in parent. Fails if reward claimed/pending.
// @Tags Parent - Rewards
// @Produce json
// @Param rewardId path int true "Reward Definition ID"
// @Success 200 {object} models.Response "Reward definition deleted"
// @Failure 400 {object} models.Response "Invalid Reward ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not the owner)"
// @Failure 404 {object} models.Response "Reward definition not found"
// @Failure 409 {object} models.Response "Conflict (Reward has been claimed)"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/rewards/{rewardId} [delete]
func (h *ParentHandler) DeleteMyRewardDefinition(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	rewardID, err := strconv.Atoi(c.Params("rewardId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Reward ID parameter"})
	}

	ctx := c.Context()
	err = h.RewardRepo.DeleteReward(ctx, rewardID, parentID)
	if err != nil {
		if strings.Contains(err.Error(), "claimed or is pending") {
			return c.Status(fiber.StatusConflict).JSON(models.Response{Success: false, Message: err.Error()})
		}
		return handleParentError(c, err, "DeleteMyRewardDefinition")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Reward definition deleted successfully"})
}

// --- Reward Claim Review ---

// ReviewRewardClaim godoc
// @Summary Review Reward Claim
// @Description Approve or reject a reward claim submitted by a child.
// @Tags Parent - Rewards
// @Accept json
// @Produce json
// @Param claimId path int true "UserReward Claim ID"
// @Param review_input body map[string]string true "Review Input (e.g., {\"status\": \"approved\"} or {\"status\": \"rejected\"})"
// @Success 200 {object} models.Response "Claim review successful"
// @Failure 400 {object} models.Response "Invalid input or Claim ID"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not parent or claim not pending)"
// @Failure 404 {object} models.Response "Claim not found"
// @Failure 500 {object} models.Response "Internal server error (e.g., point update failed)"
// @Security ApiKeyAuth
// @Router /parent/claims/{claimId}/review [patch]
func (h *ParentHandler) ReviewRewardClaim(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	claimID, err := strconv.Atoi(c.Params("claimId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Claim ID parameter"})
	}

	var input struct {
		Status string `json:"status" validate:"required,oneof=approved rejected"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}
	newStatus := models.UserRewardStatus(input.Status)

	// --- Panggil Service Layer ---
	ctx := c.Context()
	err = h.RewardService.ReviewClaim(ctx, claimID, parentID, newStatus) // Gunakan RewardService
	if err != nil {
		// Gunakan helper untuk tangani error dari service
		return handleParentError(c, err, "ReviewRewardClaim")
	}

	return c.Status(http.StatusOK).JSON(models.Response{Success: true, Message: "Reward claim reviewed successfully"})
}

// GetPendingClaims godoc
// @Summary Get Pending Reward Claims
// @Description Retrieves pending reward claims from children associated with the logged-in parent (paginated).
// @Tags Parent - Rewards
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utils.PaginatedResponseGeneric "Pending claims retrieved"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/claims/pending [get]
func (h *ParentHandler) GetPendingClaims(c *fiber.Ctx) error {
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	pagination := utils.ParsePaginationParams(c)
	ctx := c.Context()

	claims, totalCount, err := h.UserRewardRepo.GetPendingClaimsByParentID(ctx, parentID, pagination.Page, pagination.Limit)
	if err != nil {
		return handleParentError(c, err, "GetPendingClaims")
	}

	meta := utils.BuildPaginationMeta(totalCount, pagination.Limit, pagination.Page)
	response := utils.NewPaginatedResponse("Pending reward claims retrieved successfully", claims, meta)

	return c.Status(http.StatusOK).JSON(response)
}

// AdjustChildPoints godoc
// @Summary Adjust Child Points Manually
// @Description Allows a parent to manually add or subtract points from a child's balance.
// @Tags Parent - Points
// @Accept json
// @Produce json
// @Param childId path int true "Child User ID"
// @Param adjust_points_input body models.AdjustPointsInput true "Point Adjustment Details (change_amount, notes)"
// @Success 200 {object} models.Response "Points adjusted successfully"
// @Failure 400 {object} models.Response "Invalid input, Child ID, or attempting zero adjustment"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 403 {object} models.Response "Forbidden (Not parent of this child)"
// @Failure 404 {object} models.Response "Child user not found"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children/{childId}/points [post] // Menggunakan POST karena ini membuat resource baru (transaksi poin)
func (h *ParentHandler) AdjustChildPoints(c *fiber.Ctx) error {
	// 1. Dapatkan ID Parent dari JWT
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT for AdjustChildPoints")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	// 2. Dapatkan ID Child dari Parameter URL
	childIDStr := c.Params("childId")
	childID, err := strconv.Atoi(childIDStr)
	if err != nil {
		zlog.Warn().Err(err).Str("param", childIDStr).Msg("Handler: Invalid Child ID parameter for AdjustChildPoints")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid Child ID parameter"})
	}

	// 3. Validasi: Parent tidak bisa adjust poin dirinya sendiri
	if childID == parentID {
		zlog.Warn().Int("parent_id", parentID).Msg("Handler: Parent attempted to adjust own points")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Cannot adjust your own points"})
	}

	// 4. Parse & Validasi Input Body
	input := new(models.AdjustPointsInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for AdjustChildPoints")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Handler: Validation failed for AdjustChildPoints input")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	// 5. Validasi Relasi Parent-Child
	ctx := c.Context()
	isParent, err := h.UserRelRepo.IsParentOf(ctx, parentID, childID)
	if err != nil {
		// Handle error saat cek relasi (error DB)
		return handleParentError(c, err, "AdjustChildPoints - Check Relationship")
	}
	if !isParent {
		zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("Handler: Attempt to adjust points for a non-child user")
		return c.Status(fiber.StatusForbidden).JSON(models.Response{Success: false, Message: "You are not authorized to adjust points for this child"})
	}

	// 6. Buat Objek PointTransaction
	pointTx := &models.PointTransaction{
		UserID:          childID,                                // ID Anak
		ChangeAmount:    input.ChangeAmount,                     // Jumlah perubahan (+/-)
		TransactionType: models.TransactionTypeManualAdjustment, // Tipe transaksi
		CreatedByUserID: parentID,                               // ID Parent yang melakukan adjusment
		Notes:           input.Notes,                            // Alasan penyesuaian
		// Related IDs akan NULL untuk tipe manual
	}

	// 7. Panggil Repository untuk Membuat Transaksi Poin
	// Operasi ini biasanya tidak memerlukan transaksi DB yang kompleks,
	// jadi bisa langsung panggil repo non-Tx.
	// Namun, jika Anda ingin memastikan konsistensi total poin (misal, ada check constraint saldo >= 0
	// yang ingin ditangani di service), operasi ini bisa dipindah ke PointService nanti.
	err = h.PointRepo.CreateTransaction(ctx, pointTx)
	if err != nil {
		// Handle error dari repo (misal, FK violation jika childID tidak ada)
		if strings.Contains(err.Error(), "invalid user") { // Cek pesan error spesifik dari repo
			zlog.Warn().Err(err).Int("child_id", childID).Msg("Handler: Child user not found during point adjustment")
			return c.Status(fiber.StatusNotFound).JSON(models.Response{Success: false, Message: "Child user not found"})
		}
		return handleParentError(c, err, "AdjustChildPoints - Create Transaction")
	}

	// 8. Kirim Respons Sukses
	zlog.Info().Int("parent_id", parentID).Int("child_id", childID).Int("change_amount", input.ChangeAmount).Msg("Handler: Points adjusted manually successfully")
	return c.Status(http.StatusOK).JSON(models.Response{
		Success: true,
		Message: fmt.Sprintf("Points adjusted successfully for child %d by %d", childID, input.ChangeAmount),
	})
}

// CreateChildAccount godoc
// @Summary Create Child Account by Parent
// @Description Creates a new user account with 'Child' role and links it to the logged-in parent.
// @Tags Parent - Children
// @Accept json
// @Produce json
// @Param create_child_input body models.CreateChildInput true "Child Account Details (username, password, email, names)"
// @Success 201 {object} models.Response{data=map[string]int} "Child account created successfully, returns child user ID"
// @Failure 400 {object} models.Response "Validation failed or invalid input"
// @Failure 401 {object} models.Response "Unauthorized"
// @Failure 409 {object} models.Response "Username or Email already exists for the child"
// @Failure 500 {object} models.Response "Internal server error"
// @Security ApiKeyAuth
// @Router /parent/children/create [post] // Endpoint baru
func (h *ParentHandler) CreateChildAccount(c *fiber.Ctx) error {
	// 1. Dapatkan ID Parent dari JWT
	parentID, err := utils.ExtractUserIDFromJWT(c)
	if err != nil {
		zlog.Error().Err(err).Msg("Handler: Failed to extract parentID from JWT for CreateChildAccount")
		return c.Status(fiber.StatusUnauthorized).JSON(models.Response{Success: false, Message: "Unauthorized: Invalid token"})
	}

	// 2. Parse & Validasi Input Body
	input := new(models.CreateChildInput)
	if err := c.BodyParser(input); err != nil {
		zlog.Warn().Err(err).Msg("Handler: Invalid request body for CreateChildAccount")
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Invalid request body"})
	}
	if err := h.Validate.Struct(input); err != nil {
		zlog.Warn().Err(err).Int("parent_id", parentID).Msg("Handler: Validation failed for CreateChildAccount input")
		errorDetails := utils.FormatValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(models.Response{Success: false, Message: "Validation failed", Data: errorDetails})
	}

	// 3. Panggil Service Layer
	ctx := c.Context()
	// Perlu inject UserService ke ParentHandler
	childID, err := h.UserService.CreateChildAccount(ctx, parentID, input) // Panggil service
	if err != nil {
		// Tangani error dari service
		if errors.Is(err, service.ErrUsernameOrEmailExists) {
			return c.Status(fiber.StatusConflict).JSON(models.Response{
				Success: false, Message: service.ErrUsernameOrEmailExists.Error(),
			})
		}
		// Handle error registrasi generik atau internal lainnya
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Handler: Error returned from UserService.CreateChildAccount")
		return c.Status(fiber.StatusInternalServerError).JSON(models.Response{
			Success: false, Message: "Failed to create child account", // Pesan generik
		})
	}

	// Sukses
	zlog.Info().Int("parent_id", parentID).Int("child_id", childID).Msg("Handler: Child account created successfully via service")
	return c.Status(fiber.StatusCreated).JSON(models.Response{
		Success: true,
		Message: "Child account created successfully",
		Data:    fiber.Map{"child_id": childID},
	})
}
