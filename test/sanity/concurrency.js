// Concurrency probe for a deployed environment: fires parallel requests at
// the same product/order and asserts the invariants that must hold under any
// interleaving — exactly one winner, no server errors, stock conserved.
//
// The deterministic race proofs live in test/integration (Go + real Postgres
// with synchronized barriers). This probe cannot force a specific
// interleaving, but it exercises the deployed environment (connection pool,
// replicas) and fails on any outcome a correct implementation cannot produce.
import { check } from 'k6';
import http from 'k6/http';
import { batchRequest, createProduct, expectStatus, registerUser, request } from './lib.js';

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

const RACERS = 5;

export default function concurrency() {
  raceForLastItem();
  raceToCancelOnce();
  raceMixedCreateCancel();
}

// raceForLastItem: RACERS users order the same product that has a single unit
// left. Exactly one order may win; everyone else must get INSUFFICIENT_STOCK.
function raceForLastItem() {
  const racers = [];
  for (let i = 0; i < RACERS; i++) {
    racers.push(registerUser());
  }
  const product = createProduct(racers[0].token, { stock: 1 });

  const responses = http.batch(
    racers.map((racer) =>
      batchRequest('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 1 }] }, racer.token),
    ),
  );

  const winners = responses.filter((res) => res.status === 201);
  const losers = responses.filter((res) => res.status === 409);
  const serverErrors = responses.filter((res) => res.status >= 500);

  check(null, {
    'last item: exactly one order wins': () => winners.length === 1,
    [`last item: other ${RACERS - 1} orders are rejected`]: () => losers.length === RACERS - 1,
    'last item: losers report INSUFFICIENT_STOCK': () =>
      losers.every((res) => res.json('error.code') === 'INSUFFICIENT_STOCK'),
    'last item: no server errors during the race': () => serverErrors.length === 0,
  });

  const after = request('GET', `/v1/products/${product.id}`);
  expectStatus(after, 200, 'last item: product still readable');
  check(after, { 'last item: stock ends at exactly zero': (r) => r.json('data.stock') === 0 });
}

// raceToCancelOnce: the owner fires RACERS cancels at the same order in
// parallel. Exactly one may win; the rest must get ORDER_ALREADY_CANCELLED,
// and the stock must be restored exactly once.
function raceToCancelOnce() {
  const owner = registerUser();
  const product = createProduct(owner.token, { stock: 5 });

  const created = request('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 2 }] }, owner.token);
  expectStatus(created, 201, 'cancel race: order created');
  const orderID = created.json('data.id');

  const responses = http.batch(
    Array.from({ length: RACERS }, () =>
      batchRequest('POST', `/v1/orders/${orderID}/cancel`, undefined, owner.token),
    ),
  );

  const winners = responses.filter((res) => res.status === 200);
  const losers = responses.filter((res) => res.status === 409);
  const serverErrors = responses.filter((res) => res.status >= 500);

  check(null, {
    'cancel race: exactly one cancel wins': () => winners.length === 1,
    [`cancel race: other ${RACERS - 1} cancels are rejected`]: () => losers.length === RACERS - 1,
    'cancel race: losers report ORDER_ALREADY_CANCELLED': () =>
      losers.every((res) => res.json('error.code') === 'ORDER_ALREADY_CANCELLED'),
    'cancel race: no server errors during the race': () => serverErrors.length === 0,
  });

  const after = request('GET', `/v1/products/${product.id}`);
  expectStatus(after, 200, 'cancel race: product still readable');
  check(after, { 'cancel race: stock is restored exactly once': (r) => r.json('data.stock') === 5 });
}

// raceMixedCreateCancel: buyers create orders while owners cancel theirs, all
// in parallel on the same product — stock decreases and increases at once.
// With ample stock every request must succeed in any interleaving, and the
// final stock must match the exact arithmetic of both directions.
function raceMixedCreateCancel() {
  const buyers = [registerUser(), registerUser()];
  const owners = [registerUser(), registerUser()];
  const product = createProduct(buyers[0].token, { stock: 10 });

  // Seed two pending orders so there is something to cancel.
  const seeded = owners.map((owner) => {
    const res = request('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 1 }] }, owner.token);
    expectStatus(res, 201, 'mixed race: seeded order created');
    return res.json('data.id');
  });

  const responses = http.batch([
    ...buyers.map((buyer) =>
      batchRequest('POST', '/v1/orders', { items: [{ sku: product.sku, quantity: 1 }] }, buyer.token),
    ),
    ...seeded.map((orderID, i) =>
      batchRequest('POST', `/v1/orders/${orderID}/cancel`, undefined, owners[i].token),
    ),
  ]);

  const creates = responses.slice(0, buyers.length);
  const cancels = responses.slice(buyers.length);

  check(null, {
    'mixed race: every create succeeds': () => creates.every((res) => res.status === 201),
    'mixed race: every cancel succeeds': () => cancels.every((res) => res.status === 200),
    'mixed race: creates return pending orders': () => creates.every((res) => res.json('data.status') === 'pending'),
    'mixed race: cancels return cancelled orders': () =>
      cancels.every((res) => res.json('data.status') === 'cancelled'),
    'mixed race: no server errors during the race': () => responses.every((res) => res.status < 500),
  });

  const after = request('GET', `/v1/products/${product.id}`);
  expectStatus(after, 200, 'mixed race: product still readable');
  // 10 initial − 2 seeded − 2 new creates + 2 restores = 8, whatever the order.
  check(after, { 'mixed race: stock matches the exact arithmetic': (r) => r.json('data.stock') === 8 });
}
