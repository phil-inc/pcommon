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
	configurations = make(map[string]CircuitBreakerConfig)
	configMu       sync.Mutex
)

// SetCircuitBreakerConfig sets the configuration for a given URL.
func SetCircuitBreakerConfig(url string, config CircuitBreakerConfig) {
	configMu.Lock()
	defer configMu.Unlock()
	configurations[url] = config
}

// GetCircuitBreakerConfig retrieves the configuration for a given URL.
func GetCircuitBreakerConfig(url string) CircuitBreakerConfig {
	configMu.Lock()
	defer configMu.Unlock()

	config, exists := configurations[url]
	if !exists {
		// Default settings if not configured
		return CircuitBreakerConfig{
			FailureThreshold: 20,
			HalfOpenSuccess:  10,
			OpenTimeout:      time.Minute * 5,
		}
	}

	return config
}
