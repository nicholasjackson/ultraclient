package loadbalancer

import (
	"net/url"
	"time"
)

// LoadBalancingStrategy is an interface to be implemented by loadbalancing
// strategies like round robin or random.
type LoadbalancingStrategy interface {
	NextEndpoint() url.URL
	SetEndpoints([]url.URL)
	GetEndpoints() []url.URL
	Length() int
}

type BackoffStrategy interface {
	Create(retries int, delay time.Duration) []time.Duration
}
