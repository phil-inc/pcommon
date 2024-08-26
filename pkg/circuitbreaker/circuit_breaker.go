package circuitbreaker

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/phil-inc/pcommon/pkg/redis"
	logger "github.com/phil-inc/plog-ng/pkg/core"
)

var redisClient *redis.RedisClient
var ctx = context.Background() //TODO

// SetupRedis initializes the Redis client with the given URL.
// It must be called before using the circuit breaker functionality.
func SetupRedis(url string) {
	redisClient = new(redis.RedisClient)
	redisClient.SetupRedis(url)
}

// State represents the state of the circuit breaker.
// It can be one of Closed, Open, or HalfOpen.
type State string

const (
	// Closed indicates that the circuit breaker is in a stable state,
	// allowing all requests to pass through.
	Closed State = "CLOSED"

	// Open indicates that the circuit breaker is open and all requests
	// are blocked until it transitions to HalfOpen.
	Open State = "OPEN"

	// HalfOpen indicates that the circuit breaker is partially open,
	// allowing a limited number of requests to test if the issue has been resolved.
	HalfOpen State = "HALF_OPEN"
)

// CircuitBreaker defines the interface for a circuit breaker.
// It includes methods for handling requests, loading and saving state,
// transitioning between states, and recording failures.
type CircuitBreaker interface {
	HandleRequest(f func() ([]byte, error)) ([]byte, error)
	loadState() error
	saveState() error
	transitionToHalfOpen()
	transitionToClosed()
	recordFailure()
	transitionToOpen()
}

// BaseCircuitBreaker is a basic implementation of the CircuitBreaker interface.
// It maintains state, failure counts, and success counts, and manages transitions
// between states based on request outcomes and configurations.
type BaseCircuitBreaker struct {
	state                    State
	manualOverride           ManualOverride
	overrideConfigured       bool // Indicates if override is set
	failureThreshold         int
	failureCount             int
	successCount             int
	mu                       sync.RWMutex
	halfOpenSuccessThreshold int
	baseTimeout              time.Duration
	openTimeout              time.Duration
	maxTimeout               time.Duration
	lastFailureTime          time.Time
	endpoint                 string
	isInbound                bool // Indicates whether this is an inbound or outbound circuit breaker
}

// HandleRequest executes the given function while managing the circuit breaker state.
// It locks the circuit breaker, checks the state, and transitions if necessary.
// After unlocking, it calls the provided function and then locks again to record
// failures or successes and save the state.
// Returns the result of the function and any error encountered.
func (cb *BaseCircuitBreaker) HandleRequest(f func() ([]byte, error)) ([]byte, error) {
	cb.mu.Lock()

	switch cb.checkManualOverride() {
	case Open:
		if time.Since(cb.lastFailureTime) > cb.openTimeout {
			cb.transitionToHalfOpen()
		} else {
			cb.mu.Unlock()
			return nil, fmt.Errorf("circuit breaker is open")
		}
	case HalfOpen:
		if cb.successCount >= cb.halfOpenSuccessThreshold {
			cb.transitionToClosed()
		}
	}

	// Release lock before function call
	cb.mu.Unlock()

	resp, err := f()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return nil, err
	}

	if cb.state == HalfOpen {
		cb.successCount++
	}

	cb.saveState() // save state after success
	return resp, nil
}

// newCircuitBreaker creates and initializes a new BaseCircuitBreaker instance
// for the specified endpoint. It configures the circuit breaker based on whether
// it is inbound or outbound and loads the initial state from Redis.
func newCircuitBreaker(endpoint string, isInbound bool) *BaseCircuitBreaker {
	var config CircuitBreakerConfig

	if isInbound {
		config = GetInboundCircuitBreakerConfig(endpoint)
	} else {
		config = GetOutboundCircuitBreakerConfig(endpoint)
	}

	cb := &BaseCircuitBreaker{
		endpoint:                 endpoint,
		failureThreshold:         config.FailureThreshold,
		halfOpenSuccessThreshold: config.HalfOpenSuccess,
		openTimeout:              config.OpenTimeout,
		overrideConfigured:       config.OverrideConfigured,
		manualOverride:           config.ManualOverride, // Set the default manual override
		isInbound:                isInbound,
	}

	// Load initial state from Redis
	if err := cb.loadState(); err != nil {
		logger.Errorf("Error loading initial state: %v", err)
	}

	return cb
}

// loadState retrieves the current state of the circuit breaker from Redis.
func (cb *BaseCircuitBreaker) loadState() error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	keyPrefix := "outbound:" // Default prefix for outbound
	if cb.isInbound {
		keyPrefix = "inbound:"
	}

	state, err := redisClient.HGetAll(ctx, keyPrefix+cb.endpoint).Result()
	if err != nil {
		return fmt.Errorf("failed to load state from Redis: %v", err)
	}

	if len(state) > 0 {
		cb.state = State(state["state"])
		cb.failureCount = parseOrDefault(strconv.Atoi, state["failureCount"], cb.failureCount)
		cb.successCount = parseOrDefault(strconv.Atoi, state["successCount"], cb.successCount)
		cb.lastFailureTime = parseTimeOrDefault(time.RFC3339, state["lastFailureTime"], cb.lastFailureTime)
		cb.manualOverride = ManualOverride(state["manualOverride"])
		cb.overrideConfigured = parseOrDefault(strconv.ParseBool, state["overrideConfigured"], cb.overrideConfigured)
	}

	return nil
}

// saveState stores the current state of the circuit breaker, including the manual override,
// in Redis. It updates the state, failure count, success count, last failure time,
// manual override, and override configuration status.
func (cb *BaseCircuitBreaker) saveState() error {
	keyPrefix := "outbound:" // Default prefix for outbound
	if cb.isInbound {
		keyPrefix = "inbound:"
	}

	state := map[string]interface{}{
		"state":              string(cb.state),
		"failureCount":       cb.failureCount,
		"successCount":       cb.successCount,
		"lastFailureTime":    cb.lastFailureTime.Format(time.RFC3339),
		"manualOverride":     string(cb.manualOverride),
		"overrideConfigured": cb.overrideConfigured,
	}

	if err := redisClient.HSet(ctx, keyPrefix+cb.endpoint, state).Err(); err != nil {
		return err
	}

	return nil
}

// transitionToHalfOpen changes the state of the circuit breaker to HALF_OPEN.
// It resets the failure and success counts and adjusts the open timeout to the base timeout.
func (cb *BaseCircuitBreaker) transitionToHalfOpen() {
	cb.state = HalfOpen
	cb.failureCount = 0
	cb.successCount = 0
	cb.openTimeout = cb.baseTimeout // Reset to base timeout
	logger.Infof("Circuit breaker transitioned to HALF_OPEN for endpoint %s", cb.endpoint)
}

// transitionToClosed changes the state of the circuit breaker to CLOSED.
// It resets the failure and success counts and adjusts the open timeout to the base timeout.
func (cb *BaseCircuitBreaker) transitionToClosed() {
	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0
	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0
	cb.openTimeout = cb.baseTimeout // Reset to base timeout
	logger.Infof("Circuit breaker transitioned to CLOSED for endpoint %s", cb.endpoint)
}

// recordFailure increments the failure count and transitions to OPEN if the failure
// threshold is reached. It also saves the current state.
func (cb *BaseCircuitBreaker) recordFailure() {
	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.transitionToOpen()
	} else {
		cb.saveState()
	}
}

// transitionToOpen changes the state of the circuit breaker to OPEN.
// It sets the open timeout with exponential backoff based on the failure count,
// capping it at the maximum timeout value. It also updates the last failure time.
func (cb *BaseCircuitBreaker) transitionToOpen() {
	cb.state = Open
	cb.lastFailureTime = time.Now()

	// Exponential backoff: Increase openTimeout based on the failure count
	cb.openTimeout = cb.baseTimeout * time.Duration(1<<cb.failureCount) // 1 << n is equivalent to 2^n

	if cb.openTimeout > cb.maxTimeout {
		cb.openTimeout = cb.maxTimeout
	}

	logger.Errorf("Circuit breaker transitioned to OPEN for endpoint %s", cb.endpoint)
	cb.saveState()
}
