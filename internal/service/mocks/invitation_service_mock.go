package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockInvitationService struct {
	mock.Mock
}

func (m *MockInvitationService) GenerateAndStoreCode(ctx context.Context, parentID int, childID int) (string, error) {
	args := m.Called(ctx, parentID, childID)
	return args.String(0), args.Error(1)
}

func (m *MockInvitationService) AcceptInvitation(ctx context.Context, joiningParentID int, code string) error {
	args := m.Called(ctx, joiningParentID, code)
	return args.Error(0)
}
