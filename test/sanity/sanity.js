// Post-deploy sanity suite: runs every per-resource file in sequence against
// a deployed environment. Every check must pass (threshold below), so a
// broken release fails the deploy workflow and can be rolled back.
//
// Each file is also standalone: k6 run test/sanity/orders.js
import auth from './auth.js';
import health from './health.js';
import orders from './orders.js';
import products from './products.js';

// One virtual user, one pass: a functional gate, not a load test.
export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

export default function sanity() {
  health();
  auth();
  products();
  orders();
}
