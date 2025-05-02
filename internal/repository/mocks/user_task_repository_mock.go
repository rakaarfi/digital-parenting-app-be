package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockUserTaskRepository is a mock type for the UserTaskRepository type
type MockUserTaskRepository struct {
	mock.Mock
}

// AssignTask provides a mock function with given fields: ctx, userID, taskID, assignedByID
func (_m *MockUserTaskRepository) AssignTask(ctx context.Context, userID int, taskID int, assignedByID int) (int, error) {
	ret := _m.Called(ctx, userID, taskID, assignedByID)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) int); ok {
		r0 = rf(ctx, userID, taskID, assignedByID)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) error); ok {
		r1 = rf(ctx, userID, taskID, assignedByID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserTaskByID provides a mock function with given fields: ctx, id
func (_m *MockUserTaskRepository) GetUserTaskByID(ctx context.Context, id int) (*models.UserTask, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.UserTask
	if rf, ok := ret.Get(0).(func(context.Context, int) *models.UserTask); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.UserTask)
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

// GetTasksByChildID provides a mock function with given fields: ctx, childID, statusFilter, page, limit
func (_m *MockUserTaskRepository) GetTasksByChildID(ctx context.Context, childID int, statusFilter string, page int, limit int) ([]models.UserTask, int, error) {
	ret := _m.Called(ctx, childID, statusFilter, page, limit)

	var r0 []models.UserTask
	if rf, ok := ret.Get(0).(func(context.Context, int, string, int, int) []models.UserTask); ok {
		r0 = rf(ctx, childID, statusFilter, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.UserTask)
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

// GetTasksByParentID provides a mock function with given fields: ctx, parentID, statusFilter, page, limit
func (_m *MockUserTaskRepository) GetTasksByParentID(ctx context.Context, parentID int, statusFilter string, page int, limit int) ([]models.UserTask, int, error) {
	ret := _m.Called(ctx, parentID, statusFilter, page, limit)

	var r0 []models.UserTask
	if rf, ok := ret.Get(0).(func(context.Context, int, string, int, int) []models.UserTask); ok {
		r0 = rf(ctx, parentID, statusFilter, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.UserTask)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, string, int, int) int); ok {
		r1 = rf(ctx, parentID, statusFilter, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, string, int, int) error); ok {
		r2 = rf(ctx, parentID, statusFilter, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateUserTaskStatus provides a mock function with given fields: ctx, id, newStatus, verifierID
func (_m *MockUserTaskRepository) UpdateUserTaskStatus(ctx context.Context, id int, newStatus models.UserTaskStatus, verifierID *int) error {
	ret := _m.Called(ctx, id, newStatus, verifierID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, models.UserTaskStatus, *int) error); ok {
		r0 = rf(ctx, id, newStatus, verifierID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SubmitTask provides a mock function with given fields: ctx, id, childID
func (_m *MockUserTaskRepository) SubmitTask(ctx context.Context, id int, childID int) error {
	ret := _m.Called(ctx, id, childID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) error); ok {
		r0 = rf(ctx, id, childID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// VerifyTask provides a mock function with given fields: ctx, id, parentID, newStatus
func (_m *MockUserTaskRepository) VerifyTask(ctx context.Context, id int, parentID int, newStatus models.UserTaskStatus) (*models.UserTask, error) {
	ret := _m.Called(ctx, id, parentID, newStatus)

	var r0 *models.UserTask
	if rf, ok := ret.Get(0).(func(context.Context, int, int, models.UserTaskStatus) *models.UserTask); ok {
		r0 = rf(ctx, id, parentID, newStatus)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.UserTask)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int, models.UserTaskStatus) error); ok {
		r1 = rf(ctx, id, parentID, newStatus)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckExistingActiveTask provides a mock function with given fields: ctx, userID, taskID
func (_m *MockUserTaskRepository) CheckExistingActiveTask(ctx context.Context, userID int, taskID int) (bool, error) {
	ret := _m.Called(ctx, userID, taskID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, int, int) bool); ok {
		r0 = rf(ctx, userID, taskID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, int) error); ok {
		r1 = rf(ctx, userID, taskID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTaskDetailsForVerificationTx provides a mock function with given fields: ctx, tx, userTaskID
func (_m *MockUserTaskRepository) GetTaskDetailsForVerificationTx(ctx context.Context, tx pgx.Tx, userTaskID int) (*repository.TaskVerificationDetails, error) {
	ret := _m.Called(ctx, tx, userTaskID)

	var r0 *repository.TaskVerificationDetails
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int) *repository.TaskVerificationDetails); ok {
		r0 = rf(ctx, tx, userTaskID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*repository.TaskVerificationDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int) error); ok {
		r1 = rf(ctx, tx, userTaskID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateStatusTx provides a mock function with given fields: ctx, tx, id, newStatus, verifierID
func (_m *MockUserTaskRepository) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, newStatus models.UserTaskStatus, verifierID int) error {
	ret := _m.Called(ctx, tx, id, newStatus, verifierID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int, models.UserTaskStatus, int) error); ok {
		r0 = rf(ctx, tx, id, newStatus, verifierID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockUserTaskRepository creates a new instance of MockUserTaskRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUserTaskRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserTaskRepository {
	mock := &MockUserTaskRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
