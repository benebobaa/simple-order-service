// Sanity checks for registration, login and the authenticated profile.
import { check } from 'k6';
import { expectJson, expectStatus, request, uniqueSuffix } from './lib.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

const PASSWORD = 'sanity-pass-123';

export default function auth() {
  const email = `sanity+${uniqueSuffix()}@example.com`;
  const credentials = { name: 'Sanity Check', email, password: PASSWORD };

  // Register returns the user and a usable bearer token.
  const register = request('POST', '/v1/auth/register', credentials);
  expectStatus(register, 201, 'POST /v1/auth/register');
  expectJson(register, 'register returns a bearer token', 'data.token_type', 'Bearer');
  check(register, {
    'register returns a token': (r) => Boolean(r.json('data.token')),
    'register returns the user': (r) => r.json('data.user.email') === email,
  });

  // Invalid payloads and duplicate emails are rejected.
  const invalid = request('POST', '/v1/auth/register', {
    name: 'Sanity Check',
    email: `sanity+${uniqueSuffix()}@example.com`,
    password: 'short',
  });
  expectStatus(invalid, 400, 'POST /v1/auth/register (short password)');
  expectJson(invalid, 'invalid payload reports VALIDATION_ERROR', 'error.code', 'VALIDATION_ERROR');

  const duplicate = request('POST', '/v1/auth/register', credentials);
  expectStatus(duplicate, 409, 'POST /v1/auth/register (duplicate email)');
  expectJson(duplicate, 'duplicate email reports EMAIL_ALREADY_EXISTS', 'error.code', 'EMAIL_ALREADY_EXISTS');

  // Login accepts the right password and rejects the wrong one.
  const login = request('POST', '/v1/auth/login', { email, password: PASSWORD });
  expectStatus(login, 200, 'POST /v1/auth/login');
  const token = login.json('data.token');
  check(login, { 'login returns a token': () => Boolean(token) });

  const wrongPassword = request('POST', '/v1/auth/login', { email, password: 'wrong-password' });
  expectStatus(wrongPassword, 401, 'POST /v1/auth/login (wrong password)');
  expectJson(wrongPassword, 'bad credentials report INVALID_CREDENTIALS', 'error.code', 'INVALID_CREDENTIALS');

  // The profile endpoint is owner-scoped.
  const me = request('GET', '/v1/auth/me', undefined, token);
  expectStatus(me, 200, 'GET /v1/auth/me');
  expectJson(me, 'me returns the registered email', 'data.email', email);

  const anonymous = request('GET', '/v1/auth/me');
  expectStatus(anonymous, 401, 'GET /v1/auth/me (no token)');
}
