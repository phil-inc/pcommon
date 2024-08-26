package circuitbreaker

type ManualOverride string

const (
	NORMAL      ManualOverride = "NORMAL"
	ForceOpen   ManualOverride = "FORCE_OPEN"
	ForceClosed ManualOverride = "FORCE_CLOSED"
)

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

func (cb *BaseCircuitBreaker) SetManualOverride(state ManualOverride) {
	cb.manualOverride = state
	cb.overrideConfigured = true
}

func (cb *BaseCircuitBreaker) ResetManualOverride() {
	cb.manualOverride = NORMAL
	cb.overrideConfigured = false
}
