//docker-compose up -d
//Build k6: xk6 build v0.33.0 --with github.com/k6io/xk6-redis=$(pwd)
import redis from 'k6/x/redis';

const client = new redis.ClusterClient({
  addrs: ['localhost:6379', 'localhost:6380', 'localhost:6381', 'localhost:6382', 'localhost:6383', 'localhost:6384'],
  password: 'bitnami',
  db: 0,
});

export default function () {
  client.set('mykey_' + __VU + '_' + __ITER, 'myvalue_' + __VU + '_' + __ITER, 0);
  console.log(`mykey => ${client.get('mykey_'+ __VU + '_' + __ITER)}`);
}
