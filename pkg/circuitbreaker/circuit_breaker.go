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

type CircuitBreaker struct {
	state                    State
	failureThreshold         int
	failureCount             int
	successCount             int
	mu                       sync.RWMutex
	halfOpenSuccessThreshold int
	openTimeout              time.Duration
	lastFailureTime          time.Time
	url                      string
}

func NewCircuitBreaker(url string, failureThreshold int, halfOpenSuccessThreshold int, openTimeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		url:                      url,
		failureThreshold:         failureThreshold,
		halfOpenSuccessThreshold: halfOpenSuccessThreshold,
		openTimeout:              openTimeout,
	}

	// Load initial state from Redis
	if err := cb.loadState(); err != nil {
		logger.Errorf("Error loading initial state: %v", err)
	}

	return cb
}

func (cb *CircuitBreaker) loadState() error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, err := redisClient.HGetAll(ctx, cb.url).Result()
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

func (cb *CircuitBreaker) saveState() error {

	state := map[string]interface{}{
		"state":           string(cb.state),
		"failureCount":    cb.failureCount,
		"successCount":    cb.successCount,
		"lastFailureTime": cb.lastFailureTime.Format(time.RFC3339),
	}

	if err := redisClient.HSet(ctx, cb.url, state).Err(); err != nil {
		return err
	}

	return nil
}

func (cb *CircuitBreaker) Call(f func() ([]byte, error)) ([]byte, error) {
	cb.mu.Lock()

	switch cb.state {
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

func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.state = HalfOpen
	cb.failureCount = 0
	cb.successCount = 0
	logger.Infof("Circuit breaker transitioned to HALF_OPEN for url %s", cb.url)
}

func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0
	logger.Infof("Circuit breaker transitioned to CLOSED for url %s", cb.url)
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.transitionToOpen()
	} else {
		cb.saveState()
	}
}

func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = Open
	cb.lastFailureTime = time.Now()
	logger.Errorf("Circuit breaker transitioned to OPEN for url %s", cb.url)
	cb.saveState()
}