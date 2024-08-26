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

func SetupRedis(url string) {
	redisClient = new(redis.RedisClient)
	redisClient.SetupRedis(url)
}

type State string

const (
	Closed   State = "CLOSED"
	Open     State = "OPEN"
	HalfOpen State = "HALF_OPEN"
)

type CircuitBreaker interface {
	HandleRequest(f func() ([]byte, error)) ([]byte, error)
	loadState() error
	saveState() error
	transitionToHalfOpen()
	transitionToClosed()
	recordFailure()
	transitionToOpen()
}

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
		isInbound:                isInbound,
	}

	// Load initial state from Redis
	if err := cb.loadState(); err != nil {
		logger.Errorf("Error loading initial state: %v", err)
	}

	return cb
}

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
		cb.failureCount, _ = strconv.Atoi(state["failureCount"])
		cb.successCount, _ = strconv.Atoi(state["successCount"])
		cb.lastFailureTime, _ = time.Parse(time.RFC3339, state["lastFailureTime"])
	}

	return nil
}

func (cb *BaseCircuitBreaker) saveState() error {

	keyPrefix := "outbound:" // Default prefix for outbound
	if cb.isInbound {
		keyPrefix = "inbound:"
	}

	state := map[string]interface{}{
		"state":           string(cb.state),
		"failureCount":    cb.failureCount,
		"successCount":    cb.successCount,
		"lastFailureTime": cb.lastFailureTime.Format(time.RFC3339),
	}

	if err := redisClient.HSet(ctx, keyPrefix+cb.endpoint, state).Err(); err != nil {
		return err
	}

	return nil
}

func (cb *BaseCircuitBreaker) transitionToHalfOpen() {
	cb.state = HalfOpen
	cb.failureCount = 0
	cb.successCount = 0
	cb.openTimeout = cb.baseTimeout // Reset to base timeout
	logger.Infof("Circuit breaker transitioned to HALF_OPEN for endpoint %s", cb.endpoint)
}

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

func (cb *BaseCircuitBreaker) recordFailure() {
	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.transitionToOpen()
	} else {
		cb.saveState()
	}
}

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
