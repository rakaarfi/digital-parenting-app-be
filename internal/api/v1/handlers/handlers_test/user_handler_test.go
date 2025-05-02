package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time" // Import time

	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	serviceMocks "github.com/rakaarfi/digital-parenting-app-be/internal/service/mocks" // Alias for service mocks
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to set user ID in context for testing protected routes
func setUserContext(c *fiber.Ctx, userID int) {
	c.Locals("user", &utils.JwtClaims{UserID: userID})
}

func TestUserHandler_UpdateMyProfile(t *testing.T) {
	const testUserID = 15 // Simulate the ID of the logged-in user

	tests := []struct {
		name           string
		inputBody      models.UpdateProfileInput
		setupContext   func(c *fiber.Ctx) // To simulate JWT middleware
		setupMock      func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			inputBody: models.UpdateProfileInput{
				Username: "newusername",
				Email:    "new@example.com",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				mockUserService.On("UpdateUserProfile", mock.Anything, userID, &input).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Profile updated successfully",
			},
		},
		{
			name:         "Unauthorized - Missing UserID in Context",
			inputBody:    models.UpdateProfileInput{Username: "test"},
			setupContext: func(c *fiber.Ctx) { /* Do nothing, simulate missing ID */ },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				// No service call expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Unauthorized: Invalid token",
			},
		},
		{
			name: "Validation Error - Invalid Email",
			inputBody: models.UpdateProfileInput{
				Email: "invalid-email",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// Data might contain details, only check message
			},
		},
		{
			name:         "Bad Request - Invalid Body",
			inputBody:    models.UpdateProfileInput{}, // Will cause BodyParser error if sent incorrectly
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest, // Expecting BodyParser to fail before validation/service
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid request body",
			},
		},
		{
			name: "Service Error - User Not Found",
			inputBody: models.UpdateProfileInput{
				Username: "test",
				Email:    "test@example.com", // Add valid email to pass validation
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				mockUserService.On("UpdateUserProfile", mock.Anything, userID, &input).Return(service.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User not found",
			},
		},
		{
			name: "Service Error - Username/Email Conflict",
			inputBody: models.UpdateProfileInput{
				Username: "existinguser",
				Email:    "existing@example.com", // Add valid email to pass validation
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				mockUserService.On("UpdateUserProfile", mock.Anything, userID, &input).Return(service.ErrUsernameOrEmailExists).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrUsernameOrEmailExists.Error(),
			},
		},
		{
			name: "Service Error - Internal Server Error",
			inputBody: models.UpdateProfileInput{
				Username: "test",
				Email:    "test2@example.com", // Add valid email to pass validation
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdateProfileInput) {
				mockUserService.On("UpdateUserProfile", mock.Anything, userID, &input).Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to update profile",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserService := serviceMocks.NewMockUserService(t)
			userHandler := handlers.NewUserHandler(mockUserService)

			// Custom handler wrapper to set context
			testHandler := func(c *fiber.Ctx) error {
				tc.setupContext(c)
				return userHandler.UpdateMyProfile(c)
			}
			app.Patch("/api/v1/user/profile", testHandler) // Register route

			// Setup mock expectations
			// Determine the userID based on context setup, default to 0 if context setup is missing
			currentUserID := 0
			if tc.name != "Unauthorized - Missing UserID in Context" {
				currentUserID = testUserID
			}
			tc.setupMock(mockUserService, currentUserID, tc.inputBody)

			// Prepare request body
			var req *http.Request
			if tc.name == "Bad Request - Invalid Body" {
				// Send invalid JSON to trigger BodyParser error
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/user/profile", bytes.NewReader([]byte("{invalid json")))
				req.Header.Set("Content-Type", "application/json")
			} else {
				bodyBytes, _ := json.Marshal(tc.inputBody)
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/user/profile", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			}

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

			// Special handling for validation error
			if tc.name == "Validation Error - Invalid Email" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserService.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetMyProfile(t *testing.T) {
	const testUserID = 15 // Simulate the ID of the logged-in user

	tests := []struct {
		name           string
		setupContext   func(c *fiber.Ctx)
		setupMock      func(mockUserService *serviceMocks.MockUserService, userID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "Success",
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int) {
				mockProfile := &models.User{ // Assuming GetUserProfile returns the full User model (minus password hash)
					ID:        userID,
					Username:  "testuser",
					Email:     "test@example.com",
					RoleID:    1,
					CreatedAt: time.Now(), // Use a valid time.Time value
					UpdatedAt: time.Now(), // Use a valid time.Time value
				}
				mockUserService.On("GetUserProfile", mock.Anything, userID).Return(mockProfile, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Profile retrieved successfully",
				"data": map[string]interface{}{
					"id":         float64(testUserID),
					"username":   "testuser",
					"email":      "test@example.com",
					"role_id":    float64(1),
					"created_at": mock.Anything, // We'll assert the structure, not exact time
					"updated_at": mock.Anything,
				},
			},
		},
		{
			name:         "Unauthorized - Missing UserID in Context",
			setupContext: func(c *fiber.Ctx) { /* Do nothing */ },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int) {
				// No service call expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Unauthorized: Invalid token",
			},
		},
		{
			name:         "Service Error - User Not Found", // Unlikely if token is valid, but test service error
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int) {
				mockUserService.On("GetUserProfile", mock.Anything, userID).Return(nil, service.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User profile not found",
			},
		},
		{
			name:         "Service Error - Internal Server Error",
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int) {
				mockUserService.On("GetUserProfile", mock.Anything, userID).Return(nil, errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to retrieve profile",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserService := serviceMocks.NewMockUserService(t)
			userHandler := handlers.NewUserHandler(mockUserService)

			// Custom handler wrapper to set context
			testHandler := func(c *fiber.Ctx) error {
				tc.setupContext(c)
				return userHandler.GetMyProfile(c)
			}
			app.Get("/api/v1/user/profile", testHandler) // Register route

			// Setup mock expectations
			currentUserID := 0
			if tc.name != "Unauthorized - Missing UserID in Context" {
				currentUserID = testUserID
			}
			tc.setupMock(mockUserService, currentUserID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)

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

			// Special handling for success case data (ignore time)
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"], "Success flag mismatch")
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"], "Message mismatch")

				expectedData := tc.expectedBody["data"].(map[string]interface{})
				actualData := responseBody["data"].(map[string]interface{})

				assert.Equal(t, expectedData["id"], actualData["id"], "User ID mismatch")
				assert.Equal(t, expectedData["username"], actualData["username"], "Username mismatch")
				assert.Equal(t, expectedData["email"], actualData["email"], "Email mismatch")
				assert.Equal(t, expectedData["role_id"], actualData["role_id"], "RoleID mismatch")
				// Check if time fields exist, but don't compare exact values
				assert.Contains(t, actualData, "created_at", "Missing created_at field")
				assert.Contains(t, actualData, "updated_at", "Missing updated_at field")
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserService.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateMyPassword(t *testing.T) {
	const testUserID = 15 // Simulate the ID of the logged-in user

	tests := []struct {
		name           string
		inputBody      models.UpdatePasswordInput
		setupContext   func(c *fiber.Ctx)
		setupMock      func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			inputBody: models.UpdatePasswordInput{
				OldPassword: "oldpassword",
				NewPassword: "newpassword123",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				mockUserService.On("ChangePassword", mock.Anything, userID, &input).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Password updated successfully",
			},
		},
		{
			name:         "Unauthorized - Missing UserID in Context",
			inputBody:    models.UpdatePasswordInput{OldPassword: "old", NewPassword: "new"},
			setupContext: func(c *fiber.Ctx) { /* Do nothing */ },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				// No service call expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Unauthorized: Invalid token",
			},
		},
		{
			name: "Validation Error - Missing New Password",
			inputBody: models.UpdatePasswordInput{
				OldPassword: "oldpassword",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
			},
		},
		{
			name:         "Bad Request - Invalid Body",
			inputBody:    models.UpdatePasswordInput{},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Invalid request body",
			},
		},
		{
			name: "Service Error - Incorrect Old Password",
			inputBody: models.UpdatePasswordInput{
				OldPassword: "wrongoldpassword",
				NewPassword: "newpassword123",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				mockUserService.On("ChangePassword", mock.Anything, userID, &input).Return(service.ErrIncorrectPassword).Once()
			},
			expectedStatus: http.StatusUnauthorized, // As per handler logic
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrIncorrectPassword.Error(),
			},
		},
		{
			name: "Service Error - User Not Found", // Should be rare if token is valid, but test service error handling
			inputBody: models.UpdatePasswordInput{
				OldPassword: "oldpassword",
				NewPassword: "newpassword123",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				mockUserService.On("ChangePassword", mock.Anything, userID, &input).Return(service.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "User not found",
			},
		},
		{
			name: "Service Error - Internal Server Error",
			inputBody: models.UpdatePasswordInput{
				OldPassword: "oldpassword",
				NewPassword: "newpassword123",
			},
			setupContext: func(c *fiber.Ctx) { setUserContext(c, testUserID) },
			setupMock: func(mockUserService *serviceMocks.MockUserService, userID int, input models.UpdatePasswordInput) {
				mockUserService.On("ChangePassword", mock.Anything, userID, &input).Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to change password",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockUserService := serviceMocks.NewMockUserService(t)
			userHandler := handlers.NewUserHandler(mockUserService)

			// Custom handler wrapper to set context
			testHandler := func(c *fiber.Ctx) error {
				tc.setupContext(c)
				return userHandler.UpdateMyPassword(c)
			}
			app.Patch("/api/v1/user/password", testHandler) // Register route

			// Setup mock expectations
			currentUserID := 0
			if tc.name != "Unauthorized - Missing UserID in Context" {
				currentUserID = testUserID
			}
			tc.setupMock(mockUserService, currentUserID, tc.inputBody)

			// Prepare request body
			var req *http.Request
			if tc.name == "Bad Request - Invalid Body" {
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/user/password", bytes.NewReader([]byte("invalid json")))
				req.Header.Set("Content-Type", "application/json")
			} else {
				bodyBytes, _ := json.Marshal(tc.inputBody)
				req = httptest.NewRequest(http.MethodPatch, "/api/v1/user/password", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			}

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

			// Special handling for validation error
			if tc.name == "Validation Error - Missing New Password" {
				assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
				assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
			} else {
				assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
			}

			// Verify mock calls
			mockUserService.AssertExpectations(t)
		})
	}
}
