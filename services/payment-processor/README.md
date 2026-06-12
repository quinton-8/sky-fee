# SkyFee Payment Processor

The payment processor is a Go HTTP API that powers the SkyFee frontend. It manages school registration, student verification, KES-to-sats invoice creation, payment tracking, Lightning settlement, and mocked M-Pesa disbursement.

## Tech Stack

- Go 1.21
- Chi router
- PostgreSQL through `database/sql` and `lib/pq`
- Gorilla WebSocket for live payment updates
- LNbits integration for Lightning invoices
- Mock Lightning and M-Pesa services for local hackathon demos

## Project Structure

```text
cmd/server/main.go          Application entrypoint and dependency wiring
internal/handlers/          HTTP routes, request handlers, settlement flow, WebSockets
internal/db/                Store interface, PostgreSQL store, in-memory store
internal/lightning/         LNbits client and mock Lightning client
internal/mpesa/             Mock off-ramp and M-Pesa payout service
internal/models/            Shared API/domain models
```

## Run Locally

```bash
cd services/payment-processor
go mod download
go run ./cmd/server
```

The API listens on `http://localhost:8080` unless `PORT` is set.

If `DATABASE_URL` is not provided, the service uses an in-memory store with seeded schools and students. If `LNBITS_URL` or `LNBITS_INVOICE_KEY` is missing, it uses a mock Lightning client that returns fake Bolt11 invoices.

## Environment Variables

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/skyfee?sslmode=disable
LNBITS_URL=http://localhost:5000
LNBITS_INVOICE_KEY=your_lnbits_invoice_key
```

| Variable | Required | Purpose |
| --- | --- | --- |
| `PORT` | No | HTTP server port. Defaults to `8080`. |
| `DATABASE_URL` | No | PostgreSQL connection string. Without it, the API uses memory storage. |
| `LNBITS_URL` | No | LNbits base URL. Required only for real Lightning invoices. |
| `LNBITS_INVOICE_KEY` | No | LNbits invoice/read key. Required only for real Lightning invoices. |

## Database

The PostgreSQL schema and seed data live at:

```bash
database/schema.sql
```

Apply it to a local database from the repository root:

```bash
psql "$DATABASE_URL" -f database/schema.sql
```

The same sample data is also loaded into the in-memory store when no database is configured.

## API Reference

### `GET /api/schools`

Returns all enrolled schools.

### `POST /api/schools`

Creates a school.

```json
{
  "name": "Sunshine Academy",
  "paybill": "123456",
  "account_number": "SchoolFees"
}
```

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

Development helper that settles a mock payment and triggers the off-ramp/M-Pesa payout flow.

## Payment Statuses

| Status | Meaning |
| --- | --- |
| `PENDING` | Invoice has been created and is waiting for payment. |
| `PAID` | Lightning invoice settlement was received. |
| `DISBURSED` | Mock M-Pesa payout to the school completed. |
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
