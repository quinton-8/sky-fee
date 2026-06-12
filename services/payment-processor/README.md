# SkyFee Payment Processor

The payment processor is a Go HTTP API that powers the SkyFee frontend. It manages school lookup, student verification, KES-to-sats invoice creation, payment tracking, Lightning settlement, BTC-to-KES off-ramp logic, and M-Pesa payout.

## Tech Stack

- Go 1.21
- Chi router and middleware
- PostgreSQL through `database/sql` and `lib/pq`
- In-memory store fallback for local demos
- Gorilla WebSocket for live payment updates
- LNbits integration for Lightning invoices
- Coinbase BTC-KES spot pricing with fallback rates
- Mock and optional real M-Pesa/Safaricom Daraja payout services

## Project Structure

```text
cmd/server/main.go          Application entrypoint, environment loading, and dependency wiring
internal/handlers/          HTTP routes, request handlers, settlement flow, and WebSockets
internal/db/                Store interface, PostgreSQL store, and in-memory store
internal/lightning/         LNbits client, mock Lightning client, and BTC-KES price lookup
internal/mpesa/             Mock/real off-ramp and M-Pesa payout services
internal/models/            Shared API/domain models
```

## Run Locally

```bash
cd services/payment-processor
go mod download
go run ./cmd/server
```

The API listens on `http://localhost:8080` unless `PORT` is set.

If `DATABASE_URL` is not provided, the service uses an in-memory store with seeded schools and students. If `LNBITS_URL` or `LNBITS_INVOICE_KEY` is missing, it uses a mock Lightning client that returns fake Bolt11 invoices. If M-Pesa credentials are missing, it uses a mock payout service.

## Environment Variables

Core:

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/skyfee?sslmode=disable
LNBITS_URL=http://localhost:5000
LNBITS_INVOICE_KEY=your_lnbits_invoice_key
```

M-Pesa/off-ramp, optional:

```env
MPESA_CONSUMER_KEY=your_daraja_consumer_key
MPESA_CONSUMER_SECRET=your_daraja_consumer_secret
MPESA_ENV=sandbox
MPESA_SHORTCODE=600192
MPESA_INITIATOR_NAME=testapi
MPESA_INITIATOR_PASSWORD=your_initiator_password
MPESA_SECURITY_CREDENTIAL=pre_encrypted_security_credential
MPESA_CERT_PEM="-----BEGIN CERTIFICATE-----..."
MPESA_CERT_PATH=/path/to/safaricom/cert.pem
MPESA_CALLBACK_URL=https://your-domain.example/mpesa
MPESA_B2C_COMMAND_ID=BusinessPayment
MPESA_B2B_COMMAND_ID=BusinessPayBill
OFFRAMP_PROVIDER=kotanipay
OFFRAMP_API_KEY=your_offramp_api_key
OFFRAMP_API_URL=https://provider.example/api
```

| Variable | Required | Purpose |
| --- | --- | --- |
| `PORT` | No | HTTP server port. Defaults to `8080`. |
| `DATABASE_URL` | No | PostgreSQL connection string. Without it, the API uses memory storage. |
| `LNBITS_URL` | No | LNbits base URL. Required only for real Lightning invoices. |
| `LNBITS_INVOICE_KEY` | No | LNbits invoice/read key. Required only for real Lightning invoices. |
| `MPESA_CONSUMER_KEY` / `MPESA_CONSUMER_SECRET` | No | Enable real Safaricom Daraja integration when both are set. |
| `OFFRAMP_API_KEY` | No | Enables an external off-ramp request; otherwise the service returns the calculated KES value. |

## Database

The PostgreSQL schema and seed data live at:

```bash
database/schema.sql
```

Apply it to a local database from the repository root:

```bash
psql "$DATABASE_URL" -f database/schema.sql
```

The same sample data is loaded into the in-memory store when no database is configured.

## API Reference

### `GET /api/schools`

Returns all enrolled schools.

### `GET /api/schools/{schoolID}/students/{admissionNumber}`

Verifies that a student belongs to a school.

### `POST /api/payments`

Creates a Lightning invoice for a school-fee payment.

```json
{
  "school_id": 1,
  "student_admission_number": "AHS-8899",
  "parent_name": "Mary Kiprop",
  "amount_kes": 2000
}
```

Response:

```json
{
  "payment_id": "uuid",
  "amount_kes": 2000,
  "amount_sats": 22222,
  "lightning_invoice": "lnbc...",
  "payment_hash": "hash",
  "btc_kes_rate": 9000000
}
```

### `GET /api/payments/{paymentID}`

Returns the current payment record and status.

### `GET /api/payments/{paymentID}/ws`

Opens a WebSocket that streams payment updates for the given payment.

### `POST /api/webhooks/lightning`

Webhook endpoint for Lightning settlement notifications.

```json
{
  "payment_hash": "hash"
}
```

### `POST /api/payments/{paymentID}/settle`

Development helper that settles a pending mock payment and triggers the off-ramp/M-Pesa payout flow.

## Settlement Flow

1. `POST /api/payments` validates the school and student, fetches BTC-KES pricing, creates a Lightning invoice, and stores the payment as `PENDING`.
2. Settlement arrives through `POST /api/webhooks/lightning` or the local mock settle endpoint.
3. The payment moves to `PAID`.
4. The M-Pesa service converts sats back to KES using the current rate or a fallback rate.
5. The payout is sent to the school's paybill/account reference.
6. The payment moves to `DISBURSED` with an M-Pesa receipt, or `FAILED` if any required step fails.

## Payment Statuses

| Status | Meaning |
| --- | --- |
| `PENDING` | Invoice has been created and is waiting for payment. |
| `PAID` | Lightning invoice settlement was received. |
| `DISBURSED` | M-Pesa payout to the school completed. |
| `FAILED` | Settlement, off-ramp, school lookup, or payout failed. |

## Useful Commands

```bash
go run ./cmd/server
go test ./...
go fmt ./...
```

Mock settlement:

```bash
curl -X POST http://localhost:8080/api/payments/<payment_id>/settle
```
