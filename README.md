# Card Payment App

A small full-stack Stripe card payment example:

- React + TypeScript + Stripe Payment Element
- Go + Stripe Go SDK
- Redis-backed payment event storage
- Live payment report page

The server decides the charge amount (`AUD 25.00`), creates the Payment Intent, and stores its initial state. Stripe.js collects card details directly in Stripe-hosted fields; card data never passes through this application. Signed Stripe webhooks update the Redis record, and the report page reads those records through the Go API.

## Prerequisites

- Node.js 20+
- Go 1.24+
- Docker (for Redis)
- A Stripe test account and, for local webhook forwarding, the Stripe CLI

## Run locally

1. Copy `.env.example` to `.env` and fill in your Stripe test keys.
2. Start Redis:

   ```sh
   docker compose up -d redis
   ```

3. Start the backend (environment variables are deliberately explicit):

   ```sh
   cd backend
   STRIPE_SECRET_KEY=sk_test_... \
   STRIPE_WEBHOOK_SECRET=whsec_... \
   REDIS_ADDR=localhost:6379 \
   APP_ORIGIN=http://localhost:5173 \
   go run ./cmd/api
   ```

4. In a second terminal, start the frontend:

   ```sh
   cd frontend
   npm install
   VITE_STRIPE_PUBLISHABLE_KEY=pk_test_... npm run dev
   ```

5. In a third terminal, forward Stripe events. Copy the printed `whsec_...` value into the backend environment above, then restart it:

   ```sh
   stripe listen --events payment_intent.created,payment_intent.processing,payment_intent.succeeded,payment_intent.payment_failed,payment_intent.canceled --forward-to localhost:8080/api/webhooks/stripe
   ```

Open <http://localhost:5173>. Stripe's successful test card is `4242 4242 4242 4242`, with any future expiry and any CVC.

## API

- `GET /api/config` — publishable configuration needed by the browser
- `POST /api/payment-intents` — creates an AUD 25.00 Payment Intent
- `POST /api/webhooks/stripe` — verifies and stores Stripe events
- `GET /api/payments` — returns stored payments, newest first
- `GET /healthz` — checks the API and Redis connection

## Security and production notes

- Never expose `STRIPE_SECRET_KEY` or the webhook secret to the frontend.
- Amount and currency are fixed server-side. Replace the demo order with a trusted server-side cart lookup before accepting variable purchases.
- The webhook handler verifies Stripe signatures and stores event IDs in Redis to make delivery idempotent.
- Restrict `APP_ORIGIN` to the deployed frontend origin and terminate TLS at the edge.
- Redis uses append-only persistence locally; use authentication, TLS, backups, and a managed Redis service in production.

