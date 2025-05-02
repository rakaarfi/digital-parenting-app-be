package mocks

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockRewardService struct {
	mock.Mock
}

func (m *MockRewardService) CreateReward(ctx context.Context, input *models.CreateRewardInput) (int, error) {
	args := m.Called(ctx, input)
	return args.Int(0), args.Error(1)
}

func (m *MockRewardService) GetRewardByID(ctx context.Context, id int) (*models.Reward, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockRewardService) UpdateReward(ctx context.Context, id int, input *models.UpdateRewardInput) error {
	args := m.Called(ctx, id, input)
	return args.Error(0)
}

func (m *MockRewardService) DeleteReward(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRewardService) ClaimReward(ctx context.Context, childID int, rewardID int) (int, error) {
	args := m.Called(ctx, childID, rewardID)
	return args.Int(0), args.Error(1)
}

func (m *MockRewardService) ReviewClaim(ctx context.Context, claimID int, parentID int, newStatus models.UserRewardStatus) error {
	args := m.Called(ctx, claimID, parentID, newStatus)
	return args.Error(0)
}
