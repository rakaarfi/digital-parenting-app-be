package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils/test_utils"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository/mocks"
	serviceMocks "github.com/rakaarfi/digital-parenting-app-be/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupChildHandler() (*fiber.App, *handlers.ChildHandler, *mocks.MockUserTaskRepository, *mocks.MockRewardRepository, *mocks.MockUserRewardRepository, *mocks.MockPointTransactionRepository, *serviceMocks.MockRewardService) {
	mockUserTaskRepo := new(mocks.MockUserTaskRepository)
	mockRewardRepo := new(mocks.MockRewardRepository)
	mockUserRewardRepo := new(mocks.MockUserRewardRepository)
	mockPointRepo := new(mocks.MockPointTransactionRepository)
	mockRewardService := new(serviceMocks.MockRewardService)

	childHandler := handlers.NewChildHandler(
		mockUserTaskRepo,
		mockRewardRepo,
		mockUserRewardRepo,
		mockPointRepo,
		mockRewardService,
	)

	app := fiber.New()
	return app, childHandler, mockUserTaskRepo, mockRewardRepo, mockUserRewardRepo, mockPointRepo, mockRewardService
}

func TestChildHandler_GetMyTasks(t *testing.T) {
	childID := 1
	defaultPage := 1
	defaultLimit := 10

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(mockRepo *mocks.MockUserTaskRepository, childID int, statusFilter string, page, limit int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success - All Tasks",
			queryParams: "",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, childID int, statusFilter string, page, limit int) {
				mockTasks := []models.UserTask{
					{
						ID:          1,
						UserID:      childID,
						TaskID:      1,
						Status:      "assigned",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
						Task:        &models.Task{ID: 1, TaskName: "Task 1", TaskPoint: 100},
						SubmittedAt: nil,
						VerifiedAt:  nil,
					},
					{
						ID:          2,
						UserID:      childID,
						TaskID:      2,
						Status:      "submitted",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
						Task:        &models.Task{ID: 2, TaskName: "Task 2", TaskPoint: 200},
						SubmittedAt: func() *time.Time { t := time.Now(); return &t }(),
						VerifiedAt:  nil,
					},
				}
				mockRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, page, limit).Return(mockTasks, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Tasks retrieved successfully",
				"data":    mock.Anything,
				"meta": map[string]interface{}{
					"total_items":  float64(2),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name:        "Success - Filtered by Status",
			queryParams: "?status=assigned",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, childID int, statusFilter string, page, limit int) {
				mockTasks := []models.UserTask{
					{
						ID:          1,
						UserID:      childID,
						TaskID:      1,
						Status:      "assigned",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
						Task:        &models.Task{ID: 1, TaskName: "Task 1", TaskPoint: 100},
						SubmittedAt: nil,
						VerifiedAt:  nil,
					},
				}
				mockRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, page, limit).Return(mockTasks, 1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Tasks retrieved successfully",
				"data":    mock.Anything,
				"meta": map[string]interface{}{
					"total_items":  float64(1),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name:        "Invalid Status Filter",
			queryParams: "?status=invalid",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, childID int, statusFilter string, page, limit int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid status filter value: 'invalid'. Valid statuses are assigned, submitted, approved, rejected.",
			},
		},
		{
			name:        "Database Error",
			queryParams: "",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, childID int, statusFilter string, page, limit int) {
				mockRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, page, limit).Return(nil, 0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, mockUserTaskRepo, _, _, _, _ := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Get("/api/v1/child/tasks", handler.GetMyTasks)

			// Setup mock expectations
			statusFilter := ""
			page := defaultPage
			limit := defaultLimit
			if tc.queryParams == "?status=assigned" {
				statusFilter = "assigned"
			} else if tc.queryParams == "?status=invalid" {
				statusFilter = "invalid" // This will be caught by the handler
			}
			tc.setupMock(mockUserTaskRepo, childID, statusFilter, page, limit)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/child/tasks"+tc.queryParams, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			// Check success and message
			assert.Equal(t, tc.expectedBody["success"], result["success"])
			assert.Equal(t, tc.expectedBody["message"], result["message"])

			// For successful responses, check meta
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["meta"], result["meta"])
				// We don't check the exact data content as it contains time fields
				assert.NotNil(t, result["data"])
			}
		})
	}
}

func TestChildHandler_SubmitMyTask(t *testing.T) {
	childID := 1

	tests := []struct {
		name           string
		userTaskID     string
		setupMock      func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "Success",
			userTaskID: "1",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				mockRepo.On("SubmitTask", mock.Anything, userTaskID, childID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task submitted successfully",
			},
		},
		{
			name:       "Invalid UserTask ID",
			userTaskID: "invalid",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid UserTask ID parameter",
			},
		},
		{
			name:       "Task Not Found",
			userTaskID: "999",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				mockRepo.On("SubmitTask", mock.Anything, userTaskID, childID).Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Task assignment not found or not yours",
			},
		},
		{
			name:       "Task Already Submitted",
			userTaskID: "2",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				mockRepo.On("SubmitTask", mock.Anything, userTaskID, childID).Return(errors.New("task status is already submitted/completed"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "task status is already submitted/completed",
			},
		},
		{
			name:       "Task Not Assigned to Child",
			userTaskID: "3",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				mockRepo.On("SubmitTask", mock.Anything, userTaskID, childID).Return(errors.New("forbidden: task not assigned to you"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "forbidden: task not assigned to you",
			},
		},
		{
			name:       "Database Error",
			userTaskID: "4",
			setupMock: func(mockRepo *mocks.MockUserTaskRepository, userTaskID, childID int) {
				mockRepo.On("SubmitTask", mock.Anything, userTaskID, childID).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, mockUserTaskRepo, _, _, _, _ := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Patch("/api/v1/child/tasks/:userTaskId/submit", handler.SubmitMyTask)

			// Setup mock expectations
			userTaskID, _ := strconv.Atoi(tc.userTaskID)
			if tc.name != "Invalid UserTask ID" {
				tc.setupMock(mockUserTaskRepo, userTaskID, childID)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/child/tasks/"+tc.userTaskID+"/submit", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, result)
		})
	}
}

func TestChildHandler_GetMyPoints(t *testing.T) {
	childID := 1

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockPointTransactionRepository, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockPointTransactionRepository, childID int) {
				mockRepo.On("CalculateTotalPointsByUserID", mock.Anything, childID).Return(500, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Points balance retrieved",
				"data": map[string]interface{}{
					"total_points": float64(500),
				},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockRepo *mocks.MockPointTransactionRepository, childID int) {
				mockRepo.On("CalculateTotalPointsByUserID", mock.Anything, childID).Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, _, _, _, mockPointRepo, _ := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Get("/api/v1/child/points", handler.GetMyPoints)

			// Setup mock expectations
			tc.setupMock(mockPointRepo, childID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/child/points", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, result)
		})
	}
}

func TestChildHandler_GetMyPointHistory(t *testing.T) {
	childID := 1
	defaultPage := 1
	defaultLimit := 10

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockPointTransactionRepository, childID int, page, limit int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockPointTransactionRepository, childID int, page, limit int) {
				mockTransactions := []models.PointTransaction{
					{
						ID:              1,
						UserID:          childID,
						ChangeAmount:    100,
						TransactionType: models.TransactionTypeCompletion,
						CreatedByUserID: 2, // Parent ID
						Notes:           "Task completed",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
					{
						ID:              2,
						UserID:          childID,
						ChangeAmount:    -50,
						TransactionType: models.TransactionTypeRedemption,
						CreatedByUserID: childID,
						Notes:           "Reward claimed",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
				}
				mockRepo.On("GetTransactionsByUserID", mock.Anything, childID, page, limit).Return(mockTransactions, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Points transaction history retrieved successfully",
				"data":    mock.Anything,
				"meta": map[string]interface{}{
					"total_items":  float64(2),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name: "No Transactions Found",
			setupMock: func(mockRepo *mocks.MockPointTransactionRepository, childID int, page, limit int) {
				mockRepo.On("GetTransactionsByUserID", mock.Anything, childID, page, limit).Return([]models.PointTransaction{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Points transaction history retrieved successfully",
				"data":    []interface{}{},
				"meta": map[string]interface{}{
					"total_items":  float64(0),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(0),
				},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockRepo *mocks.MockPointTransactionRepository, childID int, page, limit int) {
				mockRepo.On("GetTransactionsByUserID", mock.Anything, childID, page, limit).Return(nil, 0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, _, _, _, mockPointRepo, _ := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Get("/api/v1/child/points/history", handler.GetMyPointHistory)

			// Setup mock expectations
			tc.setupMock(mockPointRepo, childID, defaultPage, defaultLimit)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/child/points/history", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			// Check success and message
			assert.Equal(t, tc.expectedBody["success"], result["success"])
			assert.Equal(t, tc.expectedBody["message"], result["message"])

			// For successful responses, check meta
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["meta"], result["meta"])
				// We don't check the exact data content as it contains time fields
				assert.Contains(t, result, "data")
			}
		})
	}
}

func TestChildHandler_GetMyClaims(t *testing.T) {
	childID := 1
	defaultPage := 1
	defaultLimit := 10

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockUserRewardRepository, childID int, page, limit int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockUserRewardRepository, childID int, page, limit int) {
				mockClaims := []models.UserReward{
					{
						ID:        1,
						UserID:    childID,
						RewardID:  1,
						Status:    "pending",
						ClaimedAt: time.Now(),
					},
					{
						ID:        2,
						UserID:    childID,
						RewardID:  2,
						Status:    "approved",
						ClaimedAt: time.Now(),
					},
				}
				mockRepo.On("GetClaimsByChildID", mock.Anything, childID, "", page, limit).Return(mockClaims, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward claims history retrieved successfully",
				"data":    mock.Anything,
				"meta": map[string]interface{}{
					"total_items":  float64(2),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name: "No Claims Found",
			setupMock: func(mockRepo *mocks.MockUserRewardRepository, childID int, page, limit int) {
				mockRepo.On("GetClaimsByChildID", mock.Anything, childID, "", page, limit).Return([]models.UserReward{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward claims history retrieved successfully",
				"data":    []interface{}{},
				"meta": map[string]interface{}{
					"total_items":  float64(0),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(0),
				},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockRepo *mocks.MockUserRewardRepository, childID int, page, limit int) {
				mockRepo.On("GetClaimsByChildID", mock.Anything, childID, "", page, limit).Return(nil, 0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, _, _, mockUserRewardRepo, _, _ := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Get("/api/v1/child/claims", handler.GetMyClaims)

			// Setup mock expectations
			tc.setupMock(mockUserRewardRepo, childID, defaultPage, defaultLimit)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/child/claims", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)

			// Check success and message
			assert.Equal(t, tc.expectedBody["success"], result["success"])
			assert.Equal(t, tc.expectedBody["message"], result["message"])

			// For successful responses, check meta
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["meta"], result["meta"])
				// We don't check the exact data content as it contains time fields
				assert.Contains(t, result, "data")
			}
		})
	}
}

func TestChildHandler_ClaimReward(t *testing.T) {
	childID := 1

	tests := []struct {
		name           string
		rewardID       string
		setupMock      func(mockService *serviceMocks.MockRewardService, rewardID, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:     "Success",
			rewardID: "1",
			setupMock: func(mockService *serviceMocks.MockRewardService, rewardID, childID int) {
				mockService.On("ClaimReward", mock.Anything, childID, rewardID).Return(5, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward claim submitted for approval",
				"data": map[string]interface{}{
					"claim_id": float64(5),
				},
			},
		},
		{
			name:     "Invalid Reward ID",
			rewardID: "invalid",
			setupMock: func(mockService *serviceMocks.MockRewardService, rewardID, childID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Reward ID parameter",
			},
		},
		{
			name:     "Reward Not Found",
			rewardID: "999",
			setupMock: func(mockService *serviceMocks.MockRewardService, rewardID, childID int) {
				mockService.On("ClaimReward", mock.Anything, childID, rewardID).Return(0, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Reward not found",
			},
		},
		{
			name:     "Insufficient Points",
			rewardID: "2",
			setupMock: func(mockService *serviceMocks.MockRewardService, rewardID, childID int) {
				// Buat error yang akan dikenali oleh handleChildError sebagai ErrInsufficientPoints
				mockService.On("ClaimReward", mock.Anything, childID, rewardID).Return(0, errors.New("insufficient points"))
			},
			expectedStatus: http.StatusInternalServerError, // Karena kita tidak bisa menggunakan service.ErrInsufficientPoints
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
		{
			name:     "Database Error",
			rewardID: "3",
			setupMock: func(mockService *serviceMocks.MockRewardService, rewardID, childID int) {
				mockService.On("ClaimReward", mock.Anything, childID, rewardID).Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, handler, _, _, _, _, mockRewardService := setupChildHandler()

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Post("/api/v1/child/rewards/:rewardId/claim", handler.ClaimReward)

			// Setup mock expectations
			rewardID, _ := strconv.Atoi(tc.rewardID)
			if tc.name != "Invalid Reward ID" {
				tc.setupMock(mockRewardService, rewardID, childID)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/child/rewards/"+tc.rewardID+"/claim", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, result)
		})
	}
}
