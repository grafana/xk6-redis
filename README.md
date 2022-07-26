# xk6-redis

This is a Redis client library for [k6](https://github.com/grafana/k6),
implemented as an extension using the [xk6](https://github.com/grafana/xk6) system.

| :exclamation: This is a proof of concept, isn't supported by the k6 team, and may break in the future. USE AT YOUR OWN RISK! |
| ---------------------------------------------------------------------------------------------------------------------------- |

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
import { check } from 'k6'
import http from 'k6/http'
import redis from 'k6/x/redis'
import exec from 'k6/execution'

export const options = {
    scenarios: {
        redisPerformance: {
            executor: 'shared-iterations',
            vus: 10,
            iterations: 200,
            exec: 'measureRedisPerformance',
        },
        usingRedisData: {
            executor: 'shared-iterations',
            vus: 10,
            iterations: 200,
            exec: 'measureUsingRedisData',
        },
    },
}

// Instantiate a new redis client
const redisClient = new redis.Client({
    addrs: new Array('localhost:6379'),
    password: 'foobar',
})

// Prepare an array of crocodile ids for later use
// in the context of the measureUsingRedisData function.
const crocodileIDs = new Array(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

export function measureRedisPerformance() {
    // VUs are executed in a parallel fashion,
    // thus, to ensure that parallel VUs are not
    // modifying the same key at the same time,
    // we use keys indexed by the VU id.
    const key = `foo-${exec.vu.idInTest}`

    redisClient
        .set(`foo-${exec.vu.idInTest}`, 1)
        .then(() => redisClient.get(`foo-${exec.vu.idInTest}`))
        .then((value) => redisClient.incrBy(`foo-${exec.vu.idInTest}`, value))
        .then((_) => redisClient.del(`foo-${exec.vu.idInTest}`))
        .then((_) => redisClient.exists(`foo-${exec.vu.idInTest}`))
        .then((exists) => {
            if (exists !== 0) {
                throw new Error('foo should have been deleted')
            }
        })
}

export function setup() {
    redisClient.sadd('crocodile_ids', ...crocodileIDs)
}

export function measureUsingRedisData() {
    // Pick a random crocodile id from the dedicated redis set,
    // we have filled in setup().
    redisClient
        .srandmember('crocodile_ids')
        .then((randomID) => {
            const url = `https://test-api.k6.io/public/crocodiles/${randomID}`
            const res = http.get(url)

            check(res, {
                'status is 200': (r) => r.status === 200,
                'content-type is application/json': (r) =>
                    r.headers['content-type'] === 'application/json',
            })

            return url
        })
        .then((url) => redisClient.hincrby('k6_crocodile_fetched', url, 1))
}

export function teardown() {
    redisClient.del('crocodile_ids')
}

export function handleSummary(data) {
    redisClient
        .hgetall('k6_crocodile_fetched')
        .then((fetched) => Object.assign(data, { k6_crocodile_fetched: fetched }))
        .then((data) => redisClient.set(`k6_report_${Date.now()}`, JSON.stringify(data)))
        .then(() => redisClient.del('k6_crocodile_fetched'))
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

| Redis Command | Module function signature                                             | Description                                                                                                                                                                                                           | Returns                                                                                                                                                                                                                                     |
| :------------ | :-------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **SET**       | `set(key: string, value: any, expiration: number) => Promise<string>` | Set `key` to hold `value`, with a time to live equal to `expiration` (expressed in seconds). If `key` already holds a value, it is overwritten.                                                                       | On **success**, the promise **resolves** with `"OK"`. If the provided `value` is not of a supported type, the promise is **rejected** with an error.                                                                                        |
| **GET**       | `get(key: string) => Promise<string>`                                 | Get the value of `key`.                                                                                                                                                                                               | On **success**, the promise **resolves** with the value of `key`. If `key` does not exist, the promise is **rejected** with an error.                                                                                                       |
| **GETSET**    | `getSet(key: string, value: any) => Promise<string>`                  | Atomically sets `key` to `value` and returns the old value stored at `key`. If `key` exists but does not hold a string value, or the provided `value` is not a supported type, the promise is rejected with an error. | On **success**, the promise **resolves** with the old value stored at `key`. If `key` does not exist, the promise is rejected with an error.                                                                                                |
| **DEL**       | `del(keys: string[]) => Promise<number>`                              | Removes the specified keys. A key is ignored if it does not exist. Returns the number of keys that were removed.                                                                                                      | On **success**, the promise **resolves** with the number of keys that were removed.                                                                                                                                                         |
| **GETDEL**    | `getDel(key: string) => Promise<string>`                              | Get the value of `key` and delete the key. This functionality is similar to `get`, except for the fact that it also deletes the key on success.                                                                       | On **success**, the promise **resolves** with the value of `key`. If the key does not exist, the promise is **rejected** with an error.                                                                                                     |
| **EXISTS**    | `exists(keys: string[]) => Promise<number>`                           | Returns the number of `key` arguments that exist. Note that if the same existing key is mentioned in the argument multiple times, it will be counted multiple times.                                                  | On **success**, the promise **resolves** with the number of keys that exist from those specified as arguments.                                                                                                                              |
| **INCR**      | `incr(key: string) => Promise<number>`                                | Increments the number stored at `key` by one. If the key does not exist, it is set to zero before performing the operation.  Returns the value of key after the increment.                                            | **success**, the promise **resolves** with the value of `key` after the increment. If the key contains a value of the wrong type, or contains a string that cannot be represented as an integer, the promise is **rejected** with an error. |
| **INCRBY**    | `incrby(key: string, increment: number) => Promise<number>`           | Increments the number stored at `key` by `increment`. If the key does not exist, it is set to zero before performing the operation.                                                                                   | On **success**, the promise **resolves** with . If the key contains a value of the wrong type, or contains a string that cannot be represented as an integer, the promise is **rejected** with an error.                                    |
| **DECR**      | `decr(key: string) => Promise<number>`                                | Decrements the number stored at `key` by one. If the key does not exist, it is set to zero before performing the operation                                                                                            | On **success**, the promise **resolves** with . If the key contains a value of the wrong type, or contains a string that cannot be represented as an integer, the promise is **rejected** with an error.                                    |
| **DECRBY**    | `decrby(key: string, decrement: number) => Promise<number>`           | Decrements the number stored at `key` by `decrement`. If the key does not exist, it is set to zero before performing the operation.                                                                                   | On **success**, the promise **resolves** with . If the key contains a value of the wrong type, or contains a string that cannot be represented as an integer, the promise is **rejected** with an error.                                    |
| **RANDOMKEY** | `randomKey() => string`                                               | Returns a random key.                                                                                                                                                                                                 | On **success**, the promise **resolves** with the random key.  If the database is empty, the promise is **rejected** with an error.                                                                                                         |
| **MGET**      | `mget(keys: string[]) => Promise<any[]>)`                             | Returns the values of all specified keys. For every key that does not hold a string value, or does not exist, the value `null` will be returned.                                                                      | On **success**, the promise **resolves** with the list of values at the specified keys.                                                                                                                                                     |
| **EXPIRE**    | `expire(key: string, seconds: number) => Promise<boolean>`            | Sets a timeout on key, after which the key will automatically be deleted. Note that calling Expire with a non-positive timeout will result in the key being deleted rather than expired.                              | On **success**, the promise **resolves** with `true` if the timeout was set, and `false` if the timeout wasn't set.                                                                                                                         |
| **TTL**       | `ttl(key: string) => Promise<number>`                                 | Returns the remaining time to live of a key that has a timeout.                                                                                                                                                       | On **success**, the promise **resolves** with the TTL value in seconds.                                                                                                                                                                     |
| **PERSIST**   | `persist(key: string) => Promise<boolean>`                            | Removes the existing timeout on key.                                                                                                                                                                                  | On **success**, the promise **resolves** with `true` if the timeout was removed, `false` otherwise.                                                                                                                                         |

### List field operations

| Redis Command | Module function signature                                               | Description                                                                                                                                                                                                                                                                                        | Returns                                                                                                                                                                    |
| :------------ | :---------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **LPUSH**     | `lpsuh(key: string, values: any[]) => Promise<number>`                  | Inserts all the specified values at the head of the list stored at `key`. If `key` does not exist, it is created as empty list before performing the push operations. When `key` holds a value that is not a list, and error is returned.                                                          | On **success**, the promise **resolves** with the lenght of the list after the push operations.                                                                            |
| **RPUSH**     | `rpush(key: string, values: any[]) => Promise<number>`                  | Inserts all the specified values at the tail of the list stored at `key`. If `key` does not exist, it is created as empty list before performing the push operations.                                                                                                                              | On **success**, the promise **resolves** with the length of the list after the push operation.                                                                             |
| **LPOP**      | `lpop(key: string) => Promise<string>`                                  | Removes and returns the first element of the list stored at `key`.                                                                                                                                                                                                                                 | On **success**, the promise **resolves** with the value of the first element. If the list does not exist, the promise is **rejected** with an error.                       |
| **RPOP**      | `rpop(key: string) => Promise<string>`                                  | Removes and returns the last element of the list stored at `key`.                                                                                                                                                                                                                                  | On **success**, the promise **resolves** with the value of the last element. If the list does not exist, the promise is **rejected** with an error.                        |
| **LRANGE**    | `lrange(key: string, start: number, stop: number) => Promise<string[]>` | Returns the specified elements of the list stored at `key`. The offsets start and stop are zero-based indexes. These offsets can be negative numbers, where they indicate offsets starting at the end of the list.                                                                                 | On **success**, the promise **resolves** with the list of elements in the specified range.                                                                                 |
| **LINDEX**    | `lindex(key: string, start: number, stop: number) => Promise<string>`   | Returns the specified element of the list stored at `key`. The index is zero-based. Negative indices can be used to designate elements starting at the tail of the list.                                                                                                                           | On **success**, the promise **resolves** with the requested element. If the list does not exist, or the index is out of bounds, the promise is **rejected** with an error. |
| **LSET**      | `lset(key: string, index: number, element: string)`                     | Sets the list element at `index` to `element`.                                                                                                                                                                                                                                                     | On **success**, the promise **resolves** with `"OK"`. If the list does not exist, or the index is out of bounds, the promise is **rejected** with an error.                |
| **LREM**      | `lrem(key: string, count: number, value: string) => Promise<number>`    | Removes the first `count` occurrences of `value` from the list stored at `key`. If `count` is positive, elements are removed from the beginning of the list. If `count` is negative, elements are removed from the end of the list. If `count` is zero, all elements matching `value` are removed. | On **success**, the promise **resolves** with the number of removed elements. If the list does not exist, the promise is **rejected** with an error.                       |
| **LLEN**      | `llen(key: string) => Promise<number>`                                  | Returns the length of the list stored at `key`. If `key` does not exist, it is interpreted as an empty list and 0 is returned.                                                                                                                                                                     | On **success**, the promise **resolves** with the length of the list at `key`. If the list does not exist, the promise is **rejected** with an error.                      |
 
### Hash field operations

| Redis Command | Module function signature                                                   | Description                                                                                                                                                                                                                                                           | Returns                                                                                                                                                                                       |
| :------------ | :-------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **HSET**      | `hset(key: string, field: string, value: string) => Promise<number>`        | Sets the specified field in the hash stored at `key` to `value`. If the `key` does not exist, a new key holding a hash is created. If `field` already exists in the hash, it is overwritten.                                                                          | On **success**, the promise **resolves** with the number of fields that were added. If the hash does not exist, the promise is **rejected** with an error.                                    |
| **HSETNX**    | `hsetnx(key: string, field: string, value: string) => Promise<boolean>`     | Sets the specified field in the hash stored at `key` to `value`, only if `field` does not yet exist. If `key` does not exist, a new key holding a hash is created. If `field` already exists, this operation has no effect.                                           | On **success**, the promise **resolves** with `1` if `field` is a new field in the hash and value was set, and with `0` if `field` already exists in the hash and no operation was performed. |
| **HGET**      | `hget(key: string, field: string) => Promise<string>`                       | Returns the value associated with `field` in the hash stored at `key`.                                                                                                                                                                                                | On **success**, the promise **resolves** with the value associated with `field`. If the hash does not exist, the promise is **rejected** with an error.                                       |
| **HDEL**      | `hdel(key: string, fields: string[]) => Promise<number>`                    | Deletes the specified fields from the hash stored at `key`. The number of fields that were removed from the hash is returned on resolution (non including non existing fields).                                                                                       | On **success**, the promise **resolves** with the number of fields that were removed from the hash, not including specified, but non existing, fields.                                        |
| **HGETALL**   | `hgetall(key: string) => Promise<[key: string]string>`                      | Returns all fields and values of the hash stored at `key`.                                                                                                                                                                                                            | On **success**, the promise **resolves** with the list of fields and their values stored in the hash.                                                                                         |
| **HKEYS**     | `hkeys(key: string) => Promise<string[]>`                                   | Returns all fields of the hash stored at `key`.                                                                                                                                                                                                                       | On **success**, the promise **resolves** with the list of fields in the hash. If the hash does not exist, the promise is **rejected** with an error.                                          |
| **HVALS**     | `hvals(key: string) => Promise<string[]>`                                   | Returns all values of the hash stored at `key`.                                                                                                                                                                                                                       | On **success**, the promise **resolves** with the list of values in the hash. If the hash does not exist, the promise is **rejected** with an error.                                          |
| **HLEN**      | `hlen(key: string) => Promise<number>`                                      | Returns the number of fields in the hash stored at `key`.                                                                                                                                                                                                             | On **success**, the promise **resolves** with the number of fields in the hash. If the hash does not exist, the promise is **rejected** with an error.                                        |
| **HINCRBY**   | `hincrby(key: string, field: string, increment: number) => Promise<number>` | Increments the integer value of `field` in the hash stored at `key` by `increment`. If `key` does not exist, a new key holding a hash is created. If `field` does not exist the value is set to 0 before the operation is set to 0 before the operation is performed. | On **success**, the promise **resolves** with the value at `field` after the increment operation.                                                                                             |

### Set field operations

| Redis Command   | Module function signature                                 | Description                                                                                                                                                                                                   | Returns                                                                                                                                             |
| :-------------- | :-------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :-------------------------------------------------------------------------------------------------------------------------------------------------- |
| **SADD**        | `sadd(key: string, members: any[]) => Promise<number>`    | Adds the specified members to the set stored at `key`. Specified members that are already a member of this set are ignored. If key does not exist, a new set is created before adding the specified members.  | On **success**, the promise **resolves** with the number of elements that were added to the set, not including elements already present in the set. |
| **SREM**        | `srem(key: string, members: any[]) => Promise<number>`    | Removes the specified members from the set stored at `key`. Specified members that are not a member of this set are ignored. If key does not exist, it is treated as an empty set and this command returns 0. | On **success**, the promise **resolves** with the number of members that were remove from the set, not including non existing members.              |
| **SISMEMBER**   | `sismember(key: string, member: any) => Promise<boolean>` | Returns if member is a member of the set stored at `key`.                                                                                                                                                     | On **success**, the promise **resolves** with `true` if the element is a member of the set, `false` otherwise.                                      |
| **SMEMBERS**    | `smembers(key: string) => Promise<string[]>`              | Returns all the members of the set values stored at `keys`.                                                                                                                                                   | On **success**, the promise **resolves** with an array containing the values present in the set.                                                    |
| **SRANDMEMBER** | `srandmember(key: string) => Promise<string>`             | Returns a random element from the set value stored at `key`.                                                                                                                                                  | On **success**, the promise **resolves** with the selected random member. If the set does not exist, the promise is **rejected** with an error.     |
| **SPOP**        | `spop(key: string) => Promise<string>`                    | Removes and returns a random element from the set value stored at `key`.                                                                                                                                      | On **success**, the promise **resolves** to the returned set member. If the set does not exist, the promise is **rejected** with an error.          |

### Custom operations

In the event a redis command you wish to use is not implemented yet, the `sendCommand(command: string, args: any[]) => Promise<any>` method can be used to send a custom commands to the server.
## See also

- [Benchmarking Redis with k6](https://k6.io/blog/benchmarking-redis-with-k6/)
