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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestParentHandler_CreateRewardDefinition(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		input          models.Reward
		setupMock      func(mockRepo *mocks.MockRewardRepository, input models.Reward, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			input: models.Reward{
				RewardName:        "New Reward",
				RewardPoint:       100,
				RewardDescription: "Description for the new reward",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, input models.Reward, parentID int) {
				mockRepo.On("CreateReward", mock.Anything, mock.AnythingOfType("*models.Reward")).Return(5, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward definition created",
				"data":    map[string]interface{}{"reward_id": float64(5)},
			},
		},
		{
			name: "Validation Error - Missing Required Fields",
			input: models.Reward{
				RewardName: "New Reward",
				// Missing RewardPoint
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, input models.Reward, parentID int) {
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
			name: "Database Error",
			input: models.Reward{
				RewardName:        "New Reward",
				RewardPoint:       100,
				RewardDescription: "Description for the new reward",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, input models.Reward, parentID int) {
				mockRepo.On("CreateReward", mock.Anything, mock.AnythingOfType("*models.Reward")).Return(0, errors.New("database error"))
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
			mockRewardRepo := new(mocks.MockRewardRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				RewardRepo: mockRewardRepo,
				Validate:   validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Post("/api/v1/parent/rewards", parentHandler.CreateRewardDefinition)

			// Setup mock expectations
			tc.setupMock(mockRewardRepo, tc.input, parentID)

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/rewards", bytes.NewReader(bodyBytes))
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
			mockRewardRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_GetMyRewardDefinitions(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockRewardRepository, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockRewardRepository, parentID int) {
				mockRewards := []models.Reward{
					{ID: 1, RewardName: "Reward 1", RewardPoint: 100, RewardDescription: "Description 1", CreatedByUserID: parentID},
					{ID: 2, RewardName: "Reward 2", RewardPoint: 200, RewardDescription: "Description 2", CreatedByUserID: parentID},
				}
				mockRepo.On("GetRewardsByCreatorID", mock.Anything, parentID, 1, 10).Return(mockRewards, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward definitions retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":                 float64(1),
						"reward_name":        "Reward 1",
						"reward_point":       float64(100),
						"reward_description": "Description 1",
						"created_by_user_id": float64(parentID),
					},
					map[string]interface{}{
						"id":                 float64(2),
						"reward_name":        "Reward 2",
						"reward_point":       float64(200),
						"reward_description": "Description 2",
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
			name: "No Rewards Found",
			setupMock: func(mockRepo *mocks.MockRewardRepository, parentID int) {
				mockRepo.On("GetRewardsByCreatorID", mock.Anything, parentID, 1, 10).Return([]models.Reward{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward definitions retrieved successfully",
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
			setupMock: func(mockRepo *mocks.MockRewardRepository, parentID int) {
				mockRepo.On("GetRewardsByCreatorID", mock.Anything, parentID, 1, 10).Return(nil, 0, errors.New("database error"))
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
			mockRewardRepo := new(mocks.MockRewardRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				RewardRepo: mockRewardRepo,
				Validate:   validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Get("/api/v1/parent/rewards", parentHandler.GetMyRewardDefinitions)

			// Setup mock expectations
			tc.setupMock(mockRewardRepo, parentID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/rewards", nil)

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

				// If there are rewards, check their details
				if len(expectedData) > 0 {
					for i, expected := range expectedData {
						expectedReward := expected.(map[string]interface{})
						actualReward := actualData[i].(map[string]interface{})
						assert.Equal(t, expectedReward["id"], actualReward["id"])
						assert.Equal(t, expectedReward["reward_name"], actualReward["reward_name"])
						assert.Equal(t, expectedReward["reward_point"], actualReward["reward_point"])
						assert.Equal(t, expectedReward["reward_description"], actualReward["reward_description"])
						assert.Equal(t, expectedReward["created_by_user_id"], actualReward["created_by_user_id"])
					}
				}
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockRewardRepo.AssertExpectations(t)
		})
	}
}

func TestChildHandler_GetAvailableRewards(t *testing.T) {
	childID := 1

	tests := []struct {
		name           string
		setupMock      func(mockRepo *mocks.MockRewardRepository, childID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *mocks.MockRewardRepository, childID int) {
				mockRewards := []models.Reward{
					{ID: 1, RewardName: "Reward 1", RewardPoint: 100, RewardDescription: "Description 1", CreatedByUserID: 2},
					{ID: 2, RewardName: "Reward 2", RewardPoint: 200, RewardDescription: "Description 2", CreatedByUserID: 3},
				}
				mockRepo.On("GetAvailableRewardsForChild", mock.Anything, childID, 1, 10).Return(mockRewards, 2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Available rewards retrieved successfully",
				"data": []interface{}{
					map[string]interface{}{
						"id":                 float64(1),
						"reward_name":        "Reward 1",
						"reward_point":       float64(100),
						"reward_description": "Description 1",
						"created_by_user_id": float64(2),
					},
					map[string]interface{}{
						"id":                 float64(2),
						"reward_name":        "Reward 2",
						"reward_point":       float64(200),
						"reward_description": "Description 2",
						"created_by_user_id": float64(3),
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
			name: "No Rewards Found",
			setupMock: func(mockRepo *mocks.MockRewardRepository, childID int) {
				mockRepo.On("GetAvailableRewardsForChild", mock.Anything, childID, 1, 10).Return([]models.Reward{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Available rewards retrieved successfully",
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
			setupMock: func(mockRepo *mocks.MockRewardRepository, childID int) {
				mockRepo.On("GetAvailableRewardsForChild", mock.Anything, childID, 1, 10).Return(nil, 0, errors.New("database error"))
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
			mockRewardRepo := new(mocks.MockRewardRepository)

			// Create a minimal ChildHandler with just what we need for this test
			childHandler := &handlers.ChildHandler{
				RewardRepo: mockRewardRepo,
				Validate:   validator.New(),
			}

			// Add JWT middleware to simulate a logged-in child user
			app.Use(test_utils.MockJWTMiddleware(childID, "child_user", "Child"))

			// Register the handler
			app.Get("/api/v1/child/rewards", childHandler.GetAvailableRewards)

			// Setup mock expectations
			tc.setupMock(mockRewardRepo, childID)

			// Prepare request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/child/rewards", nil)

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

				// If there are rewards, check their details
				if len(expectedData) > 0 {
					for i, expected := range expectedData {
						expectedReward := expected.(map[string]interface{})
						actualReward := actualData[i].(map[string]interface{})
						assert.Equal(t, expectedReward["id"], actualReward["id"])
						assert.Equal(t, expectedReward["reward_name"], actualReward["reward_name"])
						assert.Equal(t, expectedReward["reward_point"], actualReward["reward_point"])
						assert.Equal(t, expectedReward["reward_description"], actualReward["reward_description"])
						assert.Equal(t, expectedReward["created_by_user_id"], actualReward["created_by_user_id"])
					}
				}
			} else {
				assert.Equal(t, tc.expectedBody, result)
			}

			// Verify mock expectations
			mockRewardRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_UpdateMyRewardDefinition(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		rewardID       string
		input          models.UpdateRewardInput
		setupMock      func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:     "Success",
			rewardID: "1",
			input: models.UpdateRewardInput{
				RewardName:        "Updated Reward",
				RewardPoint:       150,
				RewardDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput) {
				// Mock GetRewardByID
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Original Reward",
					RewardPoint:       100,
					RewardDescription: "Original Description",
					CreatedByUserID:   parentID,
				}, nil)

				// Mock UpdateReward
				mockRepo.On("UpdateReward", mock.Anything, mock.AnythingOfType("*models.Reward"), parentID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward definition updated successfully",
			},
		},
		{
			name:     "Invalid Reward ID",
			rewardID: "invalid",
			input: models.UpdateRewardInput{
				RewardName:        "Updated Reward",
				RewardPoint:       150,
				RewardDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput) {
				// No mock calls expected
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
			input: models.UpdateRewardInput{
				RewardName:        "Updated Reward",
				RewardPoint:       150,
				RewardDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput) {
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Reward definition not found",
			},
		},
		{
			name:     "Not Reward Owner",
			rewardID: "2",
			input: models.UpdateRewardInput{
				RewardName:        "Updated Reward",
				RewardPoint:       150,
				RewardDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput) {
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Original Reward",
					RewardPoint:       100,
					RewardDescription: "Original Description",
					CreatedByUserID:   parentID + 1, // Different user ID
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You do not have permission to modify this reward.",
			},
		},
		{
			name:     "Database Error",
			rewardID: "1",
			input: models.UpdateRewardInput{
				RewardName:        "Updated Reward",
				RewardPoint:       150,
				RewardDescription: "Updated Description",
			},
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int, input models.UpdateRewardInput) {
				// Mock GetRewardByID
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Original Reward",
					RewardPoint:       100,
					RewardDescription: "Original Description",
					CreatedByUserID:   parentID,
				}, nil)

				// Mock UpdateReward with error
				mockRepo.On("UpdateReward", mock.Anything, mock.AnythingOfType("*models.Reward"), parentID).Return(errors.New("database error"))
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
			mockRewardRepo := new(mocks.MockRewardRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				RewardRepo: mockRewardRepo,
				Validate:   validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Patch("/api/v1/parent/rewards/:rewardId", parentHandler.UpdateMyRewardDefinition)

			// Setup mock expectations
			rewardID, _ := strconv.Atoi(tc.rewardID)
			if tc.name != "Invalid Reward ID" {
				tc.setupMock(mockRewardRepo, rewardID, parentID, tc.input)
			}

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.input)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/parent/rewards/"+tc.rewardID, bytes.NewReader(bodyBytes))
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
			mockRewardRepo.AssertExpectations(t)
		})
	}
}

func TestParentHandler_DeleteMyRewardDefinition(t *testing.T) {
	parentID := 1

	tests := []struct {
		name           string
		rewardID       string
		setupMock      func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:     "Success",
			rewardID: "1",
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				// Mock GetRewardByID
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Reward to Delete",
					RewardPoint:       100,
					RewardDescription: "Description",
					CreatedByUserID:   parentID,
				}, nil)

				// Mock DeleteReward
				mockRepo.On("DeleteReward", mock.Anything, rewardID, parentID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "Reward definition deleted successfully",
			},
		},
		{
			name:     "Invalid Reward ID",
			rewardID: "invalid",
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				// No mock calls expected
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
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Reward definition not found",
			},
		},
		{
			name:     "Not Reward Owner",
			rewardID: "2",
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Reward to Delete",
					RewardPoint:       100,
					RewardDescription: "Description",
					CreatedByUserID:   parentID + 1, // Different user ID
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "Forbidden: You can only delete rewards you created.",
			},
		},
		{
			name:     "Reward Currently Claimed",
			rewardID: "3",
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				// Mock GetRewardByID
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Reward to Delete",
					RewardPoint:       100,
					RewardDescription: "Description",
					CreatedByUserID:   parentID,
				}, nil)

				// Mock DeleteReward with error indicating reward is claimed
				mockRepo.On("DeleteReward", mock.Anything, rewardID, parentID).Return(errors.New("reward is currently claimed by a child"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"success": false,
				"message": "An internal error occurred",
			},
		},
		{
			name:     "Database Error",
			rewardID: "4",
			setupMock: func(mockRepo *mocks.MockRewardRepository, rewardID, parentID int) {
				// Mock GetRewardByID
				mockRepo.On("GetRewardByID", mock.Anything, rewardID).Return(&models.Reward{
					ID:                rewardID,
					RewardName:        "Reward to Delete",
					RewardPoint:       100,
					RewardDescription: "Description",
					CreatedByUserID:   parentID,
				}, nil)

				// Mock DeleteReward with generic error
				mockRepo.On("DeleteReward", mock.Anything, rewardID, parentID).Return(errors.New("database error"))
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
			mockRewardRepo := new(mocks.MockRewardRepository)

			// Create a minimal ParentHandler with just what we need for this test
			parentHandler := &handlers.ParentHandler{
				RewardRepo: mockRewardRepo,
				Validate:   validator.New(),
			}

			// Add JWT middleware to simulate a logged-in parent user
			app.Use(test_utils.MockJWTMiddleware(parentID, "parent_user", "Parent"))

			// Register the handler
			app.Delete("/api/v1/parent/rewards/:rewardId", parentHandler.DeleteMyRewardDefinition)

			// Setup mock expectations
			rewardID, _ := strconv.Atoi(tc.rewardID)
			if tc.name != "Invalid Reward ID" {
				tc.setupMock(mockRewardRepo, rewardID, parentID)
			}

			// Prepare request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/parent/rewards/"+tc.rewardID, nil)

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
			mockRewardRepo.AssertExpectations(t)
		})
	}
}
