// Package redis implements a redis client for k6.
package redis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/redis/go-redis/v9"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

type (
	// RootModule is the global module instance that will create Client
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		vu modules.VU

		*Client
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{vu: vu, Client: &Client{vu: vu}}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"Client": mi.NewClient,
	}}
}

// NewClient is the JS constructor for the redis Client.
//
// Under the hood, the redis.UniversalClient will be used. The universal client
// supports failover/sentinel, cluster and single-node modes. Depending on the options,
// the internal universal client instance will be one of those.
//
// The type of the underlying client depends on the following conditions:
// If the first argument is a string, it's parsed as a Redis URL, and a
// single-node Client is used.
// Otherwise, an object is expected, and depending on its properties:
// 1. If the MasterName option is specified, a sentinel-backed FailoverClient is used.
// 2. If the number of Addrs is two or more, a ClusterClient is used.
// 3. Otherwise, a single-node Client is used.
//
// To support being instantiated in the init context, while not
// producing any IO, as it is the convention in k6, the produced
// Client is initially configured, but in a disconnected state. In
// order to connect to the configured target instance(s), the `.Connect`
// should be called.
func (mi *ModuleInstance) NewClient(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	if len(call.Arguments) != 1 {
		common.Throw(rt, errors.New("must specify one argument"))
	}

	var (
		opts     *options
		parseErr error
	)
	switch val := call.Arguments[0].Export().(type) {
	case string:
		opts, parseErr = newOptionsFromString(val)
	case map[string]interface{}:
		opts, parseErr = newOptionsFromObject(val)
	default:
		common.Throw(rt, fmt.Errorf("unknown argument type: %T; expected string or object", val))
	}

	if parseErr != nil {
		common.Throw(rt, fmt.Errorf("invalid argument; reason: %w", parseErr))
	}

	redisOptions := &redis.UniversalOptions{
		Protocol:         2,
		Addrs:            opts.Addrs,
		DB:               opts.DB,
		Username:         opts.Username,
		Password:         opts.Password,
		SentinelUsername: opts.SentinelUsername,
		SentinelPassword: opts.SentinelPassword,
		MasterName:       opts.MasterName,
		MaxRetries:       opts.MaxRetries,
		MinRetryBackoff:  time.Duration(opts.MinRetryBackoff) * time.Millisecond,
		MaxRetryBackoff:  time.Duration(opts.MaxRetryBackoff) * time.Millisecond,
		DialTimeout:      time.Duration(opts.DialTimeout) * time.Millisecond,
		ReadTimeout:      time.Duration(opts.ReadTimeout) * time.Millisecond,
		WriteTimeout:     time.Duration(opts.WriteTimeout) * time.Millisecond,
		PoolSize:         opts.PoolSize,
		MinIdleConns:     opts.MinIdleConns,
		ConnMaxLifetime:  time.Duration(opts.MaxConnAge) * time.Millisecond,
		PoolTimeout:      time.Duration(opts.PoolTimeout) * time.Millisecond,
		ConnMaxIdleTime:  time.Duration(opts.IdleTimeout) * time.Millisecond,
		MaxRedirects:     opts.MaxRedirects,
		ReadOnly:         opts.ReadOnly,
		RouteByLatency:   opts.RouteByLatency,
		RouteRandomly:    opts.RouteRandomly,
	}

	client := &Client{
		vu:           mi.vu,
		redisOptions: redisOptions,
		redisClient:  nil,
	}

	return rt.ToValue(client).ToObject(rt)
}

type options struct {
	// Either a single address or a seed list of host:port addresses
	// of cluster/sentinel nodes.
	Addrs []string `json:"addrs,omitempty"`

	// Database to be selected after connecting to the server.
	// Only used in single-node and failover modes.
	DB int `json:"db,omitempty"`

	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string `json:"username,omitempty"`

	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string `json:"password,omitempty"`

	SentinelUsername string `json:"sentinelUsername,omitempty"`
	SentinelPassword string `json:"sentinelPassword,omitempty"`

	MasterName string `json:"masterName,omitempty"`

	MaxRetries      int   `json:"maxRetries,omitempty"`
	MinRetryBackoff int64 `json:"minRetryBackoff,omitempty"`
	MaxRetryBackoff int64 `json:"maxRetryBackoff,omitempty"`

	DialTimeout  int64 `json:"dialTimeout,omitempty"`
	ReadTimeout  int64 `json:"readTimeout,omitempty"`
	WriteTimeout int64 `json:"writeTimeout,omitempty"`

	PoolSize     int   `json:"poolSize,omitempty"`
	MinIdleConns int   `json:"minIdleConns,omitempty"`
	MaxConnAge   int64 `json:"maxConnAge,omitempty"`
	PoolTimeout  int64 `json:"poolTimeout,omitempty"`
	IdleTimeout  int64 `json:"idleTimeout,omitempty"`

	MaxRedirects   int  `json:"maxRedirects,omitempty"`
	ReadOnly       bool `json:"readOnly,omitempty"`
	RouteByLatency bool `json:"routeByLatency,omitempty"`
	RouteRandomly  bool `json:"routeRandomly,omitempty"`
}

// newOptionsFromObject validates and instantiates an options struct from its
// map representation as exported from goja.Runtime.
func newOptionsFromObject(argument map[string]interface{}) (*options, error) {
	jsonStr, err := json.Marshal(argument)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize options to JSON %w", err)
	}

	// Instantiate a JSON decoder which will error on unknown
	// fields. As a result, if the input map contains an unknown
	// option, this function will produce an error.
	decoder := json.NewDecoder(bytes.NewReader(jsonStr))
	decoder.DisallowUnknownFields()

	var opts options
	err = decoder.Decode(&opts)
	if err != nil {
		return nil, fmt.Errorf("unable to decode options %w", err)
	}

	return &opts, nil
}

// newOptionsFromString parses the expected URL into the internal options struct.
func newOptionsFromString(url string) (*options, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return &options{
		Addrs:              []string{opts.Addr},
		DB:                 opts.DB,
		Username:           opts.Username,
		Password:           opts.Password,
		MaxRetries:         opts.MaxRetries,
		MinRetryBackoff:    opts.MinRetryBackoff.Milliseconds(),
		MaxRetryBackoff:    opts.MaxRetryBackoff.Milliseconds(),
		DialTimeout:        opts.DialTimeout.Milliseconds(),
		ReadTimeout:        opts.ReadTimeout.Milliseconds(),
		WriteTimeout:       opts.WriteTimeout.Milliseconds(),
		PoolSize:           opts.PoolSize,
		MinIdleConns:       opts.MinIdleConns,
		MaxConnAge:         opts.MaxConnAge.Milliseconds(),
		PoolTimeout:        opts.PoolTimeout.Milliseconds(),
		IdleTimeout:        opts.IdleTimeout.Milliseconds(),
		IdleCheckFrequency: opts.IdleCheckFrequency.Milliseconds(),
	}, nil
}
