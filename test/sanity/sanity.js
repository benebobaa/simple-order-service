// Post-deploy sanity suite: one sequential pass over the public API of a
// deployed environment. Every check must pass (threshold below), so a broken
// release fails the deploy workflow and can be rolled back deliberately.
import { check } from 'k6';
import { expectStatus, request, uniqueSuffix } from './lib.js';

// One virtual user, one pass: a functional gate, not a load test.
export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ['rate==1'],
  },
};

const PASSWORD = 'sanity-pass-123';

export default function () {
  const suffix = uniqueSuffix();
  const email = `sanity+${suffix}@example.com`;
  const sku = `SANITY-${suffix}`;

  // Health probes.
  const health = request('GET', '/healthz');
  expectStatus(health, 200, 'GET /healthz');
  check(health, { 'healthz reports ok': (r) => r.json('data.status') === 'ok' });

  const ready = request('GET', '/readyz');
  expectStatus(ready, 200, 'GET /readyz');
  check(ready, { 'readyz reports ready': (r) => r.json('data.status') === 'ready' });

  // Auth: register, login, me.
  const register = request('POST', '/v1/auth/register', {
    name: 'Sanity Check',
    email,
    password: PASSWORD,
  });
  expectStatus(register, 201, 'POST /v1/auth/register');
  check(register, { 'register returns a token': (r) => Boolean(r.json('data.token')) });

  const login = request('POST', '/v1/auth/login', { email, password: PASSWORD });
  expectStatus(login, 200, 'POST /v1/auth/login');
  const token = login.json('data.token');
  check(login, { 'login returns a token': () => Boolean(token) });

  const me = request('GET', '/v1/auth/me', undefined, token);
  expectStatus(me, 200, 'GET /v1/auth/me');
  check(me, { 'me returns the registered email': (r) => r.json('data.email') === email });

  // Products: create, read, update, list.
  const createProduct = request(
    'POST',
    '/v1/products',
    {
      sku,
      name: 'Sanity Product',
      description: 'created by the post-deploy sanity suite',
      price: 15000,
      stock: 5,
    },
    token,
  );
  expectStatus(createProduct, 201, 'POST /v1/products');
  const productID = createProduct.json('data.id');
  check(createProduct, { 'product starts with the requested stock': (r) => r.json('data.stock') === 5 });

  const getProduct = request('GET', `/v1/products/${productID}`);
  expectStatus(getProduct, 200, 'GET /v1/products/:id');
  check(getProduct, { 'product SKU round-trips': (r) => r.json('data.sku') === sku });

  const updateProduct = request(
    'PUT',
    `/v1/products/${productID}`,
    { name: 'Sanity Product v2', price: 17000, stock: 5 },
    token,
  );
  expectStatus(updateProduct, 200, 'PUT /v1/products/:id');
  check(updateProduct, { 'product update is persisted': (r) => r.json('data.price') === 17000 });

  const listProducts = request('GET', '/v1/products?limit=1');
  expectStatus(listProducts, 200, 'GET /v1/products');
  check(listProducts, { 'product list is paginated': (r) => r.json('meta.total') > 0 });

  // Orders: create, read, list, cancel, cancel again.
  const createOrder = request('POST', '/v1/orders', { items: [{ sku, quantity: 2 }] }, token);
  expectStatus(createOrder, 201, 'POST /v1/orders');
  const orderID = createOrder.json('data.id');
  check(createOrder, {
    'order snapshots the unit price': (r) => r.json('data.items.0.unit_price') === 17000,
    'order total is computed': (r) => r.json('data.total_price') === 34000,
  });

  const getOrder = request('GET', `/v1/orders/${orderID}`, undefined, token);
  expectStatus(getOrder, 200, 'GET /v1/orders/:id');
  check(getOrder, { 'order is pending after creation': (r) => r.json('data.status') === 'pending' });

  const listOrders = request('GET', '/v1/orders', undefined, token);
  expectStatus(listOrders, 200, 'GET /v1/orders');

  const cancel = request('POST', `/v1/orders/${orderID}/cancel`, undefined, token);
  expectStatus(cancel, 200, 'POST /v1/orders/:id/cancel');
  check(cancel, { 'order is cancelled': (r) => r.json('data.status') === 'cancelled' });

  const cancelAgain = request('POST', `/v1/orders/${orderID}/cancel`, undefined, token);
  expectStatus(cancelAgain, 409, 'POST /v1/orders/:id/cancel (again)');
  check(cancelAgain, {
    'second cancel reports ORDER_ALREADY_CANCELLED': (r) => r.json('error.code') === 'ORDER_ALREADY_CANCELLED',
  });

  // Auth guard: protected endpoints reject anonymous callers.
  const anonymous = request('GET', '/v1/orders');
  expectStatus(anonymous, 401, 'GET /v1/orders without token');
}
