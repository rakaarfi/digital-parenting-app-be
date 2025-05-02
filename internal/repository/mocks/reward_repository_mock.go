package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockRewardRepository is a mock type for the RewardRepository type
type MockRewardRepository struct {
	mock.Mock
}

// CreateReward provides a mock function with given fields: ctx, reward
func (_m *MockRewardRepository) CreateReward(ctx context.Context, reward *models.Reward) (int, error) {
	ret := _m.Called(ctx, reward)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, *models.Reward) int); ok {
		r0 = rf(ctx, reward)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.Reward) error); ok {
		r1 = rf(ctx, reward)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRewardByID provides a mock function with given fields: ctx, id
func (_m *MockRewardRepository) GetRewardByID(ctx context.Context, id int) (*models.Reward, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Reward
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.Reward); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Reward)
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

// GetRewardsByCreatorID provides a mock function with given fields: ctx, creatorID, page, limit
func (_m *MockRewardRepository) GetRewardsByCreatorID(ctx context.Context, creatorID int, page int, limit int) ([]models.Reward, int, error) {
	ret := _m.Called(ctx, creatorID, page, limit)

	var r0 []models.Reward
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) []models.Reward); ok {
		r0 = rf(ctx, creatorID, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Reward)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) int); ok {
		r1 = rf(ctx, creatorID, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, int, int) error); ok {
		r2 = rf(ctx, creatorID, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetAvailableRewardsForChild provides a mock function with given fields: ctx, childID, page, limit
func (_m *MockRewardRepository) GetAvailableRewardsForChild(ctx context.Context, childID int, page int, limit int) ([]models.Reward, int, error) {
	ret := _m.Called(ctx, childID, page, limit)

	var r0 []models.Reward
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) []models.Reward); ok {
		r0 = rf(ctx, childID, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Reward)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) int); ok {
		r1 = rf(ctx, childID, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, int, int) error); ok {
		r2 = rf(ctx, childID, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateReward provides a mock function with given fields: ctx, reward, parentID
func (_m *MockRewardRepository) UpdateReward(ctx context.Context, reward *models.Reward, parentID int) error {
	ret := _m.Called(ctx, reward, parentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Reward, int) error); ok {
		r0 = rf(ctx, reward, parentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteReward provides a mock function with given fields: ctx, id, parentID
func (_m *MockRewardRepository) DeleteReward(ctx context.Context, id int, parentID int) error {
	ret := _m.Called(ctx, id, parentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) error); ok {
		r0 = rf(ctx, id, parentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetRewardDetailsTx provides a mock function with given fields: ctx, tx, rewardID
func (_m *MockRewardRepository) GetRewardDetailsTx(ctx context.Context, tx pgx.Tx, rewardID int) (*repository.RewardDetails, error) {
	ret := _m.Called(ctx, tx, rewardID)

	var r0 *repository.RewardDetails
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int) *repository.RewardDetails); ok {
		r0 = rf(ctx, tx, rewardID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*repository.RewardDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int) error); ok {
		r1 = rf(ctx, tx, rewardID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockRewardRepository creates a new instance of MockRewardRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRewardRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRewardRepository {
	mock := &MockRewardRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
