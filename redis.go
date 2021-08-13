package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
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

// ClusterClient is the Redis cluster client wrapper
type ClusterClient struct {
	client *redis.ClusterClient
}

// XClient represents the Client constructor (i.e. `new redis.Client()`) and
// returns a new Redis client object.
func (r *Redis) XClient(ctxPtr *context.Context, opts *redis.Options) interface{} {
	rt := common.GetRuntime(*ctxPtr)
	return common.Bind(rt, &Client{client: redis.NewClient(opts)}, ctxPtr)
}

// XClusterClient represents the Client constructor (i.e. `new redis.Client()`) and
// returns a new Redis client object.
func (r *Redis) XClusterClient(ctxPtr *context.Context, opts *redis.ClusterOptions) interface{} {
	rt := common.GetRuntime(*ctxPtr)
	fmt.Printf("%v", opts.Addrs)
	return common.Bind(rt, &ClusterClient{client: redis.NewClusterClient(opts)}, ctxPtr)
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

// Set the given key with the given value and expiration time.
func (c *ClusterClient) Set(key, value string, exp time.Duration) {
	c.client.Set(c.client.Context(), key, value, exp)
}

// Get returns the value for the given key.
func (c *ClusterClient) Get(key string) (string, error) {
	res, err := c.client.Get(c.client.Context(), key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}
