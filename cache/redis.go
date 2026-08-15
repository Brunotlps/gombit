package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis stores cache values in Redis.
type Redis struct {
	client *redis.Client
}

// NewRedis wraps a go-redis client with Gombit's cache contract.
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
}

// Client returns the underlying go-redis client.
func (r *Redis) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Get implements Cache.
func (r *Redis) Get(ctx context.Context, key string, dst any) (bool, error) {
	payload, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: redis get %q: %w", key, err)
	}
	return true, decode(payload, dst)
}

// Set implements Cache.
func (r *Redis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}
	payload, err := encode(value)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache: redis set %q: %w", key, err)
	}
	return nil
}

// Delete implements Cache.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: redis delete: %w", err)
	}
	return nil
}

// Increment implements Cache.
func (r *Redis) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	value, err := r.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: redis increment %q: %w", key, err)
	}
	return value, nil
}
