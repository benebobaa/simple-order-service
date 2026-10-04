// Shared helpers for the post-deploy sanity suite. The suite talks to a
// deployed environment through its public HTTP API only.
import http from 'k6/http';
import { check } from 'k6';

// Defaults to the staging deployment; override with --env BASE_URL=...
export const BASE_URL = (__ENV.BASE_URL || 'https://simple-order-service.kubeletto.app').replace(/\/+$/, '');

// request performs one JSON call, attaching the bearer token when given.
export function request(method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return http.request(method, `${BASE_URL}${path}`, body === undefined ? null : JSON.stringify(body), {
    headers,
  });
}

// expectStatus asserts the documented status code for an endpoint.
export function expectStatus(res, status, name) {
  return check(res, { [`${name}: ${status}`]: (r) => r.status === status });
}

// uniqueSuffix keeps runs independent: each execution registers a fresh user
// and creates a fresh product (the API has no delete endpoints).
export function uniqueSuffix() {
  return `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}
