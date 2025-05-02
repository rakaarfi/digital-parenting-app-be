package mocks

import (
	"context"

	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockTaskRepository is a mock type for the TaskRepository type
type MockTaskRepository struct {
	mock.Mock
}

// CreateTask provides a mock function with given fields: ctx, task
func (_m *MockTaskRepository) CreateTask(ctx context.Context, task *models.Task) (int, error) {
	ret := _m.Called(ctx, task)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, *models.Task) int); ok {
		r0 = rf(ctx, task)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *models.Task) error); ok {
		r1 = rf(ctx, task)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTaskByID provides a mock function with given fields: ctx, id
func (_m *MockTaskRepository) GetTaskByID(ctx context.Context, id int) (*models.Task, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Task
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.Task); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Task)
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

// GetTasksByCreatorID provides a mock function with given fields: ctx, creatorID, page, limit
func (_m *MockTaskRepository) GetTasksByCreatorID(ctx context.Context, creatorID int, page int, limit int) ([]models.Task, int, error) {
	ret := _m.Called(ctx, creatorID, page, limit)

	var r0 []models.Task
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) []models.Task); ok {
		r0 = rf(ctx, creatorID, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Task)
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

// UpdateTask provides a mock function with given fields: ctx, task, parentID
func (_m *MockTaskRepository) UpdateTask(ctx context.Context, task *models.Task, parentID int) error {
	ret := _m.Called(ctx, task, parentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Task, int) error); ok {
		r0 = rf(ctx, task, parentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteTask provides a mock function with given fields: ctx, id, parentID
func (_m *MockTaskRepository) DeleteTask(ctx context.Context, id int, parentID int) error {
	ret := _m.Called(ctx, id, parentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) error); ok {
		r0 = rf(ctx, id, parentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockTaskRepository creates a new instance of MockTaskRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTaskRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTaskRepository {
	mock := &MockTaskRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
