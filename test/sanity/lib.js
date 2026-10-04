// Shared helpers for the post-deploy sanity suite. The suite talks to a
// deployed environment through its public HTTP API only.
import http from 'k6/http';
import { check } from 'k6';

// Defaults to the staging deployment; override with --env BASE_URL=...
export const BASE_URL = (__ENV.BASE_URL || 'https://simple-order-service.kubeletto.app').replace(/\/+$/, '');

// batchRequest builds a request descriptor for http.batch, so several calls
// can be fired in parallel (used by the concurrency probe).
export function batchRequest(method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return {
    method,
    url: `${BASE_URL}${path}`,
    body: body === undefined ? null : JSON.stringify(body),
    params: { headers },
  };
}

// request performs one JSON call, attaching the bearer token when given.
export function request(method, path, body, token) {
  const spec = batchRequest(method, path, body, token);
  return http.request(spec.method, spec.url, spec.body, spec.params);
}

// expectStatus asserts the documented HTTP status for an endpoint.
export function expectStatus(res, status, name) {
  return check(res, { [`${name}: ${status}`]: (r) => r.status === status });
}

// expectJson asserts a value inside the response body (GJSON path).
export function expectJson(res, name, path, want) {
  return check(res, { [name]: (r) => r.json(path) === want });
}

// expectRequestID asserts the response carries the tracing header.
export function expectRequestID(res, name) {
  return check(res, {
    [name]: (r) => Object.keys(r.headers).some((key) => key.toLowerCase() === 'x-request-id'),
  });
}

// uniqueSuffix keeps runs independent: each execution registers fresh users
// and creates fresh products (the API has no delete endpoints).
export function uniqueSuffix() {
  return `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}

// registerUser registers a fresh user and returns its credentials and token.
export function registerUser() {
  const email = `sanity+${uniqueSuffix()}@example.com`;
  const password = 'sanity-pass-123';
  const res = request('POST', '/v1/auth/register', { name: 'Sanity Check', email, password });
  expectStatus(res, 201, 'setup: POST /v1/auth/register');
  const token = res.json('data.token');
  check(res, { 'setup: register returns a token': () => Boolean(token) });
  return { email, password, token };
}

// createProduct creates a fresh product and returns its response body.
export function createProduct(token, overrides = {}) {
  const body = {
    sku: `SANITY-${uniqueSuffix()}`,
    name: 'Sanity Product',
    description: 'created by the post-deploy sanity suite',
    price: 15000,
    stock: 5,
    ...overrides,
  };
  const res = request('POST', '/v1/products', body, token);
  expectStatus(res, 201, 'setup: POST /v1/products');
  return res.json('data');
}
