// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Queue is an autogenerated mock type for the Queue type
type Queue struct {
	mock.Mock
}

// Publish provides a mock function with given fields: ctx, topicName, response
func (_m *Queue) Publish(ctx context.Context, topicName string, response interface{}) {
	_m.Called(ctx, topicName, response)
}

type mockConstructorTestingTNewQueue interface {
	mock.TestingT
	Cleanup(func())
}

// NewQueue creates a new instance of Queue. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewQueue(t mockConstructorTestingTNewQueue) *Queue {
	mock := &Queue{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
