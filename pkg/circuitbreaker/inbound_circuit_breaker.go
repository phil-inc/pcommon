package circuitbreaker

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
