// Package redis only exists to register the redis extension
package redis

import (
	"github.com/grafana/xk6-redis/redis"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/redis", new(redis.RootModule))
}
