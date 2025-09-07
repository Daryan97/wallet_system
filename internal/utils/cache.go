package utils

import (
	"context"       // Context for Redis operations
	"encoding/json" // JSON encoding/decoding
	"time"          // Time durations

	"github.com/redis/go-redis/v9" // Redis client
)

// GetCache retrieves a value from Redis and unmarshals it into dest
func GetCache(ctx context.Context, rdb *redis.Client, key string, dest any) (bool, error) {
	val, err := rdb.Get(ctx, key).Result() // Get value from Redis
	if err == redis.Nil {
		return false, nil // Key does not exist
	} else if err != nil {
		return false, err // Other Redis error
	}
	return true, json.Unmarshal([]byte(val), dest) // Unmarshal JSON into dest
}

// SetCache sets a value in Redis with a specified TTL
func SetCache(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value) // Marshal value to JSON
	if err != nil {
		return err // Return error if marshaling fails
	}
	return rdb.Set(ctx, key, b, ttl).Err() // Set value in Redis with TTL
}

// DeleteCache deletes a key from Redis
func DeleteCache(ctx context.Context, rdb *redis.Client, key string) error {
	return rdb.Del(ctx, key).Err() // Delete key from Redis
}
