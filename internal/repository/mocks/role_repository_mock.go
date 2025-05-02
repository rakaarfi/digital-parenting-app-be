package mocks

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockRoleRepository is a mock type for the RoleRepository type
type MockRoleRepository struct {
	mock.Mock
}

// CreateRole provides a mock function with given fields: ctx, role
func (_m *MockRoleRepository) CreateRole(ctx context.Context, role *models.Role) (int, error) {
	ret := _m.Called(ctx, role)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) int); ok {
		r0 = rf(ctx, role)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.Role) error); ok {
		r1 = rf(ctx, role)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRoleByID provides a mock function with given fields: ctx, id
func (_m *MockRoleRepository) GetRoleByID(ctx context.Context, id int) (*models.Role, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Role
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.Role); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
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

// GetAllRoles provides a mock function with given fields: ctx
func (_m *MockRoleRepository) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	ret := _m.Called(ctx)

	var r0 []models.Role
	if rf, ok := ret.Get(0).(func(context.Context) []models.Role); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Role)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateRole provides a mock function with given fields: ctx, role
func (_m *MockRoleRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	ret := _m.Called(ctx, role)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) error); ok {
		r0 = rf(ctx, role)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteRole provides a mock function with given fields: ctx, id
func (_m *MockRoleRepository) DeleteRole(ctx context.Context, id int) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockRoleRepository creates a new instance of MockRoleRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRoleRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRoleRepository {
	mock := &MockRoleRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
