import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

// Custom metrics to count served vs throttled
const served = new Counter('requests_served');
const throttled = new Counter('requests_throttled');

export const options = {
  scenarios: {
    // Many virtual users hammering the API across multiple tenants
    load: {
      executor: 'constant-vus',
      vus: 50,            // 50 concurrent virtual users
      duration: '15s',
    },
  },
};

const TENANTS = ['tenantA', 'tenantB', 'tenantC', 'tenantD', 'tenantE'];

export default function () {
  // each VU randomly acts as one of several tenants
  const tenant = TENANTS[Math.floor(Math.random() * TENANTS.length)];
  const res = http.get('http://localhost:8080/api', {
    headers: { 'X-Tenant-ID': tenant },
  });

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });

  if (res.status === 200) served.add(1);
  if (res.status === 429) throttled.add(1);
}