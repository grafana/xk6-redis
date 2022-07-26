# xk6-redis

This is a Redis client library for [k6](https://github.com/grafana/k6),
implemented as an extension using the [xk6](https://github.com/grafana/xk6) system.

| :exclamation: This is a proof of concept, isn't supported by the k6 team, and may break in the future. USE AT YOUR OWN RISK! |
|------|

Note that there is already a [k6 Redis extension](https://github.com/dgzlopes/xk6-redis)
that uses a different Go library and slightly different API. The extension in this
current repo served as an example for an [xk6 tutorial article](https://k6.io/blog/extending-k6-with-xk6),
but using one or the other is up to the user. :)

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

Then:

1. Install `xk6`:
  ```shell
  go install go.k6.io/xk6/cmd/xk6@latest
  ```

2. Build the binary:
  ```shell
  xk6 build --with github.com/grafana/xk6-redis
  ```

## Example test script

```javascript
import { check } from "k6";
import http from "k6/http";
import redis from "k6/x/redis";
import exec from "k6/execution";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  scenarios: {
    redisPerformance: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "measureRedisPerformance",
    },
    usingRedisData: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 200,
      exec: "measureUsingRedisData",
    },
  },
};

// Instantiate a new redis client
const redisClient = new redis.Client({
  addrs: __ENV.REDIS_ADDRS.split(",") || new Array("localhost:6379"), // in the form of "host:port", separated by commas
  password: __ENV.REDIS_PASSWORD || "",
});

// Prepare an array of crocodile ids for later use
// in the context of the measureUsingRedisData function.
const crocodileIDs = new Array(0, 1, 2, 3, 4, 5, 6, 7, 8, 9);

export function measureRedisPerformance() {
  // VUs are executed in a parallel fashion,
  // thus, to ensure that parallel VUs are not
  // modifying the same key at the same time,
  // we use keys indexed by the VU id.
  const key = `foo-${exec.vu.idInTest}`;

  redisClient
    .set(`foo-${exec.vu.idInTest}`, 1)
    .then(() => redisClient.get(`foo-${exec.vu.idInTest}`))
    .then((value) => redisClient.incrBy(`foo-${exec.vu.idInTest}`, value))
    .then((_) => redisClient.del(`foo-${exec.vu.idInTest}`))
    .then((_) => redisClient.exists(`foo-${exec.vu.idInTest}`))
    .then((exists) => {
      if (exists !== 0) {
        throw new Error("foo should have been deleted");
      }
    });
}

export function setup() {
  redisClient.sadd("crocodile_ids", ...crocodileIDs);
}

export function measureUsingRedisData() {
  // Pick a random crocodile id from the dedicated redis set,
  // we have filled in setup().
  redisClient
    .srandmember("crocodile_ids")
    .then((randomID) => {
      const url = `https://test-api.k6.io/public/crocodiles/${randomID}`;
      const res = http.get(url);

      check(res, {
        "status is 200": (r) => r.status === 200,
        "content-type is application/json": (r) =>
          r.headers["content-type"] === "application/json",
      });

      return url;
    })
    .then((url) => redisClient.hincrby("k6_crocodile_fetched", url, 1));
}

export function teardown() {
  redisClient.del("crocodile_ids");
}

export function handleSummary(data) {
  redisClient
    .hgetall("k6_crocodile_fetched")
    .then((fetched) => Object.assign(data, { k6_crocodile_fetched: fetched }))
    .then((data) =>
      redisClient.set(`k6_report_${Date.now()}`, JSON.stringify(data))
    )
    .then(() => redisClient.del("k6_crocodile_fetched"));

  return {
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}
```

Result output:

```shell
$ ./k6 run test.js

          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: test.js
     output: -

  scenarios: (100.00%) 1 scenario, 10 max VUs, 1m30s max duration (incl. graceful stop):
           * default: 10 looping VUs for 1m0s (gracefulStop: 30s)


running (1m00.1s), 00/10 VUs, 4954 complete and 0 interrupted iterations
default ✓ [======================================] 10 VUs  1m0s
```

## API

xk6-redis exposes a subset of Redis' [commands](https://redis.io/commands) the core team judged relevant in the context of k6 scripts.

### key/value operations
  - **`set(key: String, value: any) -> Promise<String>`**: set the given key with the given value.
  - **`get(key) -> Promise<String>`**: Get returns the value for a given key.
  - **`del(keys: String[]) -> Promise<Number>`**: removes the specified keys. A key is ignored if it does not exist. Returns the number of keys that were removed. 
  - **`exists(keys: String[]) -> Promise<Number>`**: returns the number of key arguments that exist. Note that if the same existing key is mentioned in the argument multiple times, it will be counted multiple times.
  - **`incr(key: String) -> Promise<Number>`**: increments the number stored at `key` by one. If the key does not exist, it is set to zero before performing the operation. An error is returned if the key contains a value of the wrong type, or contains a string that cannot be represented as an integer. Returns the value of key after the increment.
  - **`incrby(key: String, increment: Number) -> Promise<Number>`**: increments the number stored at `key` by `increment`. If the key does not exist, it is set to zero before performing the operation. An error is returned if the key contains a value of the wrong type, or contains a string that cannot be represented as an integer.
  - **`decr(key: String) -> Promise<Number>`**: decrements the number stored at `key` by one. If the key does not exist, it is set to zero before performing the operation. An error is returned if the key contains a value of the wrong type, or contains a string that cannot be represented as an integer.
  - **`decrby(key: String, decrement: Number) -> Promise<Number>`**: decrements the number stored at `key` by `decrement`. If the key does not exist, it is set to zero before performing the operation. An error is returned if the key contains a value of the wrong type, or contains a string that cannot be represented as an integer.
  - **`expire(key: String, seconds: Number) -> Promise<Boolean>`**: sets a timeout on key, after which the key will automatically be deleted. Note that calling Expire with a non-positive timeout will result in the key being deleted rather than expired.
  - **`ttl(key: String) -> Promise<Number>`**: returns the remaining time to live of a key that has a timeout.
  - **`persist(key: String) -> Promise<Boolean>`**: removes the existing timeout on key.
  - **`random() -> String`**: returns a random key.

### List field operations
- **`lpsuh(key: String, values: Array<any>) -> Promise<Number>`**: inserts all the specified values at the head of the list stored at `key`. If `key` does not exist, it is created as empty list before performing the push operations. When `key` holds a value that is not a list, and error is returned.
- **`rpush(key: String, values: Array<any>) -> Promise<Number>`**: inserts all the specified values at the tail of the list stored at `key`. If `key` does not exist, it is created as empty list before performing the push operations.
- **`lpop(key: String) -> Promise<String>`**: removes and returns the first element of the list stored at `key`.
- **`rpop(key: String) -> Promise<String>`**: removes and returns the last element of the list stored at `key`.
- **`lrange(key: String, start: Number, Stop: Number) -> Promise<String[]>`**: returns the specified elements of the list stored at `key`. The offsets start and stop are zero-based indexes. These offsets can be negative numbers, where they indicate offsets starting at the end of the list.
- **`lindex(key: String, start: Number, stop: Number) -> Promise<String>`**: returns the specified element of the list stored at `key`. The index is zero-based. Negative indices can be used to designate elements starting at the tail of the list.
- **`lset(key: String, index: Number, element: String)`**: sets the list element at `index` to `element`
- **`lrem(key: String, count: Number, value: String) -> Promise<Number>`**: removes the first `count` occurrences of `value` from the list stored at `key`. If `count` is positive, elements are removed from the beginning of the list. If `count` is negative, elements are removed from the end of the list. If `count` is zero, all elements matching `value` are removed.
- **`llen(key: String) -> Promise<Number>`**: returns the length of the list stored at `key`. If `key` does not exist, it is interpreted as an empty list and 0 is returned.
 
### Hash field operations
- **`hset(key: String, field: String, value: String) -> Promise<Number>`**:  sets the specified field in the hash stored at `key` to `value`. If the `key` does not exist, a new key holding a hash is created. If `field` already exists in the hash, it is overwritten.
- **`hsetnx(key: String, field: String, Value: string) -> Promise<Boolean>`**: sets the specified field in the hash stored at `key` to `value`, only if `field` does not yet exist. If `key` does not exist, a new key holding a hash is created. If `field` already exists, this operation has no effect.
- **`hget(key: String, field: String) -> Promise<String>`**: returns the value associated with `field` in the hash stored at `key`.
- **`hdel(key: String, fields: String[]) -> Promise<Number>`**: deletes the specified fields from the hash stored at `key`.
- **`hgetall(key: String) -> Promise<[key: string]string>`**: returns all fields and values of the hash stored at `key`.
- **`hkeys(key: String) -> Promise<String[]>`**: returns all fields of the hash stored at `key`.
- **`hvals(key: String) -> Promise<String[]>`**: returns all values of the hash stored at `key`.
- **`hlen(key: String) -> Promise<Number>`**: returns the number of fields in the hash stored at `key`.
- **`hincrby(key: String, field: String, increment: Number) -> Promise<Number>`**: increments the integer value of `field` in the hash stored at `key` by `increment`. If `key` does not exist, a new key holding a hash is created. If `field` does not exist the value is set to 0 before the operation is set to 0 before the operation is performed.

### Set field operations
- **`sadd(key: String, members: Any[]) -> Promise<Number>`**: adds the specified members to the set stored at key. Specified members that are already a member of this set are ignored. If key does not exist, a new set is created before adding the specified members.
- **`srem(key: String, members: Any[]) -> Promise<Number>`**: removes the specified members from the set stored at key. Specified members that are not a member of this set are ignored. If key does not exist, it is treated as an empty set and this command returns 0.
- **`sismember(key: String, member: Any) -> Promise<Boolean>`**: returns if member is a member of the set stored at key.
- **`srandmember(key: String) -> Promise<String>`**: returns a random element from the set value stored at key.
- **`spop(key: String) -> Promise<String>`**: removes and returns a random element from the set value stored at key.

### Custom operations

In the event a redis command you wish to use is not implemented yet, the `sendCommand(command: String, args: Array<any>) -> Promise<any>` method can be used to send a custom commands to the server.
## See also

- [Benchmarking Redis with k6](https://k6.io/blog/benchmarking-redis-with-k6/)
