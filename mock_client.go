package ultraclient

import "github.com/stretchr/testify/mock"

// MockClient implements a mock implementation of the ultraclient
type MockClient struct {
	mock.Mock
}

// Do is the mock execution of the Do method
func (m *MockClient) Do(work WorkFunc) error {
	args := m.Called(work)
	return args.Error(0)
}

// Clone is the mock execution of the Clone method, returns self
func (m *MockClient) Clone() Client {
	m.Called()
	return m
}

// RegisterStats is the mock execution of the RegisterStats method
func (m *MockClient) RegisterStats(stats Stats) {
	m.Called(stats)
}
