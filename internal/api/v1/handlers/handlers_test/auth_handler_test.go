package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthHandler_Register(t *testing.T) {
	// --- Test Cases ---
	tests := []struct {
		name           string
		input          models.RegisterUserInput
		setupMock      func(mockService *mocks.MockAuthService, input models.RegisterUserInput)
		expectedStatus int
		expectedBody   map[string]interface{} // Use map for flexible JSON comparison
	}{
		{
			name: "Success",
			input: models.RegisterUserInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				RoleID:   1, // Assuming RoleID 1 is Parent/Child
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				mockService.On("RegisterUser", mock.Anything, &input).Return(1, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "User registered successfully",
				"data":    map[string]interface{}{"user_id": float64(1)}, // Fiber returns float64 for JSON numbers
			},
		},
		{
			name: "Validation Error - Missing Username",
			input: models.RegisterUserInput{
				Email:    "test@example.com",
				Password: "password123",
				RoleID:   1,
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
				// We don't assert the exact validation detail structure here, just the overall message
			},
		},
		{
			name: "Service Error - Username/Email Conflict",
			input: models.RegisterUserInput{
				Username: "existinguser",
				Email:    "existing@example.com",
				Password: "password123",
				RoleID:   1,
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				mockService.On("RegisterUser", mock.Anything, &input).Return(0, service.ErrUsernameOrEmailExists).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrUsernameOrEmailExists.Error(),
			},
		},
		{
			name: "Service Error - Role Not Found",
			input: models.RegisterUserInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				RoleID:   99, // Non-existent role
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				mockService.On("RegisterUser", mock.Anything, &input).Return(0, service.ErrRoleNotFound).Once()
			},
			expectedStatus: http.StatusBadRequest, // As per handler logic
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Role with ID 99 not found",
			},
		},
		{
			name: "Service Error - Disallowed Role Registration",
			input: models.RegisterUserInput{
				Username: "adminuser",
				Email:    "admin@example.com",
				Password: "password123",
				RoleID:   3, // Assuming RoleID 3 is Admin
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				mockService.On("RegisterUser", mock.Anything, &input).Return(0, service.ErrDisallowedRoleRegistration).Once()
			},
			expectedStatus: http.StatusBadRequest, // As per handler logic
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrDisallowedRoleRegistration.Error(),
			},
		},
		{
			name: "Service Error - Generic Internal Error",
			input: models.RegisterUserInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				RoleID:   1,
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.RegisterUserInput) {
				mockService.On("RegisterUser", mock.Anything, &input).Return(0, errors.New("some internal error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Failed to register user",
			},
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockAuthService := mocks.NewMockAuthService(t)
			authHandler := handlers.NewAuthHandler(mockAuthService)
			app.Post("/api/v1/auth/register", authHandler.Register) // Register the route

			// Setup mock expectations
			tc.setupMock(mockAuthService, tc.input)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1) // -1 disables timeout
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body (if expected)
			if tc.expectedBody != nil {
				var responseBody map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				assert.NoError(t, err, "Failed to decode response body")

				// For validation errors, we only check success and message, not the data field
				if tc.name == "Validation Error - Missing Username" {
					assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
					assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
				} else {
					assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
				}
			}

			// Verify that all expected mock calls were made
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	// --- Test Cases ---
	tests := []struct {
		name           string
		input          models.LoginUserInput
		setupMock      func(mockService *mocks.MockAuthService, input models.LoginUserInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.LoginUserInput{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.LoginUserInput) {
				mockService.On("LoginUser", mock.Anything, &input).Return("valid.jwt.token", nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Login successful",
				"data":    map[string]interface{}{"token": "valid.jwt.token"},
			},
		},
		{
			name: "Validation Error - Missing Password",
			input: models.LoginUserInput{
				Username: "testuser",
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.LoginUserInput) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Validation failed",
			},
		},
		{
			name: "Service Error - Invalid Credentials",
			input: models.LoginUserInput{
				Username: "testuser",
				Password: "wrongpassword",
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.LoginUserInput) {
				mockService.On("LoginUser", mock.Anything, &input).Return("", service.ErrInvalidCredentials).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": service.ErrInvalidCredentials.Error(),
			},
		},
		{
			name: "Service Error - Generic Internal Error",
			input: models.LoginUserInput{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(mockService *mocks.MockAuthService, input models.LoginUserInput) {
				mockService.On("LoginUser", mock.Anything, &input).Return("", errors.New("some internal error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Login process failed",
			},
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			app := fiber.New()
			mockAuthService := mocks.NewMockAuthService(t)
			authHandler := handlers.NewAuthHandler(mockAuthService)
			app.Post("/api/v1/auth/login", authHandler.Login) // Register the route

			// Setup mock expectations
			tc.setupMock(mockAuthService, tc.input)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Assert status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Assert response body
			if tc.expectedBody != nil {
				var responseBody map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				assert.NoError(t, err, "Failed to decode response body")

				// For validation errors, only check success and message
				if tc.name == "Validation Error - Missing Password" {
					assert.Equal(t, tc.expectedBody["success"], responseBody["success"])
					assert.Equal(t, tc.expectedBody["message"], responseBody["message"])
				} else {
					assert.Equal(t, tc.expectedBody, responseBody, "Response body mismatch")
				}
			}

			// Verify mock calls
			mockAuthService.AssertExpectations(t)
		})
	}
}
