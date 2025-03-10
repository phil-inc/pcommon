package circuitbreaker

import "net/http"

// NewInboundCircuitBreaker creates and initializes a new BaseCircuitBreaker instance
// for the specified inbound endpoint. It sets up the circuit breaker with configurations
// retrieved from the inbound configuration map and loads its initial state from Redis.
// Returns the newly created BaseCircuitBreaker instance.
func NewInboundCircuitBreaker(endpoint string) *BaseCircuitBreaker {
	return newCircuitBreaker(endpoint, true)
}

// SetInboundCircuitBreakerConfig sets the configuration for the circuit breaker associated with inbound calls
// for a specific endpoint. It updates or adds the configuration in the `inboundConfigurations` map.
// The configuration is locked during the update to ensure thread safety.
// Parameters:
//   - endpoint: The endpoint URL for which the configuration is being set.
//   - config: The CircuitBreakerConfig object containing the configuration settings.
func SetInboundCircuitBreakerConfig(endpoint string, config CircuitBreakerConfig) {
	configMu.Lock()
	defer configMu.Unlock()

	endpoint = normalizeURL(endpoint)
	inboundConfigurations[endpoint] = config
}

// GetInboundCircuitBreakerConfig retrieves the configuration for the circuit breaker associated with inbound calls
// for a specific endpoint. It returns the configuration from the `inboundConfigurations` map if it exists,
// or a default configuration if no configuration is found for the endpoint.
// The configuration is locked during the retrieval to ensure thread safety.
// Parameters:
//   - endpoint: The endpoint URL for which the configuration is being retrieved.
//
// Returns:
//   - The CircuitBreakerConfig object for the specified endpoint. If no configuration is found, a default configuration is returned.
func GetInboundCircuitBreakerConfig(endpoint string) CircuitBreakerConfig {
	configMu.Lock()
	defer configMu.Unlock()

	endpoint = normalizeURL(endpoint)
	config, exists := inboundConfigurations[endpoint]
	if !exists {
		return DefaultCircuitBreakerConfig()
	}
	return config
}

// CircuitBreakerMiddleware returns a middleware function that applies circuit breaker logic to incoming HTTP requests.
// It creates a new circuit breaker instance for each request based on the request URL and checks the circuit breaker state.
// If the circuit breaker is in the OPEN state, it responds with a 503 Service Unavailable error.
// Otherwise, it proceeds with the request handling.
func CircuitBreakerMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			url := r.URL.RequestURI()
			cb := NewInboundCircuitBreaker(url)

			cb.mu.Lock()
			state := cb.state
			cb.mu.Unlock()

			if state == Open {
				// Return a custom response indicating the circuit is open
				http.Error(w, "Service temporarily unavailable. Please try again later.", http.StatusServiceUnavailable)
				return
			}

			if next != nil {
				// Proceed with the request handling if the circuit is not open
				next.ServeHTTP(w, r)
			}

		})
	}
}
