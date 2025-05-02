package mocks

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock type for the AuthService type
type MockAuthService struct {
	mock.Mock
}

// RegisterUser provides a mock function with given fields: ctx, input
func (_m *MockAuthService) RegisterUser(ctx context.Context, input *models.RegisterUserInput) (int, error) {
	ret := _m.Called(ctx, input)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, *models.RegisterUserInput) int); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.RegisterUserInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoginUser provides a mock function with given fields: ctx, input
func (_m *MockAuthService) LoginUser(ctx context.Context, input *models.LoginUserInput) (string, error) {
	ret := _m.Called(ctx, input)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, *models.LoginUserInput) string); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.LoginUserInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockAuthService creates a new instance of MockAuthService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAuthService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAuthService {
	mock := &MockAuthService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
