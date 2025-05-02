package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock type for the UserRepository type
type MockUserRepository struct {
	mock.Mock
}

// CreateUser provides a mock function with given fields: ctx, user, hashedPassword
func (_m *MockUserRepository) CreateUser(ctx context.Context, user *models.RegisterUserInput, hashedPassword string) (int, error) {
	ret := _m.Called(ctx, user, hashedPassword)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, *models.RegisterUserInput, string) int); ok {
		r0 = rf(ctx, user, hashedPassword)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.RegisterUserInput, string) error); ok {
		r1 = rf(ctx, user, hashedPassword)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByUsername provides a mock function with given fields: ctx, username
func (_m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ret := _m.Called(ctx, username)

	var r0 *models.User
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.User); ok {
		r0 = rf(ctx, username)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, username)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByID provides a mock function with given fields: ctx, id
func (_m *MockUserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.User
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.User); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
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

// DeleteUserByID provides a mock function with given fields: ctx, id
func (_m *MockUserRepository) DeleteUserByID(ctx context.Context, id int) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAllUsers provides a mock function with given fields: ctx, page, limit
func (_m *MockUserRepository) GetAllUsers(ctx context.Context, page int, limit int) ([]models.User, int, error) {
	ret := _m.Called(ctx, page, limit)

	var r0 []models.User
	if rf, ok := ret.Get(0).(func(context.Context, int, int) []models.User); ok {
		r0 = rf(ctx, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.User)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, int) int); ok {
		r1 = rf(ctx, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, int) error); ok {
		r2 = rf(ctx, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateUserByID provides a mock function with given fields: ctx, id, input
func (_m *MockUserRepository) UpdateUserByID(ctx context.Context, id int, input *models.AdminUpdateUserInput) error {
	ret := _m.Called(ctx, id, input)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, *models.AdminUpdateUserInput) error); ok {
		r0 = rf(ctx, id, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateUserPassword provides a mock function with given fields: ctx, id, hashedPassword
func (_m *MockUserRepository) UpdateUserPassword(ctx context.Context, id int, hashedPassword string) error {
	ret := _m.Called(ctx, id, hashedPassword)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, string) error); ok {
		r0 = rf(ctx, id, hashedPassword)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateUserProfile provides a mock function with given fields: ctx, id, input
func (_m *MockUserRepository) UpdateUserProfile(ctx context.Context, id int, input *models.UpdateProfileInput) error {
	ret := _m.Called(ctx, id, input)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, *models.UpdateProfileInput) error); ok {
		r0 = rf(ctx, id, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateUserTx provides a mock function with given fields: ctx, tx, user, hashedPassword
func (_m *MockUserRepository) CreateUserTx(ctx context.Context, tx pgx.Tx, user *models.RegisterUserInput, hashedPassword string) (int, error) {
	ret := _m.Called(ctx, tx, user, hashedPassword)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, *models.RegisterUserInput, string) int); ok {
		r0 = rf(ctx, tx, user, hashedPassword)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, *models.RegisterUserInput, string) error); ok {
		r1 = rf(ctx, tx, user, hashedPassword)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockUserRepository creates a new instance of MockUserRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUserRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserRepository {
	mock := &MockUserRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
