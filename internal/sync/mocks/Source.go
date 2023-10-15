// Code generated by mockery v2.34.2. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/inovex/CalendarSync/internal/models"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Source is an autogenerated mock type for the Source type
type Source struct {
	mock.Mock
}

type Source_Expecter struct {
	mock *mock.Mock
}

func (_m *Source) EXPECT() *Source_Expecter {
	return &Source_Expecter{mock: &_m.Mock}
}

// EventsInTimeframe provides a mock function with given fields: ctx, start, end
func (_m *Source) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	ret := _m.Called(ctx, start, end)

	var r0 []models.Event
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time, time.Time) ([]models.Event, error)); ok {
		return rf(ctx, start, end)
	}
	if rf, ok := ret.Get(0).(func(context.Context, time.Time, time.Time) []models.Event); ok {
		r0 = rf(ctx, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Event)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time, time.Time) error); ok {
		r1 = rf(ctx, start, end)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Source_EventsInTimeframe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EventsInTimeframe'
type Source_EventsInTimeframe_Call struct {
	*mock.Call
}

// EventsInTimeframe is a helper method to define mock.On call
//   - ctx context.Context
//   - start time.Time
//   - end time.Time
func (_e *Source_Expecter) EventsInTimeframe(ctx interface{}, start interface{}, end interface{}) *Source_EventsInTimeframe_Call {
	return &Source_EventsInTimeframe_Call{Call: _e.mock.On("EventsInTimeframe", ctx, start, end)}
}

func (_c *Source_EventsInTimeframe_Call) Run(run func(ctx context.Context, start time.Time, end time.Time)) *Source_EventsInTimeframe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(time.Time), args[2].(time.Time))
	})
	return _c
}

func (_c *Source_EventsInTimeframe_Call) Return(_a0 []models.Event, _a1 error) *Source_EventsInTimeframe_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Source_EventsInTimeframe_Call) RunAndReturn(run func(context.Context, time.Time, time.Time) ([]models.Event, error)) *Source_EventsInTimeframe_Call {
	_c.Call.Return(run)
	return _c
}

// GetCalendarID provides a mock function with given fields:
func (_m *Source) GetCalendarID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Source_GetCalendarID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCalendarID'
type Source_GetCalendarID_Call struct {
	*mock.Call
}

// GetCalendarID is a helper method to define mock.On call
func (_e *Source_Expecter) GetCalendarID() *Source_GetCalendarID_Call {
	return &Source_GetCalendarID_Call{Call: _e.mock.On("GetCalendarID")}
}

func (_c *Source_GetCalendarID_Call) Run(run func()) *Source_GetCalendarID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Source_GetCalendarID_Call) Return(_a0 string) *Source_GetCalendarID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Source_GetCalendarID_Call) RunAndReturn(run func() string) *Source_GetCalendarID_Call {
	_c.Call.Return(run)
	return _c
}

// Name provides a mock function with given fields:
func (_m *Source) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Source_Name_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Name'
type Source_Name_Call struct {
	*mock.Call
}

// Name is a helper method to define mock.On call
func (_e *Source_Expecter) Name() *Source_Name_Call {
	return &Source_Name_Call{Call: _e.mock.On("Name")}
}

func (_c *Source_Name_Call) Run(run func()) *Source_Name_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Source_Name_Call) Return(_a0 string) *Source_Name_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Source_Name_Call) RunAndReturn(run func() string) *Source_Name_Call {
	_c.Call.Return(run)
	return _c
}

// NewSource creates a new instance of Source. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSource(t interface {
	mock.TestingT
	Cleanup(func())
}) *Source {
	mock := &Source{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
