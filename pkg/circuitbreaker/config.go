package circuitbreaker

import (
	"sync"
	"time"
)

// CircuitBreakerConfig holds the configuration parameters for a circuit breaker.
// It includes the failure threshold, success threshold for half-open state,
// open timeout duration, maximum timeout duration, and manual override configuration.
// The manual override allows setting a default state for the circuit breaker that can
// override the automatic state transitions.
type CircuitBreakerConfig struct {
	FailureThreshold   int            // The number of failures required to open the circuit breaker.
	HalfOpenSuccess    int            // The number of successful requests needed to transition from half-open to closed.
	OpenTimeout        time.Duration  // The duration the circuit breaker stays open before transitioning to half-open.
	MaxTimeout         time.Duration  // The maximum duration for which the circuit breaker can stay open.
	OverrideConfigured bool           // Indicates if the override is configured
	ManualOverride     ManualOverride // The default manual override state (e.g., NORMAL, FORCE_OPEN, FORCE_CLOSED).
}

var (
	// inboundConfigurations stores the circuit breaker configurations for inbound requests.
	// It maps endpoint identifiers to their respective CircuitBreakerConfig.
	inboundConfigurations = make(map[string]CircuitBreakerConfig)

	// outboundConfigurations stores the circuit breaker configurations for outbound requests.
	// It maps endpoint identifiers to their respective CircuitBreakerConfig.
	outboundConfigurations = make(map[string]CircuitBreakerConfig)

	// configMu is a mutex used to synchronize access to the circuit breaker configurations.
	// It ensures that read and write operations on inboundConfigurations and outboundConfigurations are thread-safe.
	configMu sync.Mutex
)

// DefaultCircuitBreakerConfig returns the default configuration for a circuit breaker.
// TODO: Verify the values
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 20,
		HalfOpenSuccess:  10,
		OpenTimeout:      time.Minute * 5,
		ManualOverride:   NORMAL, // Default manual override state

	}
}
