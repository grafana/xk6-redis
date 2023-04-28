import redis from 'k6/x/redis';

const client = new redis.Client({
  addrs: ['localhost:32776', 'localhost:32777', 'localhost:32778'],
});

export default function() {
  const key = 'mykey';
  const value = 'myvalue';

  client.set(key, value);
  const result = client.get(key);

  console.log(result);
}