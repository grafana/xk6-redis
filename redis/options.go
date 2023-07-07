package redis

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type singleNodeOptions struct {
	Socket          *socketOptions `json:"socket,omitempty"`
	Username        string         `json:"username,omitempty"`
	Password        string         `json:"password,omitempty"`
	Database        int            `json:"database,omitempty"`
	MaxRetries      int            `json:"maxRetries,omitempty"`
	MinRetryBackoff int64          `json:"minRetryBackoff,omitempty"`
	MaxRetryBackoff int64          `json:"maxRetryBackoff,omitempty"`
}

type socketOptions struct {
	Host               string      `json:"host,omitempty"`
	Port               int         `json:"port,omitempty"`
	TLS                *tlsOptions `json:"tls,omitempty"`
	DialTimeout        int64       `json:"dialTimeout,omitempty"`
	ReadTimeout        int64       `json:"readTimeout,omitempty"`
	WriteTimeout       int64       `json:"writeTimeout,omitempty"`
	PoolSize           int         `json:"poolSize,omitempty"`
	MinIdleConns       int         `json:"minIdleConns,omitempty"`
	MaxConnAge         int64       `json:"maxConnAge,omitempty"`
	PoolTimeout        int64       `json:"poolTimeout,omitempty"`
	IdleTimeout        int64       `json:"idleTimeout,omitempty"`
	IdleCheckFrequency int64       `json:"idleCheckFrequency,omitempty"`
}

type tlsOptions struct {
	// TODO: Handle binary data (ArrayBuffer) for all these as well.
	CA   []string `json:"ca,omitempty"`
	Cert string   `json:"cert,omitempty"`
	Key  string   `json:"key,omitempty"`
}

type commonClusterSentinelOptions struct {
	MaxRedirects   int  `json:"maxRedirects,omitempty"`
	ReadOnly       bool `json:"readOnly,omitempty"`
	RouteByLatency bool `json:"routeByLatency,omitempty"`
	RouteRandomly  bool `json:"routeRandomly,omitempty"`
}

type clusterNodesMapOptions struct {
	commonClusterSentinelOptions
	Nodes []*singleNodeOptions `json:"nodes,omitempty"`
}

type clusterNodesStringOptions struct {
	commonClusterSentinelOptions
	Nodes []string `json:"nodes,omitempty"`
}

type sentinelOptions struct {
	singleNodeOptions
	commonClusterSentinelOptions
	MasterName string `json:"masterName,omitempty"`
}

// newOptionsFromObject validates and instantiates an options struct from its
// map representation as exported from goja.Runtime.
func newOptionsFromObject(obj map[string]interface{}) (*redis.UniversalOptions, error) {
	var options interface{}
	if cluster, ok := obj["cluster"].(map[string]interface{}); ok {
		obj = cluster
		switch cluster["nodes"].(type) {
		case []interface{}:
			options = &clusterNodesMapOptions{}
		case []string:
			options = &clusterNodesStringOptions{}
		}
	} else if _, ok := obj["masterName"]; ok {
		options = &sentinelOptions{}
	} else {
		options = &singleNodeOptions{}
	}

	jsonStr, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize options to JSON %w", err)
	}

	// Instantiate a JSON decoder which will error on unknown
	// fields. As a result, if the input map contains an unknown
	// option, this function will produce an error.
	decoder := json.NewDecoder(bytes.NewReader(jsonStr))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&options)
	if err != nil {
		return nil, err
	}

	return toUniversalOptions(options)
}

// newOptionsFromString parses the expected URL into redis.UniversalOptions.
func newOptionsFromString(url string) (*redis.UniversalOptions, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return toUniversalOptions(opts)
}

func readOptions(options interface{}) (*redis.UniversalOptions, error) {
	var (
		opts *redis.UniversalOptions
		err  error
	)
	switch val := options.(type) {
	case string:
		opts, err = newOptionsFromString(val)
	case map[string]interface{}:
		opts, err = newOptionsFromObject(val)
	default:
		return nil, fmt.Errorf("invalid options type: %T; expected string or object", val)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid options; reason: %w", err)
	}

	return opts, nil
}

func toUniversalOptions(options interface{}) (*redis.UniversalOptions, error) {
	var universalOpts *redis.UniversalOptions
	switch o := options.(type) {
	case *clusterNodesMapOptions:
		universalOpts = &redis.UniversalOptions{
			Protocol:       2,
			MaxRedirects:   o.MaxRedirects,
			ReadOnly:       o.ReadOnly,
			RouteByLatency: o.RouteByLatency,
			RouteRandomly:  o.RouteRandomly,
		}
		for _, n := range o.Nodes {
			err := setConsistentOptions(universalOpts, n)
			if err != nil {
				return nil, err
			}

			if err := setSocketOptions(universalOpts, n.Socket); err != nil {
				return nil, err
			}
		}
	case *clusterNodesStringOptions:
		universalOpts = &redis.UniversalOptions{
			Protocol:       2,
			MaxRedirects:   o.MaxRedirects,
			ReadOnly:       o.ReadOnly,
			RouteByLatency: o.RouteByLatency,
			RouteRandomly:  o.RouteRandomly,
		}
		for _, n := range o.Nodes {
			opts, err := redis.ParseURL(n)
			if err != nil {
				return nil, err
			}
			err = setConsistentOptionsRedis(universalOpts, opts)
			if err != nil {
				return nil, err
			}
		}
	case *sentinelOptions:
	case *singleNodeOptions:
		universalOpts = &redis.UniversalOptions{
			Protocol:        2,
			DB:              o.Database,
			Username:        o.Username,
			Password:        o.Password,
			MaxRetries:      o.MaxRetries,
			MinRetryBackoff: time.Duration(o.MinRetryBackoff) * time.Millisecond,
			MaxRetryBackoff: time.Duration(o.MaxRetryBackoff) * time.Millisecond,
		}
		if err := setSocketOptions(universalOpts, o.Socket); err != nil {
			return nil, err
		}
	case *redis.Options:
		universalOpts = &redis.UniversalOptions{
			Protocol:        2,
			Addrs:           []string{o.Addr},
			DB:              o.DB,
			Username:        o.Username,
			Password:        o.Password,
			MaxRetries:      o.MaxRetries,
			MinRetryBackoff: o.MinRetryBackoff,
			MaxRetryBackoff: o.MaxRetryBackoff,
			DialTimeout:     o.DialTimeout,
			ReadTimeout:     o.ReadTimeout,
			WriteTimeout:    o.WriteTimeout,
			PoolSize:        o.PoolSize,
			MinIdleConns:    o.MinIdleConns,
			ConnMaxLifetime: o.ConnMaxLifetime,
			PoolTimeout:     o.PoolTimeout,
			ConnMaxIdleTime: o.ConnMaxIdleTime,
		}
	default:
		panic(fmt.Sprintf("unexpected options type %T", options))
	}

	return universalOpts, nil
}

func setSocketOptions(opts *redis.UniversalOptions, sopts *socketOptions) error {
	opts.Addrs = []string{fmt.Sprintf("%s:%d", sopts.Host, sopts.Port)}
	opts.DialTimeout = time.Duration(sopts.DialTimeout) * time.Millisecond
	opts.ReadTimeout = time.Duration(sopts.ReadTimeout) * time.Millisecond
	opts.WriteTimeout = time.Duration(sopts.WriteTimeout) * time.Millisecond
	opts.PoolSize = sopts.PoolSize
	opts.MinIdleConns = sopts.MinIdleConns
	opts.ConnMaxLifetime = time.Duration(sopts.MaxConnAge) * time.Millisecond
	opts.PoolTimeout = time.Duration(sopts.PoolTimeout) * time.Millisecond
	opts.ConnMaxIdleTime = time.Duration(sopts.IdleTimeout) * time.Millisecond

	if sopts.TLS != nil {
		tlsCfg := &tls.Config{}
		if len(sopts.TLS.CA) > 0 {
			caCertPool := x509.NewCertPool()
			for _, cert := range sopts.TLS.CA {
				caCertPool.AppendCertsFromPEM([]byte(cert))
			}
			tlsCfg.RootCAs = caCertPool
		}

		if sopts.TLS.Cert != "" && sopts.TLS.Key != "" {
			clientCertPair, err := tls.X509KeyPair([]byte(sopts.TLS.Cert), []byte(sopts.TLS.Key))
			if err != nil {
				return err
			}
			tlsCfg.Certificates = []tls.Certificate{clientCertPair}
		}

		opts.TLSConfig = tlsCfg
	}

	return nil
}

// Set UniversalOption values from single-node options, ensuring that any
// previously set values are consistent with the new values. This validates that
// multiple node options set when using cluster mode are consistent with each other.
// TODO: Use generics and/or reflection to DRY?
func setConsistentOptions(uopts *redis.UniversalOptions, opts *singleNodeOptions) error {
	uopts.Addrs = append(uopts.Addrs, fmt.Sprintf("%s:%d", opts.Socket.Host, opts.Socket.Port))

	if uopts.DB != 0 && opts.Database != 0 && uopts.DB != opts.Database {
		return fmt.Errorf("inconsistent db option: %d != %d", uopts.DB, opts.Database)
	}
	uopts.DB = opts.Database

	if uopts.Username != "" && opts.Username != "" && uopts.Username != opts.Username {
		return fmt.Errorf("inconsistent username option: %s != %s", uopts.Username, opts.Username)
	}
	uopts.Username = opts.Username

	if uopts.Password != "" && opts.Password != "" && uopts.Password != opts.Password {
		return fmt.Errorf("inconsistent password option")
	}
	uopts.Password = opts.Password

	if uopts.MaxRetries != 0 && opts.MaxRetries != 0 && uopts.MaxRetries != opts.MaxRetries {
		return fmt.Errorf("inconsistent maxRetries option: %d != %d", uopts.MaxRetries, opts.MaxRetries)
	}
	uopts.MaxRetries = opts.MaxRetries

	if uopts.MinRetryBackoff != 0 && opts.MinRetryBackoff != 0 && uopts.MinRetryBackoff.Milliseconds() != opts.MinRetryBackoff {
		return fmt.Errorf("inconsistent minRetryBackoff option: %d != %d", uopts.MinRetryBackoff.Milliseconds(), opts.MinRetryBackoff)
	}
	uopts.MinRetryBackoff = time.Duration(opts.MinRetryBackoff) * time.Millisecond

	if uopts.MaxRetryBackoff != 0 && opts.MaxRetryBackoff != 0 && uopts.MaxRetryBackoff.Milliseconds() != opts.MaxRetryBackoff {
		return fmt.Errorf("inconsistent maxRetryBackoff option: %d != %d", uopts.MaxRetryBackoff, opts.MaxRetryBackoff)
	}
	uopts.MaxRetryBackoff = time.Duration(opts.MaxRetryBackoff) * time.Millisecond

	// TODO: Validate opts.Socket ...

	return nil
}

// Set UniversalOption values from single-node options, ensuring that any
// previously set values are consistent with the new values. This validates that
// multiple node options set when using cluster mode are consistent with each other.
// TODO: Use generics and/or reflection to DRY?
func setConsistentOptionsRedis(uopts *redis.UniversalOptions, opts *redis.Options) error {
	uopts.Addrs = append(uopts.Addrs, opts.Addr)

	if uopts.DB != 0 && opts.DB != 0 && uopts.DB != opts.DB {
		return fmt.Errorf("inconsistent db option: %d != %d", uopts.DB, opts.DB)
	}
	uopts.DB = opts.DB

	if uopts.Username != "" && opts.Username != "" && uopts.Username != opts.Username {
		return fmt.Errorf("inconsistent username option: %s != %s", uopts.Username, opts.Username)
	}
	uopts.Username = opts.Username

	if uopts.Password != "" && opts.Password != "" && uopts.Password != opts.Password {
		return fmt.Errorf("inconsistent password option")
	}
	uopts.Password = opts.Password

	if uopts.MaxRetries != 0 && opts.MaxRetries != 0 && uopts.MaxRetries != opts.MaxRetries {
		return fmt.Errorf("inconsistent maxRetries option: %d != %d", uopts.MaxRetries, opts.MaxRetries)
	}
	uopts.MaxRetries = opts.MaxRetries

	if uopts.MinRetryBackoff != 0 && opts.MinRetryBackoff != 0 && uopts.MinRetryBackoff != opts.MinRetryBackoff {
		return fmt.Errorf("inconsistent minRetryBackoff option: %d != %d", uopts.MinRetryBackoff, opts.MinRetryBackoff)
	}
	uopts.MinRetryBackoff = opts.MinRetryBackoff

	if uopts.MaxRetryBackoff != 0 && opts.MaxRetryBackoff != 0 && uopts.MaxRetryBackoff != opts.MaxRetryBackoff {
		return fmt.Errorf("inconsistent maxRetryBackoff option: %d != %d", uopts.MaxRetryBackoff, opts.MaxRetryBackoff)
	}
	uopts.MaxRetryBackoff = opts.MaxRetryBackoff

	if uopts.DialTimeout != 0 && opts.DialTimeout != 0 && uopts.DialTimeout != opts.DialTimeout {
		return fmt.Errorf("inconsistent dialTimeout option: %d != %d", uopts.DialTimeout, opts.DialTimeout)
	}
	uopts.DialTimeout = opts.DialTimeout

	if uopts.ReadTimeout != 0 && opts.ReadTimeout != 0 && uopts.ReadTimeout != opts.ReadTimeout {
		return fmt.Errorf("inconsistent readTimeout option: %d != %d", uopts.ReadTimeout, opts.ReadTimeout)
	}
	uopts.ReadTimeout = opts.ReadTimeout

	if uopts.WriteTimeout != 0 && opts.WriteTimeout != 0 && uopts.WriteTimeout != opts.WriteTimeout {
		return fmt.Errorf("inconsistent writeTimeout option: %d != %d", uopts.WriteTimeout, opts.WriteTimeout)
	}
	uopts.WriteTimeout = opts.WriteTimeout

	if uopts.PoolSize != 0 && opts.PoolSize != 0 && uopts.PoolSize != opts.PoolSize {
		return fmt.Errorf("inconsistent poolSize option: %d != %d", uopts.PoolSize, opts.PoolSize)
	}
	uopts.PoolSize = opts.PoolSize

	if uopts.MinIdleConns != 0 && opts.MinIdleConns != 0 && uopts.MinIdleConns != opts.MinIdleConns {
		return fmt.Errorf("inconsistent minIdleConns option: %d != %d", uopts.MinIdleConns, opts.MinIdleConns)
	}
	uopts.MinIdleConns = opts.MinIdleConns

	if uopts.ConnMaxLifetime != 0 && opts.ConnMaxLifetime != 0 && uopts.ConnMaxLifetime != opts.ConnMaxLifetime {
		return fmt.Errorf("inconsistent maxConnAge option: %d != %d", uopts.ConnMaxLifetime, opts.ConnMaxLifetime)
	}
	uopts.ConnMaxLifetime = opts.ConnMaxLifetime

	if uopts.PoolTimeout != 0 && opts.PoolTimeout != 0 && uopts.PoolTimeout != opts.PoolTimeout {
		return fmt.Errorf("inconsistent poolTimeout option: %d != %d", uopts.PoolTimeout, opts.PoolTimeout)
	}
	uopts.PoolTimeout = opts.PoolTimeout

	if uopts.ConnMaxIdleTime != 0 && opts.ConnMaxIdleTime != 0 && uopts.ConnMaxIdleTime != opts.ConnMaxIdleTime {
		return fmt.Errorf("inconsistent idleTimeout option: %d != %d", uopts.ConnMaxIdleTime, opts.ConnMaxIdleTime)
	}
	uopts.ConnMaxIdleTime = opts.ConnMaxIdleTime

	return nil
}
