import { check } from "k6";
// import http from "k6/http";
import redis from "k6/x/redis";
// import exec from "k6/execution";
// import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export const options = {
  insecureSkipTLSVerify: true,
  // scenarios: {
  //   redisPerformance: {
  //     executor: "shared-iterations",
  //     vus: 10,
  //     iterations: 200,
  //     exec: "measureRedisPerformance",
  //   },
  //   usingRedisData: {
  //     executor: "shared-iterations",
  //     vus: 10,
  //     iterations: 200,
  //     exec: "measureUsingRedisData",
  //   },
  // },
};

// Instantiate a new redis client with a URL
// const client = new redis.Client('redis://:tjkbZ8jrwz3pGiku@localhost:6379');

// A single-node client configured with an object
// const client = new redis.Client({
//   socket: {
//     host: 'localhost',
//     port: 6379,
//     // tls: {
//     //   ca: [open('ca.crt')],
//     // }
//   },
//   password: __ENV.REDIS_PASSWORD || "",
// });

// A single-node client configured with an object and mTLS
const client = new redis.Client({
  socket: {
    host: 'localhost',
    port: 6379,
    tls: {
      ca: [open('ca.crt')],
      cert: open('client.crt'),
      key: open('client.key'),
    }
  },
  password: __ENV.REDIS_PASSWORD,
});

// A cluster client configured with node URLs
// const client = new redis.Client({
//   cluster: {
//     nodes: ['redis://host1:6379', 'redis://host2:6379']
//   }
// });

// A cluster client configured with node objects
// const client = new redis.Client({
//   cluster: {
//     nodes: [
//       {
//         socket: {
//           host: 'host1',
//           port: 6379,
//         },
//         password: __ENV.REDIS_PASSWORD || "",
//       },
//       {
//         socket: {
//           host: 'host2',
//           port: 6379,
//         },
//         password: __ENV.REDIS_PASSWORD || "",
//       }
//     ]
//   }
// });

export default function () {
  client.set("a", 1).then((val) => {
    console.log(val);
  })
}

export function teardown() {
  client.get("a").then((val) => {
    console.log(val);
  })
}

// Prepare an array of crocodile ids for later use
// in the context of the measureUsingRedisData function.
const crocodileIDs = new Array(0, 1, 2, 3, 4, 5, 6, 7, 8, 9);

// export function measureRedisPerformance() {
//   // VUs are executed in a parallel fashion,
//   // thus, to ensure that parallel VUs are not
//   // modifying the same key at the same time,
//   // we use keys indexed by the VU id.
//   const key = `foo-${exec.vu.idInTest}`;

//   client
//     .set(`foo-${exec.vu.idInTest}`, 1)
//     .then(() => client.get(`foo-${exec.vu.idInTest}`))
//     .then((value) => client.incrBy(`foo-${exec.vu.idInTest}`, value))
//     .then((_) => client.del(`foo-${exec.vu.idInTest}`))
//     .then((_) => client.exists(`foo-${exec.vu.idInTest}`))
//     .then((exists) => {
//       if (exists !== 0) {
//         throw new Error("foo should have been deleted");
//       }
//     });
// }

// export function setup() {
//   client.sadd("crocodile_ids", ...crocodileIDs);
// }

// export function measureUsingRedisData() {
//   // Pick a random crocodile id from the dedicated redis set,
//   // we have filled in setup().
//   client
//     .srandmember("crocodile_ids")
//     .then((randomID) => {
//       const url = `https://test-api.k6.io/public/crocodiles/${randomID}`;
//       const res = http.get(url);

//       check(res, {
//         "status is 200": (r) => r.status === 200,
//         "content-type is application/json": (r) =>
//           r.headers["content-type"] === "application/json",
//       });

//       return url;
//     })
//     .then((url) => client.hincrby("k6_crocodile_fetched", url, 1));
// }

// export function teardown() {
//   client.del("crocodile_ids");
// }

// export function handleSummary(data) {
//   client
//     .hgetall("k6_crocodile_fetched")
//     .then((fetched) => Object.assign(data, { k6_crocodile_fetched: fetched }))
//     .then((data) =>
//       client.set(`k6_report_${Date.now()}`, JSON.stringify(data))
//     )
//     .then(() => client.del("k6_crocodile_fetched"));

//   return {
//     stdout: textSummary(data, { indent: "  ", enableColors: true }),
//   };
// }
