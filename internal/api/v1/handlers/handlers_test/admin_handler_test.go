package handlers_test

import (
	"bytes" // Import bytes
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"        // Import pgx
	"github.com/jackc/pgx/v5/pgconn" // Import pgconn
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	repoMocks "github.com/rakaarfi/digital-parenting-app-be/internal/repository/mocks" // Alias for repository mocks
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdminHandler_GetAllUsers(t *testing.T) {
	defaultPage := 1
	defaultLimit := 10

	// --- Test Cases ---
	tests := []struct {
		name           string
		queryParams    string // e.g., "?page=2&limit=5"
		setupMock      func(mockUserRepo *repoMocks.MockUserRepository, page, limit int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success - Default Pagination",
			queryParams: "",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, page, limit int) {
				mockUsers := []models.User{
					{ID: 1, Username: "user1", Email: "user1@test.com", RoleID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
					{ID: 2, Username: "user2", Email: "user2@test.com", RoleID: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				}
				mockUserRepo.On("GetAllUsers", mock.Anything, page, limit).Return(mockUsers, 2, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Users retrieved successfully",
				"data": []interface{}{ // JSON unmarshals into []interface{}
					map[string]interface{}{"id": float64(1), "username": "user1", "email": "user1@test.com", "role_id": float64(1), "created_at": mock.Anything, "updated_at": mock.Anything},
					map[string]interface{}{"id": float64(2), "username": "user2", "email": "user2@test.com", "role_id": float64(2), "created_at": mock.Anything, "updated_at": mock.Anything},
				},
				"meta": map[string]interface{}{
					"total_items":  float64(2),
					"current_page": float64(defaultPage),
					"per_page":     float64(defaultLimit),
					"total_pages":  float64(1),
				},
			},
		},
		{
			name:        "Success - Custom Pagination",
			queryParams: "?page=2&limit=5",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, page, limit int) {
				mockUsers := []models.User{
					{ID: 6, Username: "user6", Email: "user6@test.com", RoleID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				}
				mockUserRepo.On("GetAllUsers", mock.Anything, page, limit).Return(mockUsers, 6, nil).Once() // Total 6 users
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Users retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{"id": float64(6), "username": "user6", "email": "user6@test.com", "role_id": float64(1), "created_at": mock.Anything, "updated_at": mock.Anything},
				},
				"meta": map[string]interface{}{
					"total_items":  float64(6),
					"current_page": float64(2),
					"per_page":     float64(5),
					"total_pages":  float64(2), // 6 items, 5 per page -> 2 pages
				},
			},
		},
		{
			name:        "Repository Error",
			queryParams: "",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, page, limit int) {
				mockUserRepo.On("GetAllUsers", mock.Anything, page, limit).Return(nil, 0, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to retrieve users",
			},
		},
		// Add test case for invalid query params if needed (Fiber usually handles this)
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t) // Needed for handler creation
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Get("/api/v1/admin/users", adminHandler.GetAllUsers) // Register route

			// Determine expected page and limit based on query params
			page := defaultPage
			limit := defaultLimit
			if tc.queryParams == "?page=2&limit=5" {
				page = 2
				limit = 5
			}

			// Setup mock expectations
			tc.setupMock(mockUserRepo, page, limit)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users"+tc.queryParams, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// Special handling for successful responses to ignore time fields in data
			if tc.expectedStatus == http.StatusOK {
				// Compare meta first
				assert.Equal(t, tc.expectedBody["meta"], responseBody["meta"], "Meta mismatch")
				// Compare success and message
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"], "Success flag mismatch")
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"], "Message mismatch")

				// Compare data length
				expectedData := tc.expectedBody["data"].([]interface{})
				actualData := responseBody["data"].([]interface{})
				assert.Equal(t, len(expectedData), len(actualData), "Data length mismatch")

				// Compare individual user data items (ignoring time)
				for i := range expectedData {
					expectedUser := expectedData[i].(map[string]interface{})
					actualUser := actualData[i].(map[string]interface{})
					assert.Equal(t, expectedUser["id"], actualUser["id"], fmt.Sprintf("User %d ID mismatch", i))
					assert.Equal(t, expectedUser["username"], actualUser["username"], fmt.Sprintf("User %d Username mismatch", i))
					assert.Equal(t, expectedUser["email"], actualUser["email"], fmt.Sprintf("User %d Email mismatch", i))
					assert.Equal(t, expectedUser["role_id"], actualUser["role_id"], fmt.Sprintf("User %d RoleID mismatch", i))
				}
			} else {
				// For error responses, compare the whole body directly
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserRepo.AssertExpectations(t)
			// mockRoleRepo.AssertExpectations(t) // No calls expected to role repo in this handler
		})
	}
}

func TestAdminHandler_DeleteRole(t *testing.T) {
	tests := []struct {
		name           string
		roleIDParam    string
		setupMock      func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			roleIDParam: "3", // Assuming ID 3 is a deletable role
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRoleRepo.On("DeleteRole", mock.Anything, roleID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Role deleted successfully",
			},
		},
		{
			name:        "Invalid Role ID Parameter",
			roleIDParam: "abc",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Role ID parameter",
			},
		},
		{
			name:        "Attempt to Delete Base Role - Admin (ID 1)",
			roleIDParam: "1",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot delete base roles (Admin/Child)",
			},
		},
		{
			name:        "Attempt to Delete Base Role - Child (ID 2)", // Assuming Child is ID 2
			roleIDParam: "2",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot delete base roles (Admin/Child)",
			},
		},
		{
			name:        "Role Not Found",
			roleIDParam: "999",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRoleRepo.On("DeleteRole", mock.Anything, roleID).Return(pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role with ID 999 not found",
			},
		},
		{
			name:        "Conflict - Role Still In Use",
			roleIDParam: "4",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				// Simulate conflict error message from repo
				mockRoleRepo.On("DeleteRole", mock.Anything, roleID).Return(errors.New("cannot delete role: users are still assigned to this role")).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to delete role due to role still in use",
				// Data field might contain details, only check message
			},
		},
		{
			name:        "Repository Error",
			roleIDParam: "5",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRoleRepo.On("DeleteRole", mock.Anything, roleID).Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to delete role",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Delete("/api/v1/admin/roles/:roleId", adminHandler.DeleteRole) // Register route

			// Setup mock expectations
			roleID := 0
			if tc.name != "Invalid Role ID Parameter" {
				fmt.Sscan(tc.roleIDParam, &roleID)
			}
			tc.setupMock(mockRoleRepo, roleID)

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/roles/"+tc.roleIDParam, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// Special handling for conflict error
			if tc.name == "Conflict - Role Still In Use" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockRoleRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t) // No calls expected
		})
	}
}

func TestAdminHandler_UpdateRole(t *testing.T) {
	tests := []struct {
		name           string
		roleIDParam    string
		inputBody      models.Role
		setupMock      func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			roleIDParam: "3",
			inputBody: models.Role{
				Name: "UpdatedRoleName",
			},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				// Expect UpdateRole call with ID set from param
				expectedInput := input
				expectedInput.ID = roleID
				mockRoleRepo.On("UpdateRole", mock.Anything, &expectedInput).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Role updated successfully",
			},
		},
		{
			name:        "Validation Error - Missing Name",
			roleIDParam: "3",
			inputBody:   models.Role{}, // Empty name
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed: role name is required",
			},
		},
		{
			name:        "Invalid Role ID Parameter",
			roleIDParam: "abc",
			inputBody:   models.Role{Name: "Test"},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Role ID parameter",
			},
		},
		{
			name:        "Role Not Found",
			roleIDParam: "999",
			inputBody:   models.Role{Name: "Test"},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				expectedInput := input
				expectedInput.ID = roleID
				mockRoleRepo.On("UpdateRole", mock.Anything, &expectedInput).Return(pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role with ID 999 not found",
			},
		},
		{
			name:        "Conflict - Role Name Exists",
			roleIDParam: "3",
			inputBody: models.Role{
				Name: "ExistingRoleName",
			},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				expectedInput := input
				expectedInput.ID = roleID
				// Simulate unique constraint error
				mockRoleRepo.On("UpdateRole", mock.Anything, &expectedInput).Return(errors.New("ERROR: duplicate key value violates unique constraint \"roles_name_key\" (SQLSTATE 23505) - already exists")).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role name already exists, please choose a different name",
			},
		},
		{
			name:        "Repository Error",
			roleIDParam: "3",
			inputBody:   models.Role{Name: "Test"},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int, input models.Role) {
				expectedInput := input
				expectedInput.ID = roleID
				mockRoleRepo.On("UpdateRole", mock.Anything, &expectedInput).Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to update role",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Patch("/api/v1/admin/roles/:roleId", adminHandler.UpdateRole) // Register route

			// Setup mock expectations
			roleID := 0
			if tc.name != "Invalid Role ID Parameter" {
				fmt.Sscan(tc.roleIDParam, &roleID)
			}
			tc.setupMock(mockRoleRepo, roleID, tc.inputBody)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.inputBody)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/roles/"+tc.roleIDParam, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// Special handling for validation/conflict errors
			if tc.name == "Validation Error - Missing Name" || tc.name == "Conflict - Role Name Exists" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockRoleRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t) // No calls expected
		})
	}
}

func TestAdminHandler_GetRoleByID(t *testing.T) {
	tests := []struct {
		name           string
		roleIDParam    string
		setupMock      func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			roleIDParam: "1",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRole := &models.Role{ID: roleID, Name: "Admin"}
				mockRoleRepo.On("GetRoleByID", mock.Anything, roleID).Return(mockRole, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Role retrieved successfully",
				"data":    map[string]interface{}{"id": float64(1), "name": "Admin"},
			},
		},
		{
			name:        "Role Not Found",
			roleIDParam: "999",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRoleRepo.On("GetRoleByID", mock.Anything, roleID).Return(nil, pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role with ID 999 not found",
			},
		},
		{
			name:        "Invalid Role ID Parameter",
			roleIDParam: "abc",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Role ID parameter",
			},
		},
		{
			name:        "Repository Error",
			roleIDParam: "1",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, roleID int) {
				mockRoleRepo.On("GetRoleByID", mock.Anything, roleID).Return(nil, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to retrieve role",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Get("/api/v1/admin/roles/:roleId", adminHandler.GetRoleByID) // Register route

			// Setup mock expectations
			roleID := 0
			if tc.name != "Invalid Role ID Parameter" {
				fmt.Sscan(tc.roleIDParam, &roleID)
			}
			tc.setupMock(mockRoleRepo, roleID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles/"+tc.roleIDParam, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")
			assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")

			// Verify mock calls
			mockRoleRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t) // No calls expected
		})
	}
}

func TestAdminHandler_GetAllRoles(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(mockRoleRepo *repoMocks.MockRoleRepository)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success - Multiple Roles",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository) {
				mockRoles := []models.Role{
					{ID: 1, Name: "Admin"},
					{ID: 2, Name: "Parent"},
					{ID: 3, Name: "Child"},
				}
				mockRoleRepo.On("GetAllRoles", mock.Anything).Return(mockRoles, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Roles retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{"id": float64(1), "name": "Admin"},
					map[string]interface{}{"id": float64(2), "name": "Parent"},
					map[string]interface{}{"id": float64(3), "name": "Child"},
				},
			},
		},
		{
			name: "Success - No Roles",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository) {
				mockRoles := []models.Role{}
				mockRoleRepo.On("GetAllRoles", mock.Anything).Return(mockRoles, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Roles retrieved successfully",
				"data":    []interface{}{}, // Expect empty array
			},
		},
		{
			name: "Repository Error",
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository) {
				mockRoleRepo.On("GetAllRoles", mock.Anything).Return(nil, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to retrieve roles",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Get("/api/v1/admin/roles", adminHandler.GetAllRoles) // Register route

			// Setup mock expectations
			tc.setupMock(mockRoleRepo)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")
			assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")

			// Verify mock calls
			mockRoleRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t) // No calls expected
		})
	}
}

func TestAdminHandler_CreateRole(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      models.Role
		setupMock      func(mockRoleRepo *repoMocks.MockRoleRepository, input models.Role)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			inputBody: models.Role{
				Name: "NewRole",
			},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, input models.Role) {
				// Need to pass a pointer to CreateRole
				mockRoleRepo.On("CreateRole", mock.Anything, &input).Return(5, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Role created successfully",
				"data":    map[string]interface{}{"role_id": float64(5)},
			},
		},
		{
			name:      "Validation Error - Missing Name",
			inputBody: models.Role{}, // Empty name
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, input models.Role) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed: role name is required",
				// Data might contain details, only check message
			},
		},
		{
			name: "Conflict - Role Name Exists",
			inputBody: models.Role{
				Name: "ExistingRole",
			},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, input models.Role) {
				// Simulate unique constraint error
				mockRoleRepo.On("CreateRole", mock.Anything, &input).Return(0, errors.New("ERROR: duplicate key value violates unique constraint \"roles_name_key\" (SQLSTATE 23505) - already exists")).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role with same name already exists",
				// Data might contain details, only check message
			},
		},
		{
			name: "Repository Error",
			inputBody: models.Role{
				Name: "ErrorRole",
			},
			setupMock: func(mockRoleRepo *repoMocks.MockRoleRepository, input models.Role) {
				mockRoleRepo.On("CreateRole", mock.Anything, &input).Return(0, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to create role",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t) // Still needed for handler creation
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Post("/api/v1/admin/roles", adminHandler.CreateRole) // Register route

			// Setup mock expectations
			tc.setupMock(mockRoleRepo, tc.inputBody)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.inputBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// Special handling for validation/conflict errors
			if tc.name == "Validation Error - Missing Name" || tc.name == "Conflict - Role Name Exists" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockRoleRepo.AssertExpectations(t)
			// No UserRepo calls expected
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	const adminTestID = 5 // Simulate the ID of the admin performing the action

	tests := []struct {
		name           string
		targetUserID   string             // User ID to delete (from URL param)
		setupContext   func(c *fiber.Ctx) // Function to set up context (e.g., admin ID)
		setupMock      func(mockUserRepo *repoMocks.MockUserRepository, userID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "Success",
			targetUserID: "10",
			setupContext: func(c *fiber.Ctx) {
				// Simulate JWT middleware setting the user claims
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUserRepo.On("DeleteUserByID", mock.Anything, userID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "User with ID 10 deleted successfully",
			},
		},
		{
			name:         "Invalid User ID Parameter",
			targetUserID: "abc",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid User ID parameter",
			},
		},
		{
			name:         "Attempt to Delete Self",
			targetUserID: fmt.Sprintf("%d", adminTestID), // Target ID is the same as admin ID
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				// No mock call expected as the check happens before repo call
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Admin cannot delete their own account",
			},
		},
		{
			name:         "User Not Found",
			targetUserID: "999",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUserRepo.On("DeleteUserByID", mock.Anything, userID).Return(pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User with ID 999 not found",
			},
		},
		{
			name:         "Conflict - Foreign Key Violation",
			targetUserID: "11",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				// Simulate a pgconn error for foreign key violation
				pgErr := &pgconn.PgError{Code: "23503"} // Foreign key violation code
				mockUserRepo.On("DeleteUserByID", mock.Anything, userID).Return(pgErr).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Cannot delete user: User has existing related records (tasks, rewards, points, etc.).",
			},
		},
		{
			name:         "Repository Error",
			targetUserID: "12",
			setupContext: func(c *fiber.Ctx) {
				c.Locals("user", &utils.JwtClaims{UserID: adminTestID})
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUserRepo.On("DeleteUserByID", mock.Anything, userID).Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to delete user",
			},
		},
		// Note: Test for failing to extract admin ID is omitted as it relies on middleware error handling,
		// which is outside the scope of this handler unit test. We assume middleware works correctly.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)

			// Custom handler wrapper to set context before the actual handler runs
			testHandler := func(c *fiber.Ctx) error {
				tc.setupContext(c) // Set up context (e.g., admin ID)
				return adminHandler.DeleteUser(c)
			}
			app.Delete("/api/v1/admin/users/:userId", testHandler) // Register route with the wrapper

			// Setup mock expectations
			targetUserIDInt := 0
			if tc.name != "Invalid User ID Parameter" {
				fmt.Sscan(tc.targetUserID, &targetUserIDInt)
			}
			tc.setupMock(mockUserRepo, targetUserIDInt)

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+tc.targetUserID, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")
			assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")

			// Verify mock calls
			mockUserRepo.AssertExpectations(t)
			// No RoleRepo calls expected in DeleteUser
			mockRoleRepo.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_UpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		userIDParam    string
		inputBody      models.AdminUpdateUserInput
		setupMock      func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success - Update Username Only",
			userIDParam: "1",
			inputBody: models.AdminUpdateUserInput{
				Username: "updateduser",
				// Email and RoleID are omitted (or zero value)
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No RoleRepo call expected as RoleID is 0
				mockUserRepo.On("UpdateUserByID", mock.Anything, userID, &input).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "User with ID 1 updated successfully",
			},
		},
		{
			name:        "Success - Update Email and RoleID",
			userIDParam: "2",
			inputBody: models.AdminUpdateUserInput{
				Email:  "newemail@example.com",
				RoleID: 2, // Update to Role 2
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// Expect RoleRepo call because RoleID is provided and non-zero
				mockRoleRepo.On("GetRoleByID", mock.Anything, input.RoleID).Return(&models.Role{ID: input.RoleID, Name: "Child"}, nil).Once()
				mockUserRepo.On("UpdateUserByID", mock.Anything, userID, &input).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "User with ID 2 updated successfully",
			},
		},
		{
			name:        "Validation Error - Invalid Email",
			userIDParam: "1",
			inputBody: models.AdminUpdateUserInput{
				Email: "invalid-email",
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No repo calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data field might contain specific validation errors, we check only message
			},
		},
		{
			name:        "Invalid User ID Parameter",
			userIDParam: "abc",
			inputBody:   models.AdminUpdateUserInput{Username: "test"},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No repo calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid User ID parameter",
			},
		},
		{
			name:        "User Not Found During Update",
			userIDParam: "999",
			inputBody:   models.AdminUpdateUserInput{Username: "test"},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No RoleRepo call needed if RoleID is 0
				mockUserRepo.On("UpdateUserByID", mock.Anything, userID, &input).Return(pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User with ID 999 not found",
			},
		},
		{
			name:        "Invalid Role ID Provided",
			userIDParam: "1",
			inputBody: models.AdminUpdateUserInput{
				RoleID: 99, // Invalid Role ID
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				mockRoleRepo.On("GetRoleByID", mock.Anything, input.RoleID).Return(nil, pgx.ErrNoRows).Once()
				// UserRepo.UpdateUserByID should not be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid Role ID provided",
			},
		},
		{
			name:        "Role Check Internal Error",
			userIDParam: "1",
			inputBody: models.AdminUpdateUserInput{
				RoleID: 2,
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				mockRoleRepo.On("GetRoleByID", mock.Anything, input.RoleID).Return(nil, errors.New("db error")).Once()
				// UserRepo.UpdateUserByID should not be called
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to validate role",
			},
		},
		{
			name:        "Unique Constraint Violation During Update",
			userIDParam: "1",
			inputBody: models.AdminUpdateUserInput{
				Username: "existinguser", // Assume this username already exists for another user
			},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No RoleRepo call needed if RoleID is 0
				// Simulate unique constraint error from UserRepo
				mockUserRepo.On("UpdateUserByID", mock.Anything, userID, &input).Return(errors.New("ERROR: duplicate key value violates unique constraint \"users_username_key\" (SQLSTATE 23505) - already exists")).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Unique constraint violation during user update by admin",
				// Data field might contain details, we check only message
			},
		},
		{
			name:        "Generic Update Error",
			userIDParam: "1",
			inputBody:   models.AdminUpdateUserInput{Username: "newuser"},
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, mockRoleRepo *repoMocks.MockRoleRepository, userID int, input models.AdminUpdateUserInput) {
				// No RoleRepo call needed if RoleID is 0
				mockUserRepo.On("UpdateUserByID", mock.Anything, userID, &input).Return(errors.New("some other db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to update user",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Patch("/api/v1/admin/users/:userId", adminHandler.UpdateUser) // Register route

			// Setup mock expectations
			userID := 0
			if tc.name != "Invalid User ID Parameter" {
				fmt.Sscan(tc.userIDParam, &userID)
			}
			tc.setupMock(mockUserRepo, mockRoleRepo, userID, tc.inputBody)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.inputBody)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+tc.userIDParam, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// For validation errors or unique constraint, we might only check message/success
			if tc.name == "Validation Error - Invalid Email" || tc.name == "Unique Constraint Violation During Update" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserRepo.AssertExpectations(t)
			mockRoleRepo.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_GetUserByID(t *testing.T) {
	tests := []struct {
		name           string
		userIDParam    string
		setupMock      func(mockUserRepo *repoMocks.MockUserRepository, userID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			userIDParam: "1",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUser := &models.User{
					ID:        userID,
					Username:  "testuser",
					Email:     "test@example.com",
					RoleID:    1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				mockUserRepo.On("GetUserByID", mock.Anything, userID).Return(mockUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "User retrieved successfully",
				"data": map[string]interface{}{
					"id":         float64(1),
					"username":   "testuser",
					"email":      "test@example.com",
					"role_id":    float64(1),
					"created_at": mock.Anything,
					"updated_at": mock.Anything,
				},
			},
		},
		{
			name:        "User Not Found",
			userIDParam: "999",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUserRepo.On("GetUserByID", mock.Anything, userID).Return(nil, pgx.ErrNoRows).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User with ID 999 not found",
			},
		},
		{
			name:        "Invalid User ID Parameter",
			userIDParam: "abc",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				// No mock call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid User ID parameter",
			},
		},
		{
			name:        "Repository Error",
			userIDParam: "1",
			setupMock: func(mockUserRepo *repoMocks.MockUserRepository, userID int) {
				mockUserRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to retrieve user",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserRepo := repoMocks.NewMockUserRepository(t)
			mockRoleRepo := repoMocks.NewMockRoleRepository(t)
			adminHandler := handlers.NewAdminHandler(mockUserRepo, mockRoleRepo)
			app.Get("/api/v1/admin/users/:userId", adminHandler.GetUserByID) // Register route

			// Setup mock expectations
			userID := 0 // Default value, will be overwritten if param is valid int
			if tc.name != "Invalid User ID Parameter" {
				fmt.Sscan(tc.userIDParam, &userID) // Simple way to convert valid string ID
				tc.setupMock(mockUserRepo, userID)
			} else {
				tc.setupMock(mockUserRepo, 0) // Pass 0 or any value, mock won't be called
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+tc.userIDParam, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err, "Failed to decode response body")

			// Special handling for successful response data (ignore time)
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"], "Success flag mismatch")
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"], "Message mismatch")

				expectedData := tc.expectedBody["data"].(map[string]interface{})
				actualData := responseBody["data"].(map[string]interface{})

				assert.Equal(t, expectedData["id"], actualData["id"], "User ID mismatch")
				assert.Equal(t, expectedData["username"], actualData["username"], "Username mismatch")
				assert.Equal(t, expectedData["email"], actualData["email"], "Email mismatch")
				assert.Equal(t, expectedData["role_id"], actualData["role_id"], "RoleID mismatch")
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserRepo.AssertExpectations(t)
		})
	}
}
