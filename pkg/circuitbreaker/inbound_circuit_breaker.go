package circuitbreaker

func NewInboundCircuitBreaker(endpoint string) *BaseCircuitBreaker {
	return newCircuitBreaker(endpoint, true)
}

// SetInboundCircuitBreakerConfig sets the configuration for inbound calls.
func SetInboundCircuitBreakerConfig(endpoint string, config CircuitBreakerConfig) {
	configMu.Lock()
	defer configMu.Unlock()
	inboundConfigurations[endpoint] = config
}

// GetInboundCircuitBreakerConfig retrieves the configuration for inbound calls.
func GetInboundCircuitBreakerConfig(endpoint string) CircuitBreakerConfig {
	configMu.Lock()
	defer configMu.Unlock()
	// TODO: Normalize the endpoint
	config, exists := inboundConfigurations[endpoint]
	if !exists {
		return DefaultCircuitBreakerConfig()
	}
	return config
}
