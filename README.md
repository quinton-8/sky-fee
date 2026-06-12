# SkyFee

SkyFee is a hackathon prototype for Kenyan school-fee payments over Bitcoin Lightning. Parents select a partner school, verify a student's admission number, create a Lightning invoice for a KES-denominated fee amount, then track settlement and mocked M-Pesa disbursement to the school.

The project is split into a Next.js frontend and a Go payment processor backend:

```text
apps/web-portal/                 Next.js web app for parents and schools
services/payment-processor/      Go API for schools, students, invoices, and payouts
database/schema.sql              PostgreSQL schema and seed data
```

## What It Does

- Lists enrolled schools and verifies students by admission number.
- Converts KES fee amounts to satoshis using a BTC/KES exchange rate.
- Creates Lightning invoices through LNbits when configured, or a mock invoice locally.
- Tracks payment status with WebSockets and polling fallback.
- Simulates off-ramp and M-Pesa school payout after invoice settlement.
- Falls back to an in-memory store when `DATABASE_URL` is not configured.

## Quick Start

Run the backend:

```bash
cd services/payment-processor
go mod download
go run ./cmd/server
```

Run the frontend in another terminal:

```bash
cd apps/web-portal
npm install
cp .env.example .env.local
npm run dev
```

Open `http://localhost:3000`. The frontend expects the backend at `http://localhost:8080` by default.

For a no-database demo, leave `DATABASE_URL` unset. The backend will seed sample schools and students in memory:

| School | Admission number | Student |
| --- | --- | --- |
| Alliance High School | `AHS-8899` | John Kiprop |
| Alliance High School | `AHS-9012` | David Mwangi |
| Kenya High School | `KHS-4455` | Sarah Cherono |
| Lenana School | `LEN-1234` | Joseph Kamau |

## Local Payment Demo

1. Start the backend and frontend.
2. Go to `http://localhost:3000/schools`.
3. Select a seeded school and enter a matching admission number.
4. Enter the payer name and KES amount to create a Lightning invoice.
5. Copy the returned `payment_id`.
6. Settle the invoice in mock mode:

```bash
curl -X POST http://localhost:8080/api/payments/<payment_id>/settle
```

The payment should move from `PENDING` to `PAID` and then `DISBURSED`, with a mock M-Pesa receipt.

## Environment

Frontend:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Backend:

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/skyfee?sslmode=disable
LNBITS_URL=http://localhost:5000
LNBITS_INVOICE_KEY=your_lnbits_invoice_key
```

All backend values are optional for a local mock demo. `PORT` defaults to `8080`.

## Documentation

- Frontend README: [apps/web-portal/README.md](apps/web-portal/README.md)
- Backend README: [services/payment-processor/README.md](services/payment-processor/README.md)
