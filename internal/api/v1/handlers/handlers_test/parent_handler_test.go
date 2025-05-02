package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils/test_utils"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository/mocks"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	serviceMocks "github.com/rakaarfi/digital-parenting-app-be/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetMyChildren(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockUserRelationshipRepository, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockUserRelationshipRepository, parentID int) {
				mockChildren := []models.User{
					{ID: 2, Username: "child1", Email: "child1@example.com", RoleID: 2},
					{ID: 3, Username: "child2", Email: "child2@example.com", RoleID: 2},
				}
				mockRepo.On("GetChildrenByParentID", mock.Anything, parentID).Return(mockChildren, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Children retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{"id": float64(2), "username": "child1", "email": "child1@example.com", "role_id": float64(2)},
					map[string]interface{}{"id": float64(3), "username": "child2", "email": "child2@example.com", "role_id": float64(2)},
				},
			},
		},
		{
			name: "No Children Found",
			setupMock: func(mockRepo *mocks.MockUserRelationshipRepository, parentID int) {
				mockRepo.On("GetChildrenByParentID", mock.Anything, parentID).Return([]models.User{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Children retrieved successfully",
				"data":    []interface{}{},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockRepo *mocks.MockUserRelationshipRepository, parentID int) {
				mockRepo.On("GetChildrenByParentID", mock.Anything, parentID).Return(nil, errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Get("/api/v1/parent/children", parentHandler.GetMyChildren)

			// Setup mock expectations
			tc.setupMock(mockUserRelRepo, parentID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/children", nil)

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

			// For successful responses, check data
			if tc.expectedStatus == http.StatusOK {
				expectedData := tc.expectedBody["data"].([]interface{})
				actualData := result["data"].([]interface{})
				assert.Equal(t, len(expectedData), len(actualData))

				// If there are children, check their details
				if len(expectedData) > 0 {
					for i, expected := range expectedData {
						expectedChild := expected.(map[string]interface{})
						actualChild := actualData[i].(map[string]interface{})
						assert.Equal(t, expectedChild["id"], actualChild["id"])
						assert.Equal(t, expectedChild["username"], actualChild["username"])
						assert.Equal(t, expectedChild["email"], actualChild["email"])
						assert.Equal(t, expectedChild["role_id"], actualChild["role_id"])
					}
				}
			}

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
		})
	}
}

func TestAddChild(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		input          models.AddChildInput
		setupMock      func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Add Child by Username",
			input: models.AddChildInput{
				Identifier: "child_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int) {
				childUser := &models.User{
					ID:       2,
					Username: input.Identifier,
					Email:    "child@example.com",
					RoleID:   2,
					Role:     &models.Role{ID: 2, Name: "Child"},
				}
				mockUserRepo.On("GetUserByUsername", mock.Anything, input.Identifier).Return(childUser, nil)
				mockUserRelRepo.On("AddRelationship", mock.Anything, parentID, childUser.ID).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child relationship added successfully",
			},
		},
		{
			name: "Child Not Found",
			input: models.AddChildInput{
				Identifier: "nonexistent",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int) {
				mockUserRepo.On("GetUserByUsername", mock.Anything, input.Identifier).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Child user not found with the provided identifier",
			},
		},
		{
			name: "Invalid Role - Not a Child",
			input: models.AddChildInput{
				Identifier: "parent_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int) {
				foundUser := &models.User{
					ID:       3,
					Username: input.Identifier,
					Email:    "parent@example.com",
					RoleID:   1,
					Role:     &models.Role{ID: 1, Name: "Parent"},
				}
				mockUserRepo.On("GetUserByUsername", mock.Anything, input.Identifier).Return(foundUser, nil)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "The specified user is not a child account",
			},
		},
		{
			name: "Adding Self as Child",
			input: models.AddChildInput{
				Identifier: "parent_self",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int) {
				foundUser := &models.User{
					ID:       parentID, // Same as parent ID
					Username: input.Identifier,
					Email:    "parent@example.com",
					RoleID:   2,
					Role:     &models.Role{ID: 2, Name: "Child"}, // Even if role is Child
				}
				mockUserRepo.On("GetUserByUsername", mock.Anything, input.Identifier).Return(foundUser, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot add yourself as a child",
			},
		},
		{
			name: "Relationship Already Exists",
			input: models.AddChildInput{
				Identifier: "existing_child",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository, input models.AddChildInput, parentID int) {
				childUser := &models.User{
					ID:       4,
					Username: input.Identifier,
					Email:    "existing@example.com",
					RoleID:   2,
					Role:     &models.Role{ID: 2, Name: "Child"},
				}
				mockUserRepo.On("GetUserByUsername", mock.Anything, input.Identifier).Return(childUser, nil)
				mockUserRelRepo.On("AddRelationship", mock.Anything, parentID, childUser.ID).Return(errors.New("relationship already exists"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "relationship already exists",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children", parentHandler.AddChild)

			// Setup mock expectations
			tc.setupMock(mockUserRepo, mockUserRelRepo, tc.input, parentID)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

			// Verify mock expectations
			mockUserRepo.AssertExpectations(t)
			mockUserRelRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveChild(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		childID        string
		setupMock      func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:    "Success",
			childID: "2",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int) {
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, childID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child relationship removed successfully",
			},
		},
		{
			name:    "Relationship Not Found",
			childID: "999",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int) {
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, childID).Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Relationship not found",
			},
		},
		{
			name:    "Invalid Child ID",
			childID: "invalid",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Child ID parameter",
			},
		},
		{
			name:    "Removing Self",
			childID: strconv.Itoa(parentID), // Same as parent ID
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot remove relationship with yourself",
			},
		},
		{
			name:    "Database Error",
			childID: "3",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, childID int, parentID int) {
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, childID).Return(errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Delete("/api/v1/parent/children/:childId", parentHandler.RemoveChild)

			// Setup mock expectations
			childID, _ := strconv.Atoi(tc.childID)
			if tc.name != "Invalid Child ID" {
				tc.setupMock(mockUserRelRepo, childID, parentID)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/parent/children/"+tc.childID, nil)

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

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
		})
	}
}

func TestCreateChildAccount(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		input          models.CreateChildInput
		setupMock      func(mockUserService *serviceMocks.MockUserService, input models.CreateChildInput, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.CreateChildInput{
				Username:  "newchild",
				Password:  "password123",
				Email:     "newchild@example.com",
				FirstName: "New",
				LastName:  "Child",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, input models.CreateChildInput, parentID int) {
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, &input).Return(5, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child account created successfully",
				"data":    map[string]interface{}{"child_id": float64(5)},
			},
		},
		{
			name: "Validation Error - Missing Required Fields",
			input: models.CreateChildInput{
				Username: "newchild",
				// Missing password and other required fields
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, input models.CreateChildInput, parentID int) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Username/Email Already Exists",
			input: models.CreateChildInput{
				Username:  "existingchild",
				Password:  "password123",
				Email:     "existing@example.com",
				FirstName: "Existing",
				LastName:  "Child",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, input models.CreateChildInput, parentID int) {
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, &input).Return(0, service.ErrUsernameOrEmailExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrUsernameOrEmailExists.Error(),
			},
		},
		{
			name: "Internal Server Error",
			input: models.CreateChildInput{
				Username:  "newchild",
				Password:  "password123",
				Email:     "newchild@example.com",
				FirstName: "New",
				LastName:  "Child",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, input models.CreateChildInput, parentID int) {
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, &input).Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to create child account",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children/create", parentHandler.CreateChildAccount)

			// Setup mock expectations
			tc.setupMock(mockUserService, tc.input, parentID)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children/create", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Validation Error - Missing Required Fields" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserService.AssertExpectations(t)
		})
	}
}

func TestParentHandler_GetTasksForChild(t *testing.T) {
	parentID := 1
	childID := 2

	tests := []struct {
		name           string
		childIDParam   string
		statusFilter   string
		setupMock      func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "Success - All Tasks",
			childIDParam: "2",
			statusFilter: "",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil).Once()

				// Mock GetTasksByChildID
				mockTasks := []models.UserTask{
					{
						ID:               1,
						UserID:           childID,
						TaskID:           1,
						Status:           "assigned",
						AssignedByUserID: parentID,
						Task: &models.Task{
							ID:              1,
							TaskName:        "Task 1",
							TaskPoint:       100,
							TaskDescription: "Description 1",
							CreatedByUserID: parentID,
						},
					},
					{
						ID:               2,
						UserID:           childID,
						TaskID:           2,
						Status:           "submitted",
						AssignedByUserID: parentID,
						Task: &models.Task{
							ID:              2,
							TaskName:        "Task 2",
							TaskPoint:       200,
							TaskDescription: "Description 2",
							CreatedByUserID: parentID,
						},
					},
				}
				mockUserTaskRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, 1, 10).Return(mockTasks, 2, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Tasks retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":                  float64(1),
						"user_id":             float64(childID),
						"task_id":             float64(1),
						"status":              "assigned",
						"assigned_by_user_id": float64(parentID),
						"task": map[string]interface{}{
							"id":                 float64(1),
							"task_name":          "Task 1",
							"task_point":         float64(100),
							"task_description":   "Description 1",
							"created_by_user_id": float64(parentID),
						},
					},
					map[string]interface{}{
						"id":                  float64(2),
						"user_id":             float64(childID),
						"task_id":             float64(2),
						"status":              "submitted",
						"assigned_by_user_id": float64(parentID),
						"task": map[string]interface{}{
							"id":                 float64(2),
							"task_name":          "Task 2",
							"task_point":         float64(200),
							"task_description":   "Description 2",
							"created_by_user_id": float64(parentID),
						},
					},
				},
				"meta": map[string]interface{}{
					"total_items":  float64(2),
					"current_page": float64(1),
					"per_page":     float64(10),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name:         "Success - Filtered by Status",
			childIDParam: "2",
			statusFilter: "submitted",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil).Once()

				// Mock GetTasksByChildID with status filter
				mockTasks := []models.UserTask{
					{
						ID:               2,
						UserID:           childID,
						TaskID:           2,
						Status:           "submitted",
						AssignedByUserID: parentID,
						Task: &models.Task{
							ID:              2,
							TaskName:        "Task 2",
							TaskPoint:       200,
							TaskDescription: "Description 2",
							CreatedByUserID: parentID,
						},
					},
				}
				mockUserTaskRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, 1, 10).Return(mockTasks, 1, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Tasks retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":                  float64(2),
						"user_id":             float64(childID),
						"task_id":             float64(2),
						"status":              "submitted",
						"assigned_by_user_id": float64(parentID),
						"task": map[string]interface{}{
							"id":                 float64(2),
							"task_name":          "Task 2",
							"task_point":         float64(200),
							"task_description":   "Description 2",
							"created_by_user_id": float64(parentID),
						},
					},
				},
				"meta": map[string]interface{}{
					"total_items":  float64(1),
					"current_page": float64(1),
					"per_page":     float64(10),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name:         "Invalid Child ID",
			childIDParam: "invalid",
			statusFilter: "",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Child ID parameter",
			},
		},
		{
			name:         "Not Parent of Child",
			childIDParam: "3",
			statusFilter: "",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// Mock IsParentOf returning false
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, 3).Return(false, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "You are not authorized to view tasks for this child",
			},
		},
		{
			name:         "Database Error - IsParentOf",
			childIDParam: "2",
			statusFilter: "",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// Mock IsParentOf returning error
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(false, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
		{
			name:         "Database Error - GetTasksByChildID",
			childIDParam: "2",
			statusFilter: "",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID int, statusFilter string) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil).Once()

				// Mock GetTasksByChildID returning error
				mockUserTaskRepo.On("GetTasksByChildID", mock.Anything, childID, statusFilter, 1, 10).Return(nil, 0, errors.New("database error")).Once()
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRelRepo:  mockUserRelRepo,
				UserTaskRepo: mockUserTaskRepo,
				Validate:     validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Get("/api/v1/parent/children/:childId/tasks", parentHandler.GetTasksForChild)

			// Setup mock expectations
			childIDInt := childID
			if tc.name == "Invalid Child ID" {
				// Don't set childIDInt for invalid case
			} else if tc.name == "Not Parent of Child" {
				childIDInt = 3
			}
			tc.setupMock(mockUserRelRepo, mockUserTaskRepo, parentID, childIDInt, tc.statusFilter)

			// Prepare request
			url := fmt.Sprintf("/api/v1/parent/children/%s/tasks", tc.childIDParam)
			if tc.statusFilter != "" {
				url += fmt.Sprintf("?status=%s", tc.statusFilter)
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

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

			// For successful responses, check data and meta
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Equal(t, tc.expectedBody["meta"], result["meta"])

				// For the data field, we need to check the length and content
				expectedData := tc.expectedBody["data"].([]interface{})
				actualData := result["data"].([]interface{})
				assert.Equal(t, len(expectedData), len(actualData))

				// If there are tasks, check their details
				if len(expectedData) > 0 {
					for i, expected := range expectedData {
						expectedTask := expected.(map[string]interface{})
						actualTask := actualData[i].(map[string]interface{})
						assert.Equal(t, expectedTask["id"], actualTask["id"])
						assert.Equal(t, expectedTask["user_id"], actualTask["user_id"])
						assert.Equal(t, expectedTask["task_id"], actualTask["task_id"])
						assert.Equal(t, expectedTask["status"], actualTask["status"])
						assert.Equal(t, expectedTask["assigned_by_user_id"], actualTask["assigned_by_user_id"])

						// Check task details
						expectedTaskDetails := expectedTask["task"].(map[string]interface{})
						actualTaskDetails := actualTask["task"].(map[string]interface{})
						assert.Equal(t, expectedTaskDetails["id"], actualTaskDetails["id"])
						assert.Equal(t, expectedTaskDetails["task_name"], actualTaskDetails["task_name"])
						assert.Equal(t, expectedTaskDetails["task_point"], actualTaskDetails["task_point"])
						assert.Equal(t, expectedTaskDetails["task_description"], actualTaskDetails["task_description"])
						assert.Equal(t, expectedTaskDetails["created_by_user_id"], actualTaskDetails["created_by_user_id"])
					}
				}
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
			mockUserTaskRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_AssignTaskToChild(t *testing.T) {
	parentID := 1
	childID := 2
	taskID := 3

	tests := []struct {
		name           string
		input          models.AssignTaskInput
		setupMock      func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.AssignTaskInput{
				TaskID: taskID,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// No need to mock HasSharedChild for this case since taskDefinition.CreatedByUserID == parentID

				// Mock GetTaskByID
				mockTaskRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Test Task",
					TaskPoint:       100,
					TaskDescription: "Test Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock CheckExistingActiveTask
				mockUserTaskRepo.On("CheckExistingActiveTask", mock.Anything, childID, taskID).Return(false, nil)

				// Mock AssignTask
				mockUserTaskRepo.On("AssignTask", mock.Anything, childID, taskID, parentID).Return(5, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task assigned successfully",
				"data":    map[string]interface{}{"user_task_id": float64(5)},
			},
		},
		{
			name: "Validation Error - Missing Required Fields",
			input: models.AssignTaskInput{
				// Missing required fields (we'll use an empty struct)
				TaskID: 0, // Invalid task ID
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf (needed because validation happens after checking parent-child relationship)
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Not Parent of Child",
			input: models.AssignTaskInput{
				TaskID: taskID,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf returning false
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(false, nil)

				// No need to mock HasSharedChild as the handler should return early after IsParentOf check
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "You are not authorized to assign tasks to this child",
			},
		},
		{
			name: "Task Not Found",
			input: models.AssignTaskInput{
				TaskID: 999,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// No need to mock HasSharedChild as the handler should return early after GetTaskByID check

				// Mock GetTaskByID returning not found
				mockTaskRepo.On("GetTaskByID", mock.Anything, 999).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Task definition not found",
			},
		},
		{
			name: "Task Not Owned by Parent",
			input: models.AssignTaskInput{
				TaskID: taskID,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// Mock HasSharedChild (not needed in this case but the handler checks it)
				mockUserRelRepo.On("HasSharedChild", mock.Anything, parentID, parentID+1).Return(false, nil)

				// Mock GetTaskByID returning task owned by different parent
				mockTaskRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Test Task",
					TaskPoint:       100,
					TaskDescription: "Test Description",
					CreatedByUserID: parentID + 1, // Different parent
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You do not have permission to assign this specific task definition.",
			},
		},
		{
			name: "Database Error - IsParentOf",
			input: models.AssignTaskInput{
				TaskID: taskID,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf with error
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
		{
			name: "Database Error - AssignTask",
			input: models.AssignTaskInput{
				TaskID: taskID,
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockTaskRepo *mocks.MockTaskRepository, mockUserTaskRepo *mocks.MockUserTaskRepository, parentID, childID, taskID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// No need to mock HasSharedChild for this case since taskDefinition.CreatedByUserID == parentID

				// Mock GetTaskByID
				mockTaskRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Test Task",
					TaskPoint:       100,
					TaskDescription: "Test Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock CheckExistingActiveTask
				mockUserTaskRepo.On("CheckExistingActiveTask", mock.Anything, childID, taskID).Return(false, nil)

				// Mock AssignTask with error
				mockUserTaskRepo.On("AssignTask", mock.Anything, childID, taskID, parentID).Return(0, errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRelRepo:  mockUserRelRepo,
				TaskRepo:     mockTaskRepo,
				UserTaskRepo: mockUserTaskRepo,
				Validate:     validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children/:childId/tasks", parentHandler.AssignTaskToChild)

			// Setup mock expectations
			tc.setupMock(mockUserRelRepo, mockTaskRepo, mockUserTaskRepo, parentID, childID, taskID)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/parent/children/%d/tasks", childID), bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Validation Error - Missing Required Fields" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
			mockTaskRepo.AssertExpectations(t)
			mockUserTaskRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_VerifySubmittedTask(t *testing.T) {
	parentID := 1
	userTaskID := 5

	tests := []struct {
		name           string
		input          models.VerifyTaskInput
		setupMock      func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Approve Task",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask with approved status
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusApproved).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task status updated successfully",
			},
		},
		{
			name: "Success - Reject Task",
			input: models.VerifyTaskInput{
				Status: "rejected",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask with rejected status
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusRejected).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task status updated successfully",
			},
		},
		{
			name: "Invalid UserTask ID",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid UserTask ID parameter",
			},
		},
		{
			name: "Invalid Status Value",
			input: models.VerifyTaskInput{
				Status: "invalid_status",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Task Not Found",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask returning not found error
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusApproved).Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Resource not found",
			},
		},
		{
			name: "Not Parent of Child",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask returning forbidden error
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusApproved).Return(errors.New("forbidden: you are not authorized to verify this task"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You are not authorized for this action",
			},
		},
		{
			name: "Task Not in Submitted State",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask returning state error
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusApproved).Return(errors.New("cannot verify task: current status is 'assigned', expected 'submitted'"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "cannot verify task: current status is 'assigned', expected 'submitted'",
			},
		},
		{
			name: "Database Error",
			input: models.VerifyTaskInput{
				Status: "approved",
			},
			setupMock: func(mockTaskService *serviceMocks.MockTaskService, userTaskID, parentID int) {
				// Mock TaskService.VerifyTask returning database error
				mockTaskService.On("VerifyTask", mock.Anything, userTaskID, parentID, models.UserTaskStatusApproved).Return(errors.New("internal server error: could not update task status"))
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
			// Setup
			app := fiber.New()
			mockTaskService := new(serviceMocks.MockTaskService)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				TaskService: mockTaskService,
				Validate:    validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Patch("/api/v1/parent/tasks/:userTaskId/verify", parentHandler.VerifySubmittedTask)

			// Setup mock expectations
			if tc.name != "Invalid UserTask ID" {
				tc.setupMock(mockTaskService, userTaskID, parentID)
			}

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			var req *http.Request

			if tc.name == "Invalid UserTask ID" {
				// Use invalid userTaskId in URL
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/parent/tasks/invalid/verify", bytes.NewReader(bodyBytes))
			} else {
				// Use valid userTaskId in URL
				req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/parent/tasks/%d/verify", userTaskID), bytes.NewReader(bodyBytes))
			}

			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Invalid Status Value" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockTaskService.AssertExpectations(t)
		})
	}
}

func TestParentHandler_GetPendingClaims(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		setupMock      func(mockUserRewardRepo *mocks.MockUserRewardRepository)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - With Claims",
			setupMock: func(mockUserRewardRepo *mocks.MockUserRewardRepository) {
				// Mock GetPendingClaimsByParentID
				mockUserRewardRepo.On("GetPendingClaimsByParentID", mock.Anything, parentID, 1, 10).Return([]models.UserReward{
					{
						ID:        1,
						UserID:    2,
						RewardID:  3,
						Status:    models.UserRewardStatusPending,
						ClaimedAt: time.Now(),
					},
				}, 1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Pending reward claims retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":        float64(1),
						"user_id":   float64(2),
						"reward_id": float64(3),
						"status":    "pending",
						// Other fields will be present but we don't need to check them all
					},
				},
				"meta": map[string]interface{}{
					"total":       float64(1),
					"limit":       float64(10),
					"page":        float64(1),
					"total_pages": float64(1),
				},
			},
		},
		{
			name: "Success - No Claims",
			setupMock: func(mockUserRewardRepo *mocks.MockUserRewardRepository) {
				// Mock GetPendingClaimsByParentID with empty result
				mockUserRewardRepo.On("GetPendingClaimsByParentID", mock.Anything, parentID, 1, 10).Return([]models.UserReward{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Pending reward claims retrieved successfully",
				"data":    []interface{}{},
				"meta": map[string]interface{}{
					"total":       float64(0),
					"limit":       float64(10),
					"page":        float64(1),
					"total_pages": float64(0),
				},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockUserRewardRepo *mocks.MockUserRewardRepository) {
				// Mock GetPendingClaimsByParentID with error
				mockUserRewardRepo.On("GetPendingClaimsByParentID", mock.Anything, parentID, 1, 10).Return(nil, 0, errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRewardRepo: mockUserRewardRepo,
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Get("/api/v1/parent/claims/pending", parentHandler.GetPendingClaims)

			// Setup mock expectations
			tc.setupMock(mockUserRewardRepo)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/claims/pending", nil)

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

			if tc.name == "Success - With Claims" || tc.name == "Success - No Claims" {
				// For success cases, check structure but not exact content of data
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data")
				assert.Contains(t, result, "meta")

				// Check meta exists
				assert.Contains(t, result, "meta")

				// For "Success - With Claims", check that data is not empty
				if tc.name == "Success - With Claims" {
					data := result["data"].([]interface{})
					assert.NotEmpty(t, data)

					// Check first item has expected structure
					firstItem := data[0].(map[string]interface{})
					assert.Contains(t, firstItem, "id")
					assert.Contains(t, firstItem, "user_id")
					assert.Contains(t, firstItem, "reward_id")
					assert.Contains(t, firstItem, "status")
					assert.Equal(t, "pending", firstItem["status"])
				} else {
					// For "Success - No Claims", check that data is empty
					data := result["data"].([]interface{})
					assert.Empty(t, data)
				}
			} else {
				// For error cases, check exact match
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserRewardRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_ReviewRewardClaim(t *testing.T) {
	parentID := 1
	claimID := 5

	tests := []struct {
		name           string
		input          models.ReviewClaimInput
		setupMock      func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Approve Claim",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim with approved status
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward claim reviewed successfully",
			},
		},
		{
			name: "Success - Reject Claim",
			input: models.ReviewClaimInput{
				Status: "rejected",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim with rejected status
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusRejected).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward claim reviewed successfully",
			},
		},
		{
			name: "Invalid Claim ID",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Claim ID parameter",
			},
		},
		{
			name: "Invalid Status Value",
			input: models.ReviewClaimInput{
				Status: "invalid_status",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Claim Not Found",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim returning not found error
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Resource not found",
			},
		},
		{
			name: "Not Parent of Child",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim returning forbidden error
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(errors.New("forbidden: you are not authorized to review this claim"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You are not authorized for this action",
			},
		},
		{
			name: "Claim Not in Pending State",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim returning state error
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(errors.New("cannot review claim: current status is 'approved', expected 'pending'"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "cannot review claim: current status is 'approved', expected 'pending'",
			},
		},
		{
			name: "Insufficient Points",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim returning insufficient points error
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(errors.New("insufficient points: child does not have enough points for this reward"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
		{
			name: "Database Error",
			input: models.ReviewClaimInput{
				Status: "approved",
			},
			setupMock: func(mockRewardService *serviceMocks.MockRewardService, claimID, parentID int) {
				// Mock RewardService.ReviewClaim returning database error
				mockRewardService.On("ReviewClaim", mock.Anything, claimID, parentID, models.UserRewardStatusApproved).Return(errors.New("internal server error: could not update claim status"))
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
			// Setup
			app := fiber.New()
			mockRewardService := new(serviceMocks.MockRewardService)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				RewardService: mockRewardService,
				Validate:      validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Patch("/api/v1/parent/claims/:claimId/review", parentHandler.ReviewRewardClaim)

			// Setup mock expectations
			if tc.name != "Invalid Claim ID" {
				tc.setupMock(mockRewardService, claimID, parentID)
			}

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			var req *http.Request

			if tc.name == "Invalid Claim ID" {
				// Use invalid claimId in URL
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/parent/claims/invalid/review", bytes.NewReader(bodyBytes))
			} else {
				// Use valid claimId in URL
				req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/parent/claims/%d/review", claimID), bytes.NewReader(bodyBytes))
			}

			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Invalid Status Value" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockRewardService.AssertExpectations(t)
		})
	}
}

func TestParentHandler_AdjustChildPoints(t *testing.T) {
	parentID := 1
	childID := 2

	tests := []struct {
		name           string
		input          models.AdjustPointsInput
		setupMock      func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Add Points",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "Bonus points for good behavior",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// Mock CreateTransaction
				mockPointRepo.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx *models.PointTransaction) bool {
					return tx.UserID == childID &&
						tx.ChangeAmount == 100 &&
						tx.TransactionType == models.TransactionTypeManualAdjustment &&
						tx.Notes == "Bonus points for good behavior" &&
						tx.CreatedByUserID == parentID
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Points adjusted successfully for child 2 by 100",
			},
		},
		{
			name: "Success - Subtract Points",
			input: models.AdjustPointsInput{
				ChangeAmount: -50,
				Notes:        "Penalty for not cleaning room",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// Mock CreateTransaction
				mockPointRepo.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx *models.PointTransaction) bool {
					return tx.UserID == childID &&
						tx.ChangeAmount == -50 &&
						tx.TransactionType == models.TransactionTypeManualAdjustment &&
						tx.Notes == "Penalty for not cleaning room" &&
						tx.CreatedByUserID == parentID
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Points adjusted successfully for child 2 by -50",
			},
		},
		{
			name: "Invalid Child ID",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "Test notes",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Child ID parameter",
			},
		},
		{
			name: "Zero Change Amount",
			input: models.AdjustPointsInput{
				ChangeAmount: 0,
				Notes:        "Test notes",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Missing Notes",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "", // Empty notes
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Not Parent of Child",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "Test notes",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// Mock IsParentOf returning false
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(false, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "You are not authorized to adjust points for this child",
			},
		},
		{
			name: "Child Not Found",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "Test notes",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// Mock CreateTransaction with "invalid user" error
				mockPointRepo.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx *models.PointTransaction) bool {
					return tx.UserID == childID && tx.ChangeAmount == 100
				})).Return(errors.New("invalid user"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Child user not found",
			},
		},
		{
			name: "Database Error",
			input: models.AdjustPointsInput{
				ChangeAmount: 100,
				Notes:        "Test notes",
			},
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, mockPointRepo *mocks.MockPointTransactionRepository, parentID, childID int) {
				// Mock IsParentOf
				mockUserRelRepo.On("IsParentOf", mock.Anything, parentID, childID).Return(true, nil)

				// Mock CreateTransaction with database error
				mockPointRepo.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx *models.PointTransaction) bool {
					return tx.UserID == childID && tx.ChangeAmount == 100
				})).Return(errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRelRepo: mockUserRelRepo,
				PointRepo:   mockPointRepo,
				Validate:    validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children/:childId/points", parentHandler.AdjustChildPoints)

			// Setup mock expectations
			if tc.name != "Invalid Child ID" {
				tc.setupMock(mockUserRelRepo, mockPointRepo, parentID, childID)
			}

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			var req *http.Request

			if tc.name == "Invalid Child ID" {
				// Use invalid childId in URL
				req = httptest.NewRequest(http.MethodPost, "/api/v1/parent/children/invalid/points", bytes.NewReader(bodyBytes))
			} else {
				// Use valid childId in URL
				req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/parent/children/%d/points", childID), bytes.NewReader(bodyBytes))
			}

			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Zero Change Amount" || tc.name == "Missing Notes" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
			mockPointRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_AddChild(t *testing.T) {
	parentID := 1
	childID := 2

	tests := []struct {
		name           string
		input          models.AddChildInput
		setupMock      func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Add Child by Username",
			input: models.AddChildInput{
				Identifier: "child_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername
				mockUserRepo.On("GetUserByUsername", mock.Anything, "child_user").Return(&models.User{
					ID:       childID,
					Username: "child_user",
					Role:     &models.Role{Name: "Child"},
				}, nil)

				// Mock AddRelationship
				mockUserRelRepo.On("AddRelationship", mock.Anything, parentID, childID).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child relationship added successfully",
			},
		},
		{
			name: "Missing Identifier",
			input: models.AddChildInput{
				Identifier: "",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Child Not Found",
			input: models.AddChildInput{
				Identifier: "nonexistent_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername returning not found
				mockUserRepo.On("GetUserByUsername", mock.Anything, "nonexistent_user").Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Child user not found with the provided identifier",
			},
		},
		{
			name: "User Not a Child",
			input: models.AddChildInput{
				Identifier: "parent_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername returning a parent user
				mockUserRepo.On("GetUserByUsername", mock.Anything, "parent_user").Return(&models.User{
					ID:       3,
					Username: "parent_user",
					Role:     &models.Role{Name: "Parent"},
				}, nil)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "The specified user is not a child account",
			},
		},
		{
			name: "Adding Self as Child",
			input: models.AddChildInput{
				Identifier: "self_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername returning the parent's own user
				mockUserRepo.On("GetUserByUsername", mock.Anything, "self_user").Return(&models.User{
					ID:       parentID, // Same as parent ID
					Username: "self_user",
					Role:     &models.Role{Name: "Child"}, // Even if role is Child
				}, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot add yourself as a child",
			},
		},
		{
			name: "Relationship Already Exists",
			input: models.AddChildInput{
				Identifier: "existing_child",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername
				mockUserRepo.On("GetUserByUsername", mock.Anything, "existing_child").Return(&models.User{
					ID:       childID,
					Username: "existing_child",
					Role:     &models.Role{Name: "Child"},
				}, nil)

				// Mock AddRelationship returning relationship already exists error
				mockUserRelRepo.On("AddRelationship", mock.Anything, parentID, childID).Return(errors.New("relationship already exists"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "relationship already exists",
			},
		},
		{
			name: "Database Error",
			input: models.AddChildInput{
				Identifier: "child_user",
			},
			setupMock: func(mockUserRepo *mocks.MockUserRepository, mockUserRelRepo *mocks.MockUserRelationshipRepository) {
				// Mock GetUserByUsername
				mockUserRepo.On("GetUserByUsername", mock.Anything, "child_user").Return(&models.User{
					ID:       childID,
					Username: "child_user",
					Role:     &models.Role{Name: "Child"},
				}, nil)

				// Mock AddRelationship with database error
				mockUserRelRepo.On("AddRelationship", mock.Anything, parentID, childID).Return(errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRepo := new(mocks.MockUserRepository)
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRepo:    mockUserRepo,
				UserRelRepo: mockUserRelRepo,
				Validate:    validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children", parentHandler.AddChild)

			// Setup mock expectations
			tc.setupMock(mockUserRepo, mockUserRelRepo)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Missing Identifier" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserRepo.AssertExpectations(t)
			mockUserRelRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_RemoveChild(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		childIDParam   string
		setupMock      func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "Success",
			childIDParam: "2",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int) {
				// Mock RemoveRelationship
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, childID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child relationship removed successfully",
			},
		},
		{
			name:         "Invalid Child ID",
			childIDParam: "invalid",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Child ID parameter",
			},
		},
		{
			name:         "Attempting to Remove Self",
			childIDParam: "1", // Same as parentID
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot remove relationship with yourself",
			},
		},
		{
			name:         "Relationship Not Found",
			childIDParam: "3",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int) {
				// Mock RemoveRelationship returning not found
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, 3).Return(pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Relationship not found",
			},
		},
		{
			name:         "Database Error",
			childIDParam: "2",
			setupMock: func(mockUserRelRepo *mocks.MockUserRelationshipRepository, parentID, childID int) {
				// Mock RemoveRelationship with database error
				mockUserRelRepo.On("RemoveRelationship", mock.Anything, parentID, childID).Return(errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				UserRelRepo: mockUserRelRepo,
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Delete("/api/v1/parent/children/:childId", parentHandler.RemoveChild)

			// Setup mock expectations
			if tc.name != "Invalid Child ID" && tc.name != "Attempting to Remove Self" {
				childIDInt, _ := strconv.Atoi(tc.childIDParam)
				tc.setupMock(mockUserRelRepo, parentID, childIDInt)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/parent/children/%s", tc.childIDParam), nil)

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

			// Verify mock expectations
			mockUserRelRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_CreateChildAccount(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		input          models.CreateChildInput
		setupMock      func(mockUserService *serviceMocks.MockUserService, parentID int, input *models.CreateChildInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.CreateChildInput{
				Username:  "child_user",
				Email:     "child@example.com",
				FirstName: "Child",
				LastName:  "User",
				Password:  "password123",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, parentID int, input *models.CreateChildInput) {
				// Mock UserService.CreateChildAccount
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, input).Return(2, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Child account created successfully",
				"data":    map[string]interface{}{"child_id": float64(2)},
			},
		},
		{
			name: "Validation Error - Missing Required Fields",
			input: models.CreateChildInput{
				// Missing required fields
				Username: "child_user",
				// Missing email, password, etc.
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, parentID int, input *models.CreateChildInput) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field will contain validation details
			},
		},
		{
			name: "Username or Email Already Exists",
			input: models.CreateChildInput{
				Username:  "existing_user",
				Email:     "existing@example.com",
				FirstName: "Existing",
				LastName:  "User",
				Password:  "password123",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, parentID int, input *models.CreateChildInput) {
				// Mock UserService.CreateChildAccount returning username/email exists error
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, input).Return(0, service.ErrUsernameOrEmailExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrUsernameOrEmailExists.Error(),
			},
		},
		{
			name: "Database Error",
			input: models.CreateChildInput{
				Username:  "child_user",
				Email:     "child@example.com",
				FirstName: "Child",
				LastName:  "User",
				Password:  "password123",
			},
			setupMock: func(mockUserService *serviceMocks.MockUserService, parentID int, input *models.CreateChildInput) {
				// Mock UserService.CreateChildAccount with database error
				mockUserService.On("CreateChildAccount", mock.Anything, parentID, input).Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to create child account",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children/create", parentHandler.CreateChildAccount)

			// Setup mock expectations
			tc.setupMock(mockUserService, parentID, &tc.input)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children/create", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

			// For validation errors, only check success and message
			if tc.name == "Validation Error - Missing Required Fields" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data") // Should contain validation details
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockUserService.AssertExpectations(t)
		})
	}
}

func TestParentHandler_GenerateInvitationCode(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		childIDParam   string
		setupMock      func(mockInvitationService *serviceMocks.MockInvitationService, parentID, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "Success",
			childIDParam: "2",
			setupMock: func(mockInvitationService *serviceMocks.MockInvitationService, parentID, childID int) {
				// Mock GenerateAndStoreCode
				mockInvitationService.On("GenerateAndStoreCode", mock.Anything, parentID, childID).Return("INVITATION123", nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Invitation code generated successfully",
				"data":    map[string]interface{}{"invitation_code": "INVITATION123"},
			},
		},
		{
			name:         "Invalid Child ID",
			childIDParam: "invalid",
			setupMock: func(mockInvitationService *serviceMocks.MockInvitationService, parentID, childID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Child ID parameter",
			},
		},
		{
			name:         "Not Parent of Child",
			childIDParam: "2",
			setupMock: func(mockInvitationService *serviceMocks.MockInvitationService, parentID, childID int) {
				// Mock GenerateAndStoreCode returning forbidden error
				mockInvitationService.On("GenerateAndStoreCode", mock.Anything, parentID, childID).Return("", errors.New("forbidden: you are not authorized to generate codes for this child"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You are not authorized for this action",
			},
		},
		{
			name:         "Database Error",
			childIDParam: "2",
			setupMock: func(mockInvitationService *serviceMocks.MockInvitationService, parentID, childID int) {
				// Mock GenerateAndStoreCode with database error
				mockInvitationService.On("GenerateAndStoreCode", mock.Anything, parentID, childID).Return("", errors.New("database error"))
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
			// Setup
			app := fiber.New()
			mockUserRelRepo := new(mocks.MockUserRelationshipRepository)
			mockTaskRepo := new(mocks.MockTaskRepository)
			mockUserTaskRepo := new(mocks.MockUserTaskRepository)
			mockRewardRepo := new(mocks.MockRewardRepository)
			mockUserRewardRepo := new(mocks.MockUserRewardRepository)
			mockPointRepo := new(mocks.MockPointTransactionRepository)
			mockUserRepo := new(mocks.MockUserRepository)
			mockTaskService := new(serviceMocks.MockTaskService)
			mockRewardService := new(serviceMocks.MockRewardService)
			mockUserService := new(serviceMocks.MockUserService)
			mockInvitationService := new(serviceMocks.MockInvitationService)

			parentHandler := handlers.NewParentHandler(
				mockUserRelRepo,
				mockTaskRepo,
				mockUserTaskRepo,
				mockRewardRepo,
				mockUserRewardRepo,
				mockPointRepo,
				mockUserRepo,
				mockTaskService,
				mockRewardService,
				mockUserService,
				mockInvitationService,
			)

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/children/:childId/invitations", parentHandler.GenerateInvitationCode)

			// Setup mock expectations
			childIDInt, _ := strconv.Atoi(tc.childIDParam)
			if tc.name != "Invalid Child ID" {
				tc.setupMock(mockInvitationService, parentID, childIDInt)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/parent/children/%s/invitations", tc.childIDParam), nil)

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

			// For success cases, check structure but not exact content of data
			if tc.name == "Success" {
				assert.Equal(t, tc.expectedBody["success"], result["success"])
				assert.Equal(t, tc.expectedBody["message"], result["message"])
				assert.Contains(t, result, "data")

				// Check data contains invitation_code
				data := result["data"].(map[string]interface{})
				assert.Contains(t, data, "invitation_code")
				assert.NotEmpty(t, data["invitation_code"])
			} else {
				// For error cases, check exact match
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockInvitationService.AssertExpectations(t)
		})
	}
}
