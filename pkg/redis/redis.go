package predis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	*redis.Client
}

func (rc *RedisClient) SetupRedis(addr string) {

	// Connect to redis.
	c := redis.NewClient(&redis.Options{
		Addr:     addr, // Redis server address
		Password: "",   // Password
		DB:       0,    //default DB
	})

	rc.Client = c
}

func (rc *RedisClient) AcquireLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	// Try to set the key with expiration (NX means set only if key does not exist)
	result, err := rc.SetNX(ctx, key, true, expiration).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

func (rc *RedisClient) ReleaseLock(ctx context.Context, key string) error {

	// Delete the key to release the lock
	_, err := rc.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	return nil
}
