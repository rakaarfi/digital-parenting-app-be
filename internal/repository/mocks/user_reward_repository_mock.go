package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockUserRewardRepository is a mock type for the UserRewardRepository type
type MockUserRewardRepository struct {
	mock.Mock
}

// CreateClaim provides a mock function with given fields: ctx, userID, rewardID, pointsDeducted
func (_m *MockUserRewardRepository) CreateClaim(ctx context.Context, userID int, rewardID int, pointsDeducted int) (int, error) {
	ret := _m.Called(ctx, userID, rewardID, pointsDeducted)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) int); ok {
		r0 = rf(ctx, userID, rewardID, pointsDeducted)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) error); ok {
		r1 = rf(ctx, userID, rewardID, pointsDeducted)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserRewardByID provides a mock function with given fields: ctx, id
func (_m *MockUserRewardRepository) GetUserRewardByID(ctx context.Context, id int) (*models.UserReward, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.UserReward
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.UserReward); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.UserReward)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetClaimsByChildID provides a mock function with given fields: ctx, childID, statusFilter, page, limit
func (_m *MockUserRewardRepository) GetClaimsByChildID(ctx context.Context, childID int, statusFilter string, page int, limit int) ([]models.UserReward, int, error) {
	ret := _m.Called(ctx, childID, statusFilter, page, limit)

	var r0 []models.UserReward
	if rf, ok := ret.Get(0).(func(context.Context, int, string, int, int) []models.UserReward); ok {
		r0 = rf(ctx, childID, statusFilter, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.UserReward)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, string, int, int) int); ok {
		r1 = rf(ctx, childID, statusFilter, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, string, int, int) error); ok {
		r2 = rf(ctx, childID, statusFilter, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetPendingClaimsByParentID provides a mock function with given fields: ctx, parentID, page, limit
func (_m *MockUserRewardRepository) GetPendingClaimsByParentID(ctx context.Context, parentID int, page int, limit int) ([]models.UserReward, int, error) {
	ret := _m.Called(ctx, parentID, page, limit)

	var r0 []models.UserReward
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) []models.UserReward); ok {
		r0 = rf(ctx, parentID, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.UserReward)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) int); ok {
		r1 = rf(ctx, parentID, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, int, int) error); ok {
		r2 = rf(ctx, parentID, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateClaimStatus provides a mock function with given fields: ctx, id, newStatus, reviewerID
func (_m *MockUserRewardRepository) UpdateClaimStatus(ctx context.Context, id int, newStatus models.UserRewardStatus, reviewerID int) error {
	ret := _m.Called(ctx, id, newStatus, reviewerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, models.UserRewardStatus, int) error); ok {
		r0 = rf(ctx, id, newStatus, reviewerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateClaimTx provides a mock function with given fields: ctx, tx, userID, rewardID, pointsDeducted
func (_m *MockUserRewardRepository) CreateClaimTx(ctx context.Context, tx pgx.Tx, userID int, rewardID int, pointsDeducted int) (int, error) {
	ret := _m.Called(ctx, tx, userID, rewardID, pointsDeducted)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, int, int) int); ok {
		r0 = rf(ctx, tx, userID, rewardID, pointsDeducted)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int, int, int) error); ok {
		r1 = rf(ctx, tx, userID, rewardID, pointsDeducted)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateClaimStatusTx provides a mock function with given fields: ctx, tx, id, newStatus, reviewerID
func (_m *MockUserRewardRepository) UpdateClaimStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserRewardStatus, reviewerID int) error {
	ret := _m.Called(ctx, tx, id, newStatus, reviewerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, models.UserRewardStatus, int) error); ok {
		r0 = rf(ctx, tx, id, newStatus, reviewerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetClaimDetailsForReviewTx provides a mock function with given fields: ctx, tx, claimID
func (_m *MockUserRewardRepository) GetClaimDetailsForReviewTx(ctx context.Context, tx pgx.Tx, claimID int) (*repository.ClaimReviewDetails, error) {
	ret := _m.Called(ctx, tx, claimID)

	var r0 *repository.ClaimReviewDetails
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int) *repository.ClaimReviewDetails); ok {
		r0 = rf(ctx, tx, claimID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*repository.ClaimReviewDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int) error); ok {
		r1 = rf(ctx, tx, claimID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockUserRewardRepository creates a new instance of MockUserRewardRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUserRewardRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserRewardRepository {
	mock := &MockUserRewardRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
