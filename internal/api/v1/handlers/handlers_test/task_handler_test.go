package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-playground/validator/v10"
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

func TestTaskHandler_CreateTask(t *testing.T) {
	tests := []struct {
		name           string
		input          models.CreateTaskInput
		setupMock      func(mockService *serviceMocks.MockTaskService, input models.CreateTaskInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.CreateTaskInput{
				TaskName:        "Test Task",
				TaskDescription: "Test Description",
				TaskPoint:       100,
			},
			setupMock: func(mockService *serviceMocks.MockTaskService, input models.CreateTaskInput) {
				// We're not using the TaskService in CreateTaskDefinition, so no need to set up expectations
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task definition created",
				"data": map[string]interface{}{
					"task_id": float64(1),
				},
			},
		},
		{
			name:  "Validation Error",
			input: models.CreateTaskInput{
				// Missing required fields
			},
			setupMock: func(mockService *serviceMocks.MockTaskService, input models.CreateTaskInput) {
				// We're not using the TaskService in CreateTaskDefinition, so no need to set up expectations
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
			},
		},
		{
			name: "Service Error",
			input: models.CreateTaskInput{
				TaskName:        "Test Task",
				TaskDescription: "Test Description",
				TaskPoint:       100,
			},
			setupMock: func(mockService *serviceMocks.MockTaskService, input models.CreateTaskInput) {
				// We're not using the TaskService in CreateTaskDefinition, so no need to set up expectations
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
			mockTaskRepo := new(mocks.MockTaskRepository)

			// Setup the mock for TaskRepo.CreateTask based on the test case
			if tc.name == "Service Error" {
				mockTaskRepo.On("CreateTask", mock.Anything, mock.AnythingOfType("*models.Task")).
					Return(0, errors.New("database error"))
			} else {
				mockTaskRepo.On("CreateTask", mock.Anything, mock.AnythingOfType("*models.Task")).
					Return(1, nil)
			}

			taskHandler := &handlers.ParentHandler{
				TaskService: mockTaskService,
				TaskRepo:    mockTaskRepo,
				Validate:    validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(1, "parent_user", "Parent"))

			app.Post("/api/v1/tasks", taskHandler.CreateTaskDefinition)

			// We're not using the TaskService in CreateTaskDefinition, so no need to set up expectations

			// Prepare request
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err)

			if tc.name == "Validation Error" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody)
			}

			// Verify mock expectations
			mockTaskService.AssertExpectations(t)
		})
	}
}

func TestParentHandler_GetMyTaskDefinitions(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockTaskRepository, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockTaskRepository, parentID int) {
				mockTasks := []models.Task{
					{ID: 1, TaskName: "Task 1", TaskPoint: 100, TaskDescription: "Description 1", CreatedByUserID: parentID},
					{ID: 2, TaskName: "Task 2", TaskPoint: 200, TaskDescription: "Description 2", CreatedByUserID: parentID},
				}
				mockRepo.On("GetTasksByCreatorID", mock.Anything, parentID, 1, 10).Return(mockTasks, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task definitions retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":                 float64(1),
						"task_name":          "Task 1",
						"task_point":         float64(100),
						"task_description":   "Description 1",
						"created_by_user_id": float64(parentID),
					},
					map[string]interface{}{
						"id":                 float64(2),
						"task_name":          "Task 2",
						"task_point":         float64(200),
						"task_description":   "Description 2",
						"created_by_user_id": float64(parentID),
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
			name: "No Tasks Found",
			setupMock: func(mockRepo *mocks.MockTaskRepository, parentID int) {
				mockRepo.On("GetTasksByCreatorID", mock.Anything, parentID, 1, 10).Return([]models.Task{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task definitions retrieved successfully",
				"data":    []interface{}{},
				"meta": map[string]interface{}{
					"total_items":  float64(0),
					"current_page": float64(1),
					"per_page":     float64(10),
					"total_pages":  float64(0),
				},
			},
		},
		{
			name: "Database Error",
			setupMock: func(mockRepo *mocks.MockTaskRepository, parentID int) {
				mockRepo.On("GetTasksByCreatorID", mock.Anything, parentID, 1, 10).Return(nil, 0, errors.New("database error"))
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
			mockTaskRepo := new(mocks.MockTaskRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				TaskRepo: mockTaskRepo,
				Validate: validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Get("/api/v1/parent/tasks", parentHandler.GetMyTaskDefinitions)

			// Setup mock expectations
			tc.setupMock(mockTaskRepo, parentID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/tasks", nil)

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
						assert.Equal(t, expectedTask["task_name"], actualTask["task_name"])
						assert.Equal(t, expectedTask["task_point"], actualTask["task_point"])
						assert.Equal(t, expectedTask["task_description"], actualTask["task_description"])
						assert.Equal(t, expectedTask["created_by_user_id"], actualTask["created_by_user_id"])
					}
				}
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_UpdateMyTaskDefinition(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		taskID         string
		input          models.UpdateTaskInput
		setupMock      func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "Success",
			taskID: "1",
			input: models.UpdateTaskInput{
				TaskName:        "Updated Task",
				TaskPoint:       150,
				TaskDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput) {
				// Mock GetTaskByID
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Original Task",
					TaskPoint:       100,
					TaskDescription: "Original Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock UpdateTask
				mockRepo.On("UpdateTask", mock.Anything, mock.AnythingOfType("*models.Task"), parentID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task definition updated successfully",
			},
		},
		{
			name:   "Invalid Task ID",
			taskID: "invalid",
			input: models.UpdateTaskInput{
				TaskName:        "Updated Task",
				TaskPoint:       150,
				TaskDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Task ID parameter",
			},
		},
		{
			name:   "Task Not Found",
			taskID: "999",
			input: models.UpdateTaskInput{
				TaskName:        "Updated Task",
				TaskPoint:       150,
				TaskDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput) {
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Task definition not found",
			},
		},
		{
			name:   "Not Task Owner",
			taskID: "2",
			input: models.UpdateTaskInput{
				TaskName:        "Updated Task",
				TaskPoint:       150,
				TaskDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput) {
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Original Task",
					TaskPoint:       100,
					TaskDescription: "Original Description",
					CreatedByUserID: parentID + 1, // Different user ID
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You do not have permission to modify this task.",
			},
		},
		{
			name:   "Database Error",
			taskID: "1",
			input: models.UpdateTaskInput{
				TaskName:        "Updated Task",
				TaskPoint:       150,
				TaskDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int, input models.UpdateTaskInput) {
				// Mock GetTaskByID
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Original Task",
					TaskPoint:       100,
					TaskDescription: "Original Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock UpdateTask with error
				mockRepo.On("UpdateTask", mock.Anything, mock.AnythingOfType("*models.Task"), parentID).Return(errors.New("database error"))
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
			mockTaskRepo := new(mocks.MockTaskRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				TaskRepo: mockTaskRepo,
				Validate: validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Patch("/api/v1/parent/tasks/:taskId", parentHandler.UpdateMyTaskDefinition)

			// Setup mock expectations
			taskID, _ := strconv.Atoi(tc.taskID)
			if tc.name != "Invalid Task ID" {
				tc.setupMock(mockTaskRepo, taskID, parentID, tc.input)
			}

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/parent/tasks/"+tc.taskID, bytes.NewReader(bodyBytes))
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
			mockTaskRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_DeleteMyTaskDefinition(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(mockRepo *mocks.MockTaskRepository, taskID, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "Success",
			taskID: "1",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				// Mock GetTaskByID
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Task to Delete",
					TaskPoint:       100,
					TaskDescription: "Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock DeleteTask
				mockRepo.On("DeleteTask", mock.Anything, taskID, parentID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Task definition deleted successfully",
			},
		},
		{
			name:   "Invalid Task ID",
			taskID: "invalid",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Task ID parameter",
			},
		},
		{
			name:   "Task Not Found",
			taskID: "999",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Task definition not found",
			},
		},
		{
			name:   "Not Task Owner",
			taskID: "2",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Task to Delete",
					TaskPoint:       100,
					TaskDescription: "Description",
					CreatedByUserID: parentID + 1, // Different user ID
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You can only delete tasks you created.",
			},
		},
		{
			name:   "Task Currently Assigned",
			taskID: "3",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				// Mock GetTaskByID
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Task to Delete",
					TaskPoint:       100,
					TaskDescription: "Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock DeleteTask with error indicating task is assigned
				mockRepo.On("DeleteTask", mock.Anything, taskID, parentID).Return(errors.New("task is currently assigned to a child"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "task is currently assigned to a child",
			},
		},
		{
			name:   "Database Error",
			taskID: "4",
			setupMock: func(mockRepo *mocks.MockTaskRepository, taskID, parentID int) {
				// Mock GetTaskByID
				mockRepo.On("GetTaskByID", mock.Anything, taskID).Return(&models.Task{
					ID:              taskID,
					TaskName:        "Task to Delete",
					TaskPoint:       100,
					TaskDescription: "Description",
					CreatedByUserID: parentID,
				}, nil)

				// Mock DeleteTask with generic error
				mockRepo.On("DeleteTask", mock.Anything, taskID, parentID).Return(errors.New("database error"))
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
			mockTaskRepo := new(mocks.MockTaskRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				TaskRepo: mockTaskRepo,
				Validate: validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Delete("/api/v1/parent/tasks/:taskId", parentHandler.DeleteMyTaskDefinition)

			// Setup mock expectations
			taskID, _ := strconv.Atoi(tc.taskID)
			if tc.name != "Invalid Task ID" {
				tc.setupMock(mockTaskRepo, taskID, parentID)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/parent/tasks/"+tc.taskID, nil)

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
			mockTaskRepo.AssertExpectations(t)
		})
	}
}
