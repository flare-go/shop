// Package driver

package driver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	maxRetries      = 3
	minRetryBackoff = 100 * time.Millisecond
	maxRetryBackoff = 300 * time.Millisecond
	dialTimeout     = 5 * time.Second
	readTimeout     = 3 * time.Second
	writeTimeout    = 3 * time.Second
)

// ConnectRedis connects to the Redis server and returns a *redis.Client and an errors
func ConnectRedis(addr string, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		MaxRetries:      maxRetries,
		MinRetryBackoff: minRetryBackoff,
		MaxRetryBackoff: maxRetryBackoff,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
	})

	// Test the connection
	if err := testRedis(client); err != nil {
		log.Println(fmt.Sprintf("Redis connection error: %s", err))
		return nil, err
	}

	return client, nil
}

// testRedis pings the Redis server to verify the connection
func testRedis(client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return client.Ping(ctx).Err()
}
