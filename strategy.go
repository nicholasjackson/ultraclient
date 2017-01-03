package loadbalancer

import "net/url"

// Strategy is an interface to be implemented by loadbalancing
// strategies like round robin or random.
type Strategy interface {
	NextEndpoint() url.URL
	SetEndpoints([]url.URL)
}
