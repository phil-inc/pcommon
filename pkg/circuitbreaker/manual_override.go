package circuitbreaker

import "fmt"

// ManualOverride defines the possible states for manual override of the circuit breaker.
type ManualOverride string

const (
	// NORMAL indicates no manual override is configured.
	NORMAL ManualOverride = "NORMAL"
	// ForceOpen forces the circuit breaker to the OPEN state.
	ForceOpen ManualOverride = "FORCE_OPEN"
	// ForceClosed forces the circuit breaker to the CLOSED state.
	ForceClosed ManualOverride = "FORCE_CLOSED"
)

// checkManualOverride checks if a manual override is configured for the circuit breaker.
// If an override is set, it returns the overridden state. Otherwise, it returns the current state.
func (cb *BaseCircuitBreaker) checkManualOverride() State {
	if cb.overrideConfigured {
		switch cb.manualOverride {
		case ForceOpen:
			return Open
		case ForceClosed:
			return Closed
		}
	}

	return cb.state
}

// SetManualOverride configures the circuit breaker to use a manual override state.
// This method sets the override state, marks the override as configured, and saves the state to Redis.
// Returns an error if saving the state to Redis fails.
func (cb *BaseCircuitBreaker) SetManualOverride(state ManualOverride) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.manualOverride = state
	cb.overrideConfigured = true

	if err := cb.saveState(); err != nil {
		return fmt.Errorf("failed to save manual override state: %w", err)
	}
	return nil
}

// ResetManualOverride resets the manual override to NORMAL and marks it as not configured.
// This method disables any manual override, restores the circuit breaker to its default behavior,
// and saves the updated state to Redis.
// Returns an error if saving the state to Redis fails.
func (cb *BaseCircuitBreaker) ResetManualOverride() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.manualOverride = NORMAL
	cb.overrideConfigured = false

	if err := cb.saveState(); err != nil {
		return fmt.Errorf("failed to save manual override reset state: %w", err)
	}
	return nil
}
