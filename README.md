# UltraClient
Ultra client is a wrapper around exisiting packages to provide loadbalancing, circuit breaking and backoffs for networking code in go.  Rather than extend net/http you pass a function to ultraclient which accepts a url.URL, this allows you to use RPC based clients as well as net/http.

[![GoDoc](https://godoc.org/github.com/nicholasjackson/ultraclient?status.svg)](https://godoc.org/github.com/nicholasjackson/ultraclient)

[![CircleCI](https://circleci.com/gh/nicholasjackson/ultraclient.svg?style=svg)](https://circleci.com/gh/nicholasjackson/ultraclient)

## Usage
First create the ultraclient instance
```go
lb := ultraclient.RoundRobinStrategy{}
bs := ultraclient.ExponentialBackoff{}
stats, _ := ultraclient.NewDogStatsD(url.URL{Host:"statsd:8125"})
endpoints := []url.URL{
  url.URL{Host: "server1:8080"},
  url.URL{Host: "server2:8080"},
}

config := loadbalancer.Config{
	Timeout:                50 * time.Millisecond,
	MaxConcurrentRequests:  500,
	ErrorPercentThreshold:  25,
	DefaultVolumeThreshold: 10,
  Retries:                100*time.Millisecond,
  Endpoints: endpoints,
	StatsD: loadbalancer.StatsD{
		Prefix: "application.client",
	},
}

client := ultraclient.NewClient(client, &lb, &bs)
client.RegisterStats(stats)
```

Then you can use it like so, this example shows how to use ultraclient with the http.Client

```go

// When Do is called ultraclient will call the passed function with a url which has been returned from the loadbalancer
client.Do(func(uri string) error {
  resp, err := http.DefaultClient().Get(uri)
  if err != nil {
  	// If an error is returned ultraclient will re call the function based on the backoff and loadbalancer
	// configuration.
	// Should a particular uri raise so many errors that it open the circuit breaker then this uri will be 
	// removed from the load balancer and will not be used for future calls until the circuit half opens again.
  	return err
  }
  defer resp.Body.Close()
  ... do stuff
  return nil
})
```
