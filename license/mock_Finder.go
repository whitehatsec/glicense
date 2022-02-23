// Code generated by mockery v1.0.0. DO NOT EDIT.

package license

import context "context"
import mock "github.com/stretchr/testify/mock"
import module "github.com/whitehatsec/glicense/module"

// MockFinder is an autogenerated mock type for the Finder type
type MockFinder struct {
	mock.Mock
}

// License provides a mock function with given fields: _a0, _a1
func (_m *MockFinder) License(_a0 context.Context, _a1 module.Module) (*License, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *License
	if rf, ok := ret.Get(0).(func(context.Context, module.Module) *License); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*License)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, module.Module) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
