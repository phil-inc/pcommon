package util

import (
	"context"
	"errors"
	"sync"
	"time"
)

type TTLMap struct {
	data map[string]element
	mu   sync.Mutex
}

type element struct {
	Value      interface{}
	Expiration time.Time
}

func (tm *TTLMap) Init() {
	tm.data = make(map[string]element, 0)
}

// AcquireLock acquires the lock if key does not exist or the key has expired
func (tm *TTLMap) AcquireLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	// Validate expiration. Must be positive.
	if expiration < 0 {
		return false, errors.New("invalid expiration")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	//Check if the lock exists and hasn't expired
	if !tm.IsExpired(key) {
		return false, errors.New("lock cannot be acquired")
	}

	tm.data[key] = element{
		Value:      true,
		Expiration: time.Now().Add(expiration),
	}

	return true, nil
}

// ReleaseLock relaeases the lock if key exists, otherwise throws an error
func (tm *TTLMap) ReleaseLock(ctx context.Context, key string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, ok := tm.data[key]
	if !ok {
		return errors.New("key does not exist")
	}

	// Delete the key
	delete(tm.data, key)

	return nil
}

func (tm *TTLMap) Put(key string, val interface{}, expiration time.Duration) error {
	// Validate expiration. Must be positive.
	if expiration < 0 {
		return errors.New("invalid expiration")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	exp := time.Time{}
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}
	tm.data[key] = element{Value: val, Expiration: exp}
	return nil
}

func (tm *TTLMap) Get(key string) (interface{}, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	e, ok := tm.data[key]
	if !ok {
		return nil, false
	}

	//Check if the lock exists and hasn't expired

	if tm.IsExpired(key) {
		delete(tm.data, key)
		return nil, false
	}
	return e.Value, true
}

func (tm *TTLMap) IsExpired(key string) bool {
	item, ok := tm.data[key]
	if !ok {
		return true
	}
	if !item.Expiration.IsZero() && time.Now().After(item.Expiration) {
		return true
	}
	return false
}
