// Sanity checks for the product endpoints: create, read, update and list.
import { check } from 'k6';
import { expectJson, expectStatus, registerUser, request, uniqueSuffix } from './lib.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

const MISSING_ID = '00000000-0000-0000-0000-000000000000';

export default function products() {
  const { token } = registerUser();
  const sku = `SANITY-${uniqueSuffix()}`;

  // Create canonicalizes the SKU to uppercase and stores the requested values.
  const created = request(
    'POST',
    '/v1/products',
    { sku: sku.toLowerCase(), name: 'Sanity Product', price: 15000, stock: 5 },
    token,
  );
  expectStatus(created, 201, 'POST /v1/products');
  const productID = created.json('data.id');
  check(created, {
    'create canonicalizes the SKU': (r) => r.json('data.sku') === sku,
    'create stores the requested stock': (r) => r.json('data.stock') === 5,
  });

  // Duplicate SKUs, invalid payloads and anonymous callers are rejected.
  const duplicate = request('POST', '/v1/products', { sku, name: 'Sanity Product', price: 15000, stock: 5 }, token);
  expectStatus(duplicate, 409, 'POST /v1/products (duplicate SKU)');
  expectJson(duplicate, 'duplicate SKU reports SKU_ALREADY_EXISTS', 'error.code', 'SKU_ALREADY_EXISTS');

  const invalid = request('POST', '/v1/products', { sku: `SANITY-${uniqueSuffix()}`, name: 'Missing price' }, token);
  expectStatus(invalid, 400, 'POST /v1/products (missing price)');
  expectJson(invalid, 'invalid payload reports VALIDATION_ERROR', 'error.code', 'VALIDATION_ERROR');

  const anonymous = request('POST', '/v1/products', {
    sku: `SANITY-${uniqueSuffix()}`,
    name: 'Anonymous',
    price: 1000,
    stock: 1,
  });
  expectStatus(anonymous, 401, 'POST /v1/products (no token)');

  // Reads: a known id succeeds, an unknown id is a 404.
  const got = request('GET', `/v1/products/${productID}`);
  expectStatus(got, 200, 'GET /v1/products/:id');
  expectJson(got, 'get returns the created product', 'data.sku', sku);

  const missing = request('GET', `/v1/products/${MISSING_ID}`);
  expectStatus(missing, 404, 'GET /v1/products/:id (unknown id)');

  // Update replaces the mutable fields and keeps the immutable SKU.
  const updated = request(
    'PUT',
    `/v1/products/${productID}`,
    { name: 'Sanity Product v2', price: 17000, stock: 7 },
    token,
  );
  expectStatus(updated, 200, 'PUT /v1/products/:id');
  check(updated, {
    'update persists the new price': (r) => r.json('data.price') === 17000,
    'update persists the new stock': (r) => r.json('data.stock') === 7,
    'update keeps the SKU': (r) => r.json('data.sku') === sku,
  });

  const updateMissing = request('PUT', `/v1/products/${MISSING_ID}`, { name: 'Ghost', price: 1000, stock: 1 }, token);
  expectStatus(updateMissing, 404, 'PUT /v1/products/:id (unknown id)');

  // List honours pagination.
  const list = request('GET', '/v1/products?limit=1');
  expectStatus(list, 200, 'GET /v1/products');
  check(list, {
    'list respects the limit': (r) => r.json('data').length === 1,
    'list reports the applied limit': (r) => r.json('meta.limit') === 1,
  });
}
