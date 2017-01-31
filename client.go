package loadbalancer

import (
	"fmt"
	"net/url"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/eapache/go-resiliency/retrier"
)

// WorkFunc defines the work function to be passed to the Client.Do method
type WorkFunc func(endpoint url.URL) error

// Config defines the configuration for the Client
type Config struct {
	// Timeout is the length of time to wait before the work function times out
	Timeout time.Duration

	// MaxConcurrentRequests is the maximum number of work requests which can be
	// active at anyone time.
	MaxConcurrentRequests int

	// ErrorPercentThreshold is the percentage of work requests which result in
	// error before the circuit opens.
	ErrorPercentThreshold int

	//
	DefaultVolumeThreshold int

	// Retries is the number of times a request should be attempted before
	// returning.
	Retries int

	// RetryDelay is the default amount of time to wait before retrying the
	// work.
	RetryDelay time.Duration

	// Endpoints which are passed to the loadbalancing strategy and then to the
	// work function.
	Endpoints []url.URL

	// Enable statsd metrixs for client
	StatsD StatsD
}

// StatsD is the configuration for the StatsD endpoint
type StatsD struct {
	Enabled bool
	Server  string
	Prefix  string
	Tags    []string
}

// Client is a loadbalancing client with configurable backoff and loadbalancing strategy,
// Client also implements circuit breaking for fail fast.
type Client struct {
	config                Config
	loadbalancingStrategy LoadbalancingStrategy
	backoffStrategy       BackoffStrategy
	retry                 *retrier.Retrier
	statsCollection       []Stats
}

// Do perfoms the work for the client, the WorkFunc passed as a parameter
// contains all your business logic to execute.  The url from the loadbalancer is injected
// into the provided function.
// var response MyResponse
// clientError := client.Do(func(endpoint url.URL) error {
//   resp, err := http.DefaultClient.Get(endpoint)
//   if err != nil {
//     return err
//   }
//
//   response = resp // set the outer variable
//   return nil
// }
func (c *Client) Do(work WorkFunc) error {
	var clientError ClientError

	c.retry.Run(func() error {
		if requestErr := c.doRequest(work); requestErr != nil {
			clientError.AddError(requestErr)
			return requestErr
		}

		return nil
	})

	if len(clientError.errors) > 0 {
		return clientError
	}

	return nil
}

// RegisterStats registers a stats interface with the client, multiple interfaces can
// be registered with a single client
func (c *Client) RegisterStats(stats Stats) {
	c.statsCollection = append(c.statsCollection, stats)
}

func (c *Client) doRequest(work WorkFunc) error {
	endpoint := c.loadbalancingStrategy.NextEndpoint()

	c.incrementStats(&endpoint, StatsCalled)
	startTime := time.Now()
	defer c.timingStats(&endpoint, time.Now().Sub(startTime), StatsTiming)

	err := hystrix.Do(endpoint.String(), func() error {
		return work(endpoint)
	}, nil)

	return c.handleError(&endpoint, err)
}

func (c *Client) handleError(endpoint *url.URL, err error) error {
	switch err {
	case hystrix.ErrTimeout:
		c.incrementStats(endpoint, StatsTimeout)
		return fmt.Errorf(ErrorTimeout)
	case hystrix.ErrCircuitOpen:
		c.incrementStats(endpoint, StatsCircuitOpen)
		return fmt.Errorf(ErrorCircuitOpen)
	case nil:
		c.incrementStats(endpoint, StatsSuccess)
		return nil
	default:
		return err
	}
}

func (c *Client) timingStats(endpoint *url.URL, duration time.Duration, action string) {
	bucket := fmt.Sprintf("%v.%v.%v",
		c.config.StatsD.Prefix,
		PrettyPrintURL(endpoint),
		action)

	for _, stats := range c.statsCollection {
		stats.Timing(bucket, c.config.StatsD.Tags, duration, 1)
	}
}

func (c *Client) incrementStats(endpoint *url.URL, action string) {
	bucket := fmt.Sprintf("%v.%v.%v",
		c.config.StatsD.Prefix,
		PrettyPrintURL(endpoint),
		action)

	for _, stats := range c.statsCollection {
		stats.Increment(bucket, c.config.StatsD.Tags, 1)
	}
}

// NewClient creates a new instance of the loadbalancing client
func NewClient(
	config Config,
	loadbalancingStrategy LoadbalancingStrategy,
	backoffStrategy BackoffStrategy) *Client {
	client := &Client{
		config:                config,
		loadbalancingStrategy: loadbalancingStrategy,
		backoffStrategy:       backoffStrategy,
	}

	if config.Retries < 1 {
		client.config.Retries = loadbalancingStrategy.Length() - 1
	}

	loadbalancingStrategy.SetEndpoints(config.Endpoints)

	for _, url := range loadbalancingStrategy.GetEndpoints() {
		hystrix.ConfigureCommand(url.String(), hystrix.CommandConfig{
			Timeout:                int(config.Timeout / time.Millisecond),
			MaxConcurrentRequests:  config.MaxConcurrentRequests,
			ErrorPercentThreshold:  config.ErrorPercentThreshold,
			RequestVolumeThreshold: config.DefaultVolumeThreshold,
		})
	}

	client.retry = retrier.New(backoffStrategy.Create(client.config.Retries, client.config.RetryDelay), nil)

	client.statsCollection = make([]Stats, 0)

	return client
}
