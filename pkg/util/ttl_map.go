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

func (tm *TTLMap) AcquireLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	//Check if the lock exists and hasn't expired
	item, ok := tm.data[key]
	if ok {
		//  0 mean the key never expires or time is before expiration time, so cannot acquire lock
		if item.Expiration.IsZero() || time.Now().Before(item.Expiration) {
			return false, nil
		}
	}

	tm.data[key] = element{
		Value:      true,
		Expiration: time.Now().Add(expiration),
	}

	return true, nil
}

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
