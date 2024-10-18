package circuitbreaker

// NewOutboundCircuitBreaker creates and initializes a new BaseCircuitBreaker instance
// for the specified outbound endpoint. It sets up the circuit breaker with configurations
// retrieved from the outbound configuration map and loads its initial state from Redis.
// Returns the newly created BaseCircuitBreaker instance.
func NewOutboundCircuitBreaker(endpoint string) *BaseCircuitBreaker {
	return newCircuitBreaker(endpoint, false)
}

// SetOutboundCircuitBreakerConfig sets the configuration for the circuit breaker associated with outbound calls
// for a specific endpoint. It updates or adds the configuration in the `outboundConfigurations` map.
// The configuration is locked during the update to ensure thread safety.
// Parameters:
//   - endpoint: The endpoint URL for which the configuration is being set.
//   - config: The CircuitBreakerConfig object containing the configuration settings.
func SetOutboundCircuitBreakerConfig(endpoint string, config CircuitBreakerConfig) {
	configMu.Lock()
	defer configMu.Unlock()

	endpoint = normalizeURL(endpoint)
	outboundConfigurations[endpoint] = config
}

// GetOutboundCircuitBreakerConfig retrieves the configuration for the circuit breaker associated with outbound calls
// for a specific endpoint. It returns the configuration from the `outboundConfigurations` map if it exists,
// or a default configuration if no configuration is found for the endpoint.
// The configuration is locked during the retrieval to ensure thread safety.
// Parameters:
//   - endpoint: The endpoint URL for which the configuration is being retrieved.
//
// Returns:
//   - The CircuitBreakerConfig object for the specified endpoint. If no configuration is found, a default configuration is returned.
func GetOutboundCircuitBreakerConfig(endpoint string) CircuitBreakerConfig {
	configMu.Lock()
	defer configMu.Unlock()

	endpoint = normalizeURL(endpoint)
	config, exists := outboundConfigurations[endpoint]
	if !exists {
		return DefaultCircuitBreakerConfig()
	}
	return config
}
