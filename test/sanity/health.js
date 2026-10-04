// Sanity checks for the liveness and readiness probes.
import { expectJson, expectRequestID, expectStatus, request } from './lib.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

export default function health() {
  const healthz = request('GET', '/healthz');
  expectStatus(healthz, 200, 'GET /healthz');
  expectJson(healthz, 'healthz reports ok', 'data.status', 'ok');

  const readyz = request('GET', '/readyz');
  expectStatus(readyz, 200, 'GET /readyz');
  expectJson(readyz, 'readyz reports ready', 'data.status', 'ready');

  expectRequestID(healthz, 'responses carry X-Request-ID');
}
