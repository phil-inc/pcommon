package circuitbreaker

import (
	"sync"
	"time"
)

type CircuitBreakerConfig struct {
	FailureThreshold int
	HalfOpenSuccess  int
	OpenTimeout      time.Duration
}

var (
	inboundConfigurations  = make(map[string]CircuitBreakerConfig)
	outboundConfigurations = make(map[string]CircuitBreakerConfig)
	configMu               sync.Mutex
)

// DefaultCircuitBreakerConfig returns default configuration with rate limiting.
// TODO: Verify the values
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 20,
		HalfOpenSuccess:  10,
		OpenTimeout:      time.Minute * 5,
	}
}
