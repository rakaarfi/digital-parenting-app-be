package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockUserRelationshipRepository is a mock type for the UserRelationshipRepository type
type MockUserRelationshipRepository struct {
	mock.Mock
}

// AddRelationship provides a mock function with given fields: ctx, parentID, childID
func (_m *MockUserRelationshipRepository) AddRelationship(ctx context.Context, parentID int, childID int) error {
	ret := _m.Called(ctx, parentID, childID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) error); ok {
		r0 = rf(ctx, parentID, childID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetChildrenByParentID provides a mock function with given fields: ctx, parentID
func (_m *MockUserRelationshipRepository) GetChildrenByParentID(ctx context.Context, parentID int) ([]models.User, error) {
	ret := _m.Called(ctx, parentID)

	var r0 []models.User
	if rf, ok := ret.Get(0).(func(context.Context, int) []models.User); ok {
		r0 = rf(ctx, parentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, parentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetParentsByChildID provides a mock function with given fields: ctx, childID
func (_m *MockUserRelationshipRepository) GetParentsByChildID(ctx context.Context, childID int) ([]models.User, error) {
	ret := _m.Called(ctx, childID)

	var r0 []models.User
	if rf, ok := ret.Get(0).(func(context.Context, int) []models.User); ok {
		r0 = rf(ctx, childID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, childID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsParentOf provides a mock function with given fields: ctx, parentID, childID
func (_m *MockUserRelationshipRepository) IsParentOf(ctx context.Context, parentID int, childID int) (bool, error) {
	ret := _m.Called(ctx, parentID, childID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, int, int) bool); ok {
		r0 = rf(ctx, parentID, childID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int) error); ok {
		r1 = rf(ctx, parentID, childID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveRelationship provides a mock function with given fields: ctx, parentID, childID
func (_m *MockUserRelationshipRepository) RemoveRelationship(ctx context.Context, parentID int, childID int) error {
	ret := _m.Called(ctx, parentID, childID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) error); ok {
		r0 = rf(ctx, parentID, childID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HasSharedChild provides a mock function with given fields: ctx, parentID1, parentID2
func (_m *MockUserRelationshipRepository) HasSharedChild(ctx context.Context, parentID1 int, parentID2 int) (bool, error) {
	ret := _m.Called(ctx, parentID1, parentID2)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, int, int) bool); ok {
		r0 = rf(ctx, parentID1, parentID2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int) error); ok {
		r1 = rf(ctx, parentID1, parentID2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsParentOfTx provides a mock function with given fields: ctx, tx, parentID, childID
func (_m *MockUserRelationshipRepository) IsParentOfTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) (bool, error) {
	ret := _m.Called(ctx, tx, parentID, childID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, int) bool); ok {
		r0 = rf(ctx, tx, parentID, childID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int, int) error); ok {
		r1 = rf(ctx, tx, parentID, childID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddRelationshipTx provides a mock function with given fields: ctx, tx, parentID, childID
func (_m *MockUserRelationshipRepository) AddRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error {
	ret := _m.Called(ctx, tx, parentID, childID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, int) error); ok {
		r0 = rf(ctx, tx, parentID, childID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetParentIDsByChildIDTx provides a mock function with given fields: ctx, tx, childID
func (_m *MockUserRelationshipRepository) GetParentIDsByChildIDTx(ctx context.Context, tx pgx.Tx, childID int) ([]int, error) {
	ret := _m.Called(ctx, tx, childID)

	var r0 []int
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int) []int); ok {
		r0 = rf(ctx, tx, childID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int) error); ok {
		r1 = rf(ctx, tx, childID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasSharedChildTx provides a mock function with given fields: ctx, tx, parentID1, parentID2
func (_m *MockUserRelationshipRepository) HasSharedChildTx(ctx context.Context, tx pgx.Tx, parentID1 int, parentID2 int) (bool, error) {
	ret := _m.Called(ctx, tx, parentID1, parentID2)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, int) bool); ok {
		r0 = rf(ctx, tx, parentID1, parentID2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int, int) error); ok {
		r1 = rf(ctx, tx, parentID1, parentID2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockUserRelationshipRepository creates a new instance of MockUserRelationshipRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUserRelationshipRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserRelationshipRepository {
	mock := &MockUserRelationshipRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
