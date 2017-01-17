package loadbalancer

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var client *Client
var loadbalancingStrategy MockLoadbalancingStrategy
var backoffStrategy MockBackoffStrategy
var urls = []url.URL{url.URL{Host: "something"}, url.URL{Host: "somethingelse"}}
var urlIndex = 0

var getURL GetEndpoint = func() url.URL {
	url := urls[urlIndex]
	urlIndex++

	return url
}

func setupClient() {
	urlIndex = 0

	loadbalancingStrategy = MockLoadbalancingStrategy{}
	loadbalancingStrategy.On("SetEndpoints", mock.Anything)
	loadbalancingStrategy.On("NextEndpoint").Return(getURL)
	loadbalancingStrategy.On("GetEndpoints").Return(urls)
	loadbalancingStrategy.On("Length").Return(len(urls))

	backoffStrategy = MockBackoffStrategy{}
	backoffStrategy.On("Create", mock.Anything, mock.Anything).
		Return([]time.Duration{1 * time.Millisecond})

	client = NewClient(
		Config{
			RetryDelay:             100 * time.Millisecond,
			Timeout:                10 * time.Millisecond,
			ErrorPercentThreshold:  50,
			DefaultVolumeThreshold: 1,
		},
		&loadbalancingStrategy,
		&backoffStrategy,
	)
}

func TestNewRailsSessionSetsRetriesToURLsLengthIfNotSet(t *testing.T) {
	setupClient()
	c := NewClient(
		Config{RetryDelay: 100 * time.Millisecond},
		&loadbalancingStrategy,
		&backoffStrategy,
	)

	assert.Equal(t, 1, c.config.Retries)
}

func TestNewRailsSessionSetsRetriesIfSet(t *testing.T) {
	setupClient()
	c := NewClient(
		Config{Retries: 3, RetryDelay: 100 * time.Millisecond},
		&loadbalancingStrategy,
		&backoffStrategy,
	)

	assert.Equal(t, 3, c.config.Retries)
}

func TestDoCallsCommand(t *testing.T) {
	setupClient()

	callCount := 0

	client.Do(func(endpoint url.URL) error {
		callCount++
		return nil
	})

	assert.Equal(t, 1, callCount)
}

func TestClientCallsLoadBalancer(t *testing.T) {
	setupClient()

	client.Do(func(endpoint url.URL) error {
		return nil
	})

	loadbalancingStrategy.AssertCalled(t, "NextEndpoint")
}

func TestClientRetriesWithDifferentURLAndReturnsError(t *testing.T) {
	setupClient()

	var urls []url.URL
	err := client.Do(func(endpoint url.URL) error {
		urls = append(urls, endpoint)
		return fmt.Errorf("aaah")
	})

	clientError := err.(ClientError)

	assert.Equal(t, 2, len(urls))
	assert.Equal(t, 2, len(clientError.Errors()))
}

func TestTimeoutReturnsError(t *testing.T) {
	setupClient()

	err := client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	clientError := err.(ClientError)

	assert.Equal(t, ErrorTimeout, clientError.Errors()[0].Error())
}

func TestOpenCircuitReturnsError(t *testing.T) {
	setupClient()

	err := client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	clientError := err.(ClientError)

	assert.Equal(t, ErrorCircuitOpen, clientError.Errors()[1].Error())
}
