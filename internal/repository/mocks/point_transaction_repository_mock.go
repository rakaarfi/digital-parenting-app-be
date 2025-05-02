package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockPointTransactionRepository is a mock type for the PointTransactionRepository type
type MockPointTransactionRepository struct {
	mock.Mock
}

// CreateTransaction provides a mock function with given fields: ctx, txData
func (_m *MockPointTransactionRepository) CreateTransaction(ctx context.Context, txData *models.PointTransaction) error {
	ret := _m.Called(ctx, txData)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.PointTransaction) error); ok {
		r0 = rf(ctx, txData)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetTransactionsByUserID provides a mock function with given fields: ctx, userID, page, limit
func (_m *MockPointTransactionRepository) GetTransactionsByUserID(ctx context.Context, userID int, page int, limit int) ([]models.PointTransaction, int, error) {
	ret := _m.Called(ctx, userID, page, limit)

	var r0 []models.PointTransaction
	if rf, ok := ret.Get(0).(func(context.Context, int, int, int) []models.PointTransaction); ok {
		r0 = rf(ctx, userID, page, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PointTransaction)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(context.Context, int, int, int) int); ok {
		r1 = rf(ctx, userID, page, limit)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int, int, int) error); ok {
		r2 = rf(ctx, userID, page, limit)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CalculateTotalPointsByUserID provides a mock function with given fields: ctx, userID
func (_m *MockPointTransactionRepository) CalculateTotalPointsByUserID(ctx context.Context, userID int) (int, error) {
	ret := _m.Called(ctx, userID)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, int) int); ok {
		r0 = rf(ctx, userID)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateTransactionTx provides a mock function with given fields: ctx, tx, txData
func (_m *MockPointTransactionRepository) CreateTransactionTx(ctx context.Context, tx pgx.Tx, txData *models.PointTransaction) error {
	ret := _m.Called(ctx, tx, txData)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, *models.PointTransaction) error); ok {
		r0 = rf(ctx, tx, txData)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CalculateTotalPointsByUserIDTx provides a mock function with given fields: ctx, tx, userID
func (_m *MockPointTransactionRepository) CalculateTotalPointsByUserIDTx(ctx context.Context, tx pgx.Tx, userID int) (int, error) {
	ret := _m.Called(ctx, tx, userID)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx, int) int); ok {
		r0 = rf(ctx, tx, userID)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx, int) error); ok {
		r1 = rf(ctx, tx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockPointTransactionRepository creates a new instance of MockPointTransactionRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockPointTransactionRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockPointTransactionRepository {
	mock := &MockPointTransactionRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
