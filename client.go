package loadbalancer

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/plugins"
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
}

// Client is a loadbalancing client with configurable backoff and loadbalancing strategy,
// Client also implements circuit breaking for fail fast.
type Client struct {
	config                Config
	loadbalancingStrategy LoadbalancingStrategy
	backoffStrategy       BackoffStrategy
	retry                 *retrier.Retrier
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

func (c *Client) doRequest(work WorkFunc) error {
	endpoint := c.loadbalancingStrategy.NextEndpoint()

	err := hystrix.Do(endpoint.String(), func() error {
		return work(endpoint)
	}, nil)

	switch err {
	case hystrix.ErrTimeout:
		return fmt.Errorf(ErrorTimeout)
	case hystrix.ErrCircuitOpen:
		return fmt.Errorf(ErrorCircuitOpen)
	case nil:
		return nil
	default:
		return err
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

	client.retry = retrier.New(backoffStrategy.Create(config.Retries, config.RetryDelay), nil)

	if config.StatsD.Enabled {
		c, err := plugins.InitializeStatsdCollector(&plugins.StatsdCollectorConfig{
			StatsdAddr: config.StatsD.Server,
			Prefix:     config.StatsD.Prefix,
		})
		if err != nil {
			log.Fatalf("could not initialize statsd client: %v", err)
		}

		metricCollector.Registry.Register(c.NewStatsdCollector)
	}

	return client
}
