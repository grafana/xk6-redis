package redis

import (
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/loadimpact/k6/js/modules"
)

// Register the extension on module initialization, available to
// import from JS as "k6/x/redis".
func init() {
	modules.Register("k6/x/redis", new(Redis))
}

// Redis is the k6 extension for a Redis client.
type Redis struct{}

// Client is the Redis client wrapper.
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client with the provided options.
func (r *Redis) NewClient(opts *redis.Options) *Client {
	return &Client{client: redis.NewClient(opts)}
}

// Set the given key with the given value and expiration time.
func (c *Client) Set(key, value string, exp time.Duration) {
	c.client.Set(c.client.Context(), key, value, exp)
}

// Get returns the value for the given key.
func (c *Client) Get(key string) (string, error) {
	res, err := c.client.Get(c.client.Context(), key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}
