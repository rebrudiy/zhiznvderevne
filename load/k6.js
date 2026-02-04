import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<1000'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  http.get(`${BASE}/`);
  http.get(`${BASE}/api/posts`);
  sleep(1);
}
