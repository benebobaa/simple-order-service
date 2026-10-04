// Sanity checks for the order endpoints: create, read, list and cancel,
// including stock reservation and restoration.
import { check } from 'k6';
import { createProduct, expectJson, expectStatus, registerUser, request } from './lib.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

export default function orders() {
  const { token } = registerUser();
  const product = createProduct(token, { price: 15000, stock: 5 });

  // Create snapshots the price and reserves stock.
  const created = request('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 2 }] }, token);
  expectStatus(created, 201, 'POST /v1/orders');
  const orderID = created.json('data.id');
  check(created, {
    'order snapshots the unit price': (r) => r.json('data.items.0.unit_price') === 15000,
    'order total is computed': (r) => r.json('data.total_price') === 30000,
    'order starts pending': (r) => r.json('data.status') === 'pending',
  });

  const reserved = request('GET', `/v1/products/${product.id}`);
  expectJson(reserved, 'create reserves stock', 'data.stock', 3);

  // Invalid and impossible orders are rejected.
  const empty = request('POST', '/v1/orders', { items: [] }, token);
  expectStatus(empty, 400, 'POST /v1/orders (empty items)');

  const unknownSKU = request('POST', '/v1/orders', { items: [{ sku: 'SANITY-DOES-NOT-EXIST', quantity: 1 }] }, token);
  expectStatus(unknownSKU, 404, 'POST /v1/orders (unknown SKU)');
  expectJson(unknownSKU, 'unknown SKU reports PRODUCT_NOT_FOUND', 'error.code', 'PRODUCT_NOT_FOUND');

  const insufficient = request('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 99 }] }, token);
  expectStatus(insufficient, 409, 'POST /v1/orders (insufficient stock)');
  expectJson(insufficient, 'short stock reports INSUFFICIENT_STOCK', 'error.code', 'INSUFFICIENT_STOCK');

  const anonymous = request('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 1 }] });
  expectStatus(anonymous, 401, 'POST /v1/orders (no token)');

  // Reads are owner-scoped: other users get a 404, not a 403.
  const got = request('GET', `/v1/orders/${orderID}`, undefined, token);
  expectStatus(got, 200, 'GET /v1/orders/:id');
  expectJson(got, 'order is readable by its owner', 'data.id', orderID);

  const other = registerUser();
  const foreign = request('GET', `/v1/orders/${orderID}`, undefined, other.token);
  expectStatus(foreign, 404, 'GET /v1/orders/:id (other user)');

  // The list can be filtered by status.
  const list = request('GET', '/v1/orders?status=pending', undefined, token);
  expectStatus(list, 200, 'GET /v1/orders?status=pending');
  expectJson(list, 'list returns the pending order', 'meta.total', 1);

  // Cancel restores stock exactly once.
  const cancel = request('POST', `/v1/orders/${orderID}/cancel`, undefined, token);
  expectStatus(cancel, 200, 'POST /v1/orders/:id/cancel');
  expectJson(cancel, 'cancel flips the order status', 'data.status', 'cancelled');

  const restored = request('GET', `/v1/products/${product.id}`);
  expectJson(restored, 'cancel restores stock', 'data.stock', 5);

  const cancelAgain = request('POST', `/v1/orders/${orderID}/cancel`, undefined, token);
  expectStatus(cancelAgain, 409, 'POST /v1/orders/:id/cancel (again)');
  expectJson(cancelAgain, 'second cancel reports ORDER_ALREADY_CANCELLED', 'error.code', 'ORDER_ALREADY_CANCELLED');

  const foreignCancel = request('POST', `/v1/orders/${orderID}/cancel`, undefined, other.token);
  expectStatus(foreignCancel, 404, 'POST /v1/orders/:id/cancel (other user)');
}
