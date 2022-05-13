import redis from "k6/x/redis";
import http from "k6/http";

export const options = {
  vus: 10,
  duration: "1m",
};

// Instantiate and configure a redis client in the init context
const r = new redis.Client({
  addrs: new Array("localhost:6379"),
  password: "foobar",
  dialTimeout: 100,
});

export function setup() {
  // Store crocodile ids in a set for later use
  for (let i = 0; i < 8; i++) {
    r.sadd("crocodile_ids", i);
  }
}

export default function () {
  r.srandmember("crocodile_ids")
    .then((id) => {
      // Get a random crocodile id and use it as the url to request
      const url = `https://test-api.k6.io/crocodile/${id}`;
      http.get(url);
      return url;
    })
    .then((url) => {
      // Store how many times we have tested this URL in a
      // redis hash.
      return r.hincrby("k6_crocodile_fetched", url);
    });
}

export function teardown() {
  // Let's clean after ourselves
  r.del("crocodile_ids");
}

export function handleSummary(data) {
  // Store the summary in Redis
  r.set(`k6_report_${Date.now()}`, JSON.stringify(data));
  r.del(`k6_crocodile_fetched`);
}
