package mocks

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock type for the UserService type
type MockUserService struct {
	mock.Mock
}

// GetUserProfile provides a mock function with given fields: ctx, userID
func (_m *MockUserService) GetUserProfile(ctx context.Context, userID int) (*models.User, error) {
	ret := _m.Called(ctx, userID)

	var r0 *models.User
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.User); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateUserProfile provides a mock function with given fields: ctx, userID, input
func (_m *MockUserService) UpdateUserProfile(ctx context.Context, userID int, input *models.UpdateProfileInput) error {
	ret := _m.Called(ctx, userID, input)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, *models.UpdateProfileInput) error); ok {
		r0 = rf(ctx, userID, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ChangePassword provides a mock function with given fields: ctx, userID, input
func (_m *MockUserService) ChangePassword(ctx context.Context, userID int, input *models.UpdatePasswordInput) error {
	ret := _m.Called(ctx, userID, input)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, *models.UpdatePasswordInput) error); ok {
		r0 = rf(ctx, userID, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateChildAccount provides a mock function with given fields: ctx, parentID, input
func (_m *MockUserService) CreateChildAccount(ctx context.Context, parentID int, input *models.CreateChildInput) (int, error) {
	ret := _m.Called(ctx, parentID, input)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, int, *models.CreateChildInput) int); ok {
		r0 = rf(ctx, parentID, input)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, *models.CreateChildInput) error); ok {
		r1 = rf(ctx, parentID, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockUserService creates a new instance of MockUserService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUserService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserService {
	mock := &MockUserService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
