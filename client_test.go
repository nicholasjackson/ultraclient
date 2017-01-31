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
var mockStats MockStats
var urls = []url.URL{url.URL{Host: "something:3232"}, url.URL{Host: "somethingelse:2323"}}
var urlIndex = 0

var getURL GetEndpoint = func() url.URL {
	url := urls[urlIndex]

	if urlIndex == 1 {
		urlIndex = 0
	} else {
		urlIndex = 1
	}

	return url
}

func setupClient(retryCount int) {
	urlIndex = 0

	loadbalancingStrategy = MockLoadbalancingStrategy{}
	loadbalancingStrategy.On("SetEndpoints", mock.Anything)
	loadbalancingStrategy.On("NextEndpoint").Return(getURL)
	loadbalancingStrategy.On("GetEndpoints").Return(urls)
	loadbalancingStrategy.On("Length").Return(len(urls))

	var retries []time.Duration
	for i := 0; i < retryCount; i++ {
		retries = append(retries, 1*time.Millisecond)
	}

	backoffStrategy = MockBackoffStrategy{}
	backoffStrategy.On("Create", mock.Anything, mock.Anything).
		Return(retries)

	mockStats = MockStats{}
	mockStats.On("Timing", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockStats.On("Increment", mock.Anything, mock.Anything, mock.Anything)

	client = NewClient(
		Config{
			RetryDelay:             100 * time.Millisecond,
			Retries:                retryCount,
			Timeout:                10 * time.Millisecond,
			ErrorPercentThreshold:  50,
			DefaultVolumeThreshold: 2,
			StatsD: StatsD{
				Enabled: true,
				Prefix:  "myapp",
				Tags:    []string{"production"},
			},
		},
		&loadbalancingStrategy,
		&backoffStrategy,
	)

	client.RegisterStats(&mockStats)
}

func TestNewRailsSessionSetsRetriesToURLsLengthIfNotSet(t *testing.T) {
	setupClient(0)
	c := NewClient(
		Config{RetryDelay: 100 * time.Millisecond},
		&loadbalancingStrategy,
		&backoffStrategy,
	)

	assert.Equal(t, 1, c.config.Retries)
}

func TestNewRailsSessionSetsRetriesIfSet(t *testing.T) {
	setupClient(0)
	c := NewClient(
		Config{Retries: 3, RetryDelay: 100 * time.Millisecond},
		&loadbalancingStrategy,
		&backoffStrategy,
	)

	assert.Equal(t, 3, c.config.Retries)
}

func TestDoCallsCommand(t *testing.T) {
	setupClient(0)

	callCount := 0

	client.Do(func(endpoint url.URL) error {
		callCount++
		return nil
	})

	assert.Equal(t, 1, callCount)
}

func TestClientCallsLoadBalancer(t *testing.T) {
	setupClient(0)

	client.Do(func(endpoint url.URL) error {
		return nil
	})

	loadbalancingStrategy.AssertCalled(t, "NextEndpoint")
}

func TestClientCallIncrementsStats(t *testing.T) {
	setupClient(0)
	client.Do(func(endpoint url.URL) error {
		return nil
	})

	mockStats.AssertCalled(t,
		"Increment",
		"myapp.something_3232.called", client.config.StatsD.Tags, mock.Anything)
}

func TestClientCallTimingStats(t *testing.T) {
	setupClient(0)
	client.Do(func(endpoint url.URL) error {
		return nil
	})

	mockStats.AssertCalled(t,
		"Timing",
		"myapp.something_3232.timing", client.config.StatsD.Tags, mock.Anything, mock.Anything)
}

func TestClientRetriesWithDifferentURLAndReturnsError(t *testing.T) {
	setupClient(2)

	var urls []url.URL
	err := client.Do(func(endpoint url.URL) error {
		urls = append(urls, endpoint)
		return fmt.Errorf("aaah")
	})

	clientError := err.(ClientError)

	assert.Equal(t, 3, len(urls))
	assert.Equal(t, 3, len(clientError.Errors()))
}

func TestSuccessIncrementsStats(t *testing.T) {
	setupClient(0)
	client.Do(func(endpoint url.URL) error {
		return nil
	})

	mockStats.AssertCalled(t,
		"Increment",
		"myapp.something_3232.success", client.config.StatsD.Tags, mock.Anything)
}

func TestTimeoutReturnsError(t *testing.T) {
	setupClient(0)

	err := client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	clientError := err.(ClientError)

	assert.Equal(t, ErrorTimeout, clientError.Errors()[0].Error())
}

func TestTimeoutIncrementsStats(t *testing.T) {
	setupClient(0)
	client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	mockStats.AssertCalled(t,
		"Increment",
		"myapp.something_3232.timeout", client.config.StatsD.Tags, mock.Anything)
}

func TestOpenCircuitReturnsError(t *testing.T) {
	setupClient(2)

	err := client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	clientError := err.(ClientError)

	assert.Equal(t, ErrorTimeout, clientError.Errors()[0].Error())
	assert.Equal(t, ErrorTimeout, clientError.Errors()[1].Error())
	assert.Equal(t, ErrorCircuitOpen, clientError.Errors()[2].Error())
}

func TestOpenCircuitIncrementsStats(t *testing.T) {
	setupClient(2)
	client.Do(func(endpoint url.URL) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	mockStats.AssertCalled(t,
		"Increment",
		"myapp.something_3232.circuitopen", client.config.StatsD.Tags, mock.Anything)
}
