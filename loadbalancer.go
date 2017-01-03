package loadbalancer

import "net/url"

// NewLoadBalancer creates a new loadbalancer and setting the given strategy
func NewLoadBalancer(strategy Strategy, endpoints []url.URL) *LoadBalancer {
	strategy.SetEndpoints(endpoints)
	return &LoadBalancer{strategy: strategy}
}

// GetEndpoint gets an endpoint based on the given strategy
func (l *LoadBalancer) GetEndpoint() url.URL {
	return l.strategy.NextEndpoint()
}

// UpdateEndpoints updates the endpoints available to the strategy
func (l *LoadBalancer) UpdateEndpoints(urls []url.URL) {
	l.strategy.SetEndpoints(urls)
}
