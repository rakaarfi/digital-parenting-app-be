package mocks

import (
    "context"
    "github.com/rakaarfi/digital-parenting-app-be/internal/models"
    "github.com/stretchr/testify/mock"
)

type MockTaskService struct {
    mock.Mock
}

func (m *MockTaskService) CreateTask(ctx context.Context, input *models.CreateTaskInput) (int, error) {
    args := m.Called(ctx, input)
    return args.Int(0), args.Error(1)
}

func (m *MockTaskService) GetTaskByID(ctx context.Context, id int) (*models.Task, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskService) UpdateTask(ctx context.Context, id int, input *models.UpdateTaskInput) error {
    args := m.Called(ctx, id, input)
    return args.Error(0)
}

func (m *MockTaskService) DeleteTask(ctx context.Context, id int) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *MockTaskService) VerifyTask(ctx context.Context, taskID int, parentID int, status models.UserTaskStatus) error {
    args := m.Called(ctx, taskID, parentID, status)
    return args.Error(0)
}

func (m *MockTaskService) SubmitTask(ctx context.Context, taskID int, childID int) error {
    args := m.Called(ctx, taskID, childID)
    return args.Error(0)
}