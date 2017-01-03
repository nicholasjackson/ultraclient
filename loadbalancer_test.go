package loadbalancer

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	lb           *LoadBalancer
	mockStrategy MockStrategy
	urls         []url.URL
)

func TestNewLoadBalancerSetsUpCorrectly(t *testing.T) {
	setup()

	assert.Equal(t, &mockStrategy, lb.strategy)
	mockStrategy.AssertCalled(t, "SetEndpoints", urls)
}

func TestGetEndpointCallsStrategy(t *testing.T) {
	setup()

	lb.GetEndpoint()

	mockStrategy.AssertCalled(t, "NextEndpoint")
}

func TestUpdateEndpointCallsStrategy(t *testing.T) {
	setup()

	lb.UpdateEndpoints(urls)

	mockStrategy.AssertCalled(t, "SetEndpoints", urls)
}

func setup() {
	urls = []url.URL{url.URL{Host: "myserver.com"}}
	mockStrategy = MockStrategy{}
	mockStrategy.On("NextEndpoint").Return(urls[0])
	mockStrategy.On("SetEndpoints", mock.Anything)
	lb = NewLoadBalancer(&mockStrategy, urls)
}
