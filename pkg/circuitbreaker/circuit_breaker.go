package circuitbreaker

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/phil-inc/pcommon/pkg/redis"
)

var redisClient *redis.RedisClient
var ctx = context.Background() //TODO

func SetupRedis(url string) {
	redisClient := new(redis.RedisClient)
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
	mu                       sync.Mutex
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
	cb.loadState()

	return cb
}

func (cb *CircuitBreaker) loadState() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := redisClient.HGetAll(ctx, cb.url).Result()
	if err == nil && len(state) > 0 {
		cb.state = State(state["state"])
		cb.failureCount, _ = strconv.Atoi(state["failureCount"])
		cb.successCount, _ = strconv.Atoi(state["successCount"])
		cb.lastFailureTime, _ = time.Parse(time.RFC3339, state["lastFailureTime"])
	}
}

func (cb *CircuitBreaker) saveState() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := map[string]interface{}{
		"state":           cb.state,
		"failureCount":    cb.failureCount,
		"successCount":    cb.successCount,
		"lastFailureTime": cb.lastFailureTime.Format(time.RFC3339),
	}

	redisClient.HMSet(ctx, cb.url, state)
}

func (cb *CircuitBreaker) Call(f func() ([]byte, error)) ([]byte, error) {
	cb.mu.Lock()

	switch cb.state {
	case Open:
		if time.Since(cb.lastFailureTime) > cb.openTimeout {
			cb.state = HalfOpen
		} else {
			cb.mu.Unlock()
			return nil, fmt.Errorf("circuit breaker is open")
		}
	case HalfOpen:
		if cb.successCount >= cb.halfOpenSuccessThreshold {
			cb.state = Closed
			cb.failureCount = 0
			cb.successCount = 0
		}
	}

	// Release lock before function call
	cb.mu.Unlock()

	resp, err := f()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = Open
			cb.lastFailureTime = time.Now()
		}
		cb.saveState() // Save state after failure
		return nil, err
	}

	if cb.state == HalfOpen {
		cb.successCount++
	}

	cb.saveState() // Save state after success

	return resp, nil
}
