package loadbalancer

import (
	"math/rand"
	"net/url"
	"time"
)

// RandomStrategy implements Strategy for random endopoint selection
type RandomStrategy struct {
	endpoints []url.URL
}

// NextEndpoint returns an endpoint using a random strategy
func (r *RandomStrategy) NextEndpoint() url.URL {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	return r.endpoints[r1.Intn(len(r.endpoints))]
}

// SetEndpoints sets the available endpoints for use by the strategy
func (r *RandomStrategy) SetEndpoints(endpoints []url.URL) {
	r.endpoints = endpoints
}

func (r *RandomStrategy) GetEndpoints() []url.URL {
	return r.endpoints
}

func (r *RandomStrategy) Length() int {
	return len(r.endpoints)
}
