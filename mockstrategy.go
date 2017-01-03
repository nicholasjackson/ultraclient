package loadbalancer

import (
	"net/url"

	"github.com/stretchr/testify/mock"
)

// MockStrategy is a mocked implementation of the Strategy interface for testing
// Usage:
// mock := MockStrategy{}
// mock.On("NextEndpoint").Returns([]url.URL{url.URL{Host: ""}})
// mock.On("SetEndpoints", mock.Anything)
// mock.AssertCalled(t, "NextEndpoint")
type MockStrategy struct {
	mock.Mock
}

// NextEndpoint returns the next endpoint in the list
func (m *MockStrategy) NextEndpoint() url.URL {
	args := m.Called()
	return args.Get(0).(url.URL)
}

// SetEndpoints sets the mocks internal register with the given arguments,
// this method can not be used to update the return values in NextEndpoint.
func (m *MockStrategy) SetEndpoints(urls []url.URL) {
	m.Called(urls)
}
