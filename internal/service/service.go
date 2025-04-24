// internal/service/service.go
package service

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
)

// Service Layer Interfaces define the business logic operations.
// Handlers will depend on these interfaces, not directly on repositories.

// TaskService defines operations related to task logic (assignment, verification).
type TaskService interface {
	// VerifyTask handles the business logic for a parent verifying a submitted task.
	// It ensures atomicity for status updates and point transactions.
	VerifyTask(ctx context.Context, userTaskID int, parentID int, newStatus models.UserTaskStatus) error
	// AssignTask(ctx context.Context, parentID int, childID int, taskID int) (int, error) // Contoh metode lain
}

// RewardService defines operations related to reward logic (claiming).
type RewardService interface {
	// ClaimReward handles the logic for a child claiming a reward.
	// It checks points, creates the claim, and deducts points atomically.
	ClaimReward(ctx context.Context, childID int, rewardID int) (int, error) // Returns claim ID

	// ReviewClaim handles the logic for a parent reviewing a reward claim.
	// Ensures atomicity for status updates and point deductions (if approved).
	ReviewClaim(ctx context.Context, claimID int, parentID int, newStatus models.UserRewardStatus) error
}

type AuthService interface {
	// RegisterUser handles the business logic for creating a new user account,
	// including role validation and password hashing.
	RegisterUser(ctx context.Context, input *models.RegisterUserInput) (int, error) // Returns new User ID

	// LoginUser handles user authentication by verifying credentials and generating a JWT.
	LoginUser(ctx context.Context, input *models.LoginUserInput) (string, error) // Returns JWT token string
}

// UserService defines operations related to user profile management and potentially other user actions.
type UserService interface {
	// GetUserProfile retrieves user details (excluding sensitive info like password hash).
	GetUserProfile(ctx context.Context, userID int) (*models.User, error)

	// UpdateUserProfile handles updating non-sensitive user profile data.
	UpdateUserProfile(ctx context.Context, userID int, input *models.UpdateProfileInput) error

	// ChangePassword handles updating the user's password after verifying the old one.
	ChangePassword(ctx context.Context, userID int, input *models.UpdatePasswordInput) error

	// CreateChildAccount handles creating a child user and linking it to the parent atomically.
	CreateChildAccount(ctx context.Context, parentID int, input *models.CreateChildInput) (int, error)

	// // DeleteUser handles the business logic for deleting a user account.
	// DeleteUser(ctx context.Context, userID int, requestedByID int) error // Contoh untuk nanti
}

// InvitationService defines operations related to managing invitation codes
// for linking parents to children.
type InvitationService interface {
	// GenerateAndStoreCode creates a unique invitation code for a specific child,
	// initiated by an existing parent. It stores the code and returns it.
	// The requestingParentID is used to validate permission.
	// Returns the generated code string or an error.
	GenerateAndStoreCode(ctx context.Context, requestingParentID int, childID int) (string, error)

	// AcceptInvitation allows a logged-in parent (joiningParentID) to use an
	// invitation code to establish a relationship with the child linked to the code.
	// This operation should be atomic (uses database transactions).
	// Returns an error if the code is invalid, expired, already used,
	// or if the relationship cannot be added.
	AcceptInvitation(ctx context.Context, joiningParentID int, code string) error
}

// PointService defines operations related to points (bisa digabung atau terpisah).
// type PointService interface {
// 	AdjustPointsManual(ctx context.Context, adminOrParentID int, childID int, amount int, notes string) error
// }

// Anda bisa menambahkan interface service lain sesuai kebutuhan (e.g., RelationshipService).
