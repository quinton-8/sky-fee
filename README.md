# SkyFee

SkyFee is a hackathon prototype for Kenyan school-fee payments over Bitcoin Lightning. A parent selects a partner school, verifies a student's admission number, creates a KES-denominated Lightning invoice, and follows settlement through BTC-to-KES off-ramp logic and M-Pesa payout.

The repository is split into a Next.js frontend and a Go payment processor backend:

```text
apps/web-portal/                 Next.js App Router web portal
services/payment-processor/      Go API for schools, students, invoices, settlement, and payouts
database/schema.sql              PostgreSQL schema and seed data
docker-compose.yml               Placeholder; currently empty
```

## Current Capabilities

<<<<<<< HEAD
- Lists enrolled schools and verifies students by admission number.
- Converts KSH fee amounts to satoshis using a BTC/KES exchange rate.
- Creates Lightning invoices through LNbits when configured, or a mock invoice locally.
- Tracks payment status with WebSockets and polling fallback.
- Simulates off-ramp and M-Pesa school payout after invoice settlement.
- Falls back to an in-memory store when `DATABASE_URL` is not configured.
=======
- Dashboard-first web portal at `/dashboard`; `/` redirects there.
- Institution registration screen at `/register` for the onboarding concept UI.
- School selection and student verification against the backend, with frontend fallback seed data when the API is unavailable.
- KES-to-satoshi conversion using a live Coinbase BTC-KES price feed where available, with a local fallback rate.
- Lightning invoice generation through LNbits when configured, or a mock invoice locally.
- Payment status polling from the frontend, plus backend WebSocket support for live updates.
- Mock settlement helper that moves payments from `PENDING` to `PAID` to `DISBURSED`.
- M-Pesa payout layer that defaults to a mock service, with optional Safaricom Daraja and off-ramp provider configuration.
- PostgreSQL storage when `DATABASE_URL` is available, or an in-memory store with seeded demo data when it is not.
>>>>>>> 5673aa9 (docs: refresh READMEs for current SkyFee architecture)

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
npm run dev
```

Open `http://localhost:3000`. The frontend uses `NEXT_PUBLIC_API_URL` when set and otherwise defaults to `http://localhost:8080`.

For a no-database demo, leave `DATABASE_URL` unset. The backend will seed sample schools and students in memory:

| School | Admission number | Student | Grade |
| --- | --- | --- | --- |
| Alliance High School | `AHS-8899` | John Kiprop | Form 3 Green |
| Alliance High School | `AHS-9012` | David Mwangi | Form 1 Blue |
| Kenya High School | `KHS-4455` | Sarah Cherono | Form 4 East |
| Lenana School | `LEN-1234` | Joseph Kamau | Form 2 West |

## Local Payment Demo

1. Start the backend and frontend.
2. Go to `http://localhost:3000/dashboard`.
3. Select a seeded school and enter a matching admission number.
4. Verify the student.
5. Enter the payer name and KES amount.
6. Create the Lightning invoice.
7. In local mock mode, click `Continue (I've Paid)` in the checkout modal to trigger settlement.

You can also settle directly against the API if you are testing with a payment ID returned by `POST /api/payments`:

```bash
curl -X POST http://localhost:8080/api/payments/<payment_id>/settle
```

The payment should move from `PENDING` to `PAID` and then `DISBURSED`, with a mock M-Pesa receipt.

## Environment

Frontend:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Backend core:

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/skyfee?sslmode=disable
LNBITS_URL=http://localhost:5000
LNBITS_INVOICE_KEY=your_lnbits_invoice_key
```

Backend M-Pesa/off-ramp, optional:

```env
MPESA_CONSUMER_KEY=your_daraja_consumer_key
MPESA_CONSUMER_SECRET=your_daraja_consumer_secret
MPESA_ENV=sandbox
MPESA_SHORTCODE=600192
MPESA_INITIATOR_NAME=testapi
MPESA_INITIATOR_PASSWORD=your_initiator_password
MPESA_SECURITY_CREDENTIAL=pre_encrypted_security_credential
MPESA_CERT_PATH=/path/to/safaricom/cert.pem
MPESA_CALLBACK_URL=https://your-domain.example/mpesa
OFFRAMP_PROVIDER=kotanipay
OFFRAMP_API_KEY=your_offramp_api_key
OFFRAMP_API_URL=https://provider.example/api
```

All backend values are optional for a local mock demo. `PORT` defaults to `8080`; missing database, LNbits, and M-Pesa credentials activate local mock/fallback services.

## Database

Apply the PostgreSQL schema from the repository root:

```bash
psql "$DATABASE_URL" -f database/schema.sql
```

The schema creates `schools`, `students`, and `payments`, and seeds the same demo records used by the in-memory store.

## Documentation

- Frontend README: [apps/web-portal/README.md](apps/web-portal/README.md)
- Backend README: [services/payment-processor/README.md](services/payment-processor/README.md)
