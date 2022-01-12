import redis from 'k6/x/redis';

const client = new redis.Client({
  addrs: ['localhost:6379'],
  password: '',
  db: 0,
});

export default function () {
  client.set('mykey', 'myvalue', 0);
  console.log(`mykey => ${client.get('mykey')}`);
}
