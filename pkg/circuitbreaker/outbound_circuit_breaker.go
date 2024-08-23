package circuitbreaker

func NewOutboundCircuitBreaker(endpoint string) *BaseCircuitBreaker {
	return newCircuitBreaker(endpoint, false)
}

// SetOutboundCircuitBreakerConfig sets the configuration for outbound calls.
func SetOutboundCircuitBreakerConfig(endpoint string, config CircuitBreakerConfig) {
	configMu.Lock()
	defer configMu.Unlock()

	endpoint = normalizeURL(endpoint)
	outboundConfigurations[endpoint] = config
}

// GetOutboundCircuitBreakerConfig retrieves the configuration for outbound calls.
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
