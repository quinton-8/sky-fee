# SkyFee Web Portal

The SkyFee web portal is a Next.js frontend for parents and schools. It lets a parent choose a school, verify a student by admission number, create a Lightning invoice for a KES school-fee payment, and track settlement through to the mocked M-Pesa payout.

## Tech Stack

- Next.js 14
- React 18
- TypeScript
- Native fetch API for backend calls
- WebSocket status updates with polling fallback

## Project Structure

```text
src/app/page.tsx             Landing page
src/app/schools/page.tsx     Student verification and payment flow
src/app/parents/page.tsx     Payment status lookup
src/components/              UI components for schools, checkout, invoice, status, toast
src/lib/api.ts               Backend API client
src/lib/types.ts             Shared frontend types
```

## Run Locally

Start the backend first from the repository root:

```bash
cd services/payment-processor
go run ./cmd/server
```

Then start the frontend:

```bash
cd apps/web-portal
npm install
cp .env.example .env.local
npm run dev
```

Open `http://localhost:3000`.

## Environment Variables

`.env.example` contains the local default:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Set `NEXT_PUBLIC_API_URL` to the deployed payment processor URL when running against a remote backend. If this variable is empty, the API client uses same-origin relative `/api` routes.

## Main Screens

- `/` - product entry page with links into the payment and parent flows.
- `/schools` - school selection, student verification, and payment invoice creation.
- `/parents?paymentId=<payment_id>` - payment status tracking for an existing payment.

## Local Demo Data

When the backend runs without `DATABASE_URL`, it uses in-memory seed data:

| School | Admission number | Student |
| --- | --- | --- |
| Alliance High School | `AHS-8899` | John Kiprop |
| Alliance High School | `AHS-9012` | David Mwangi |
| Kenya High School | `KHS-4455` | Sarah Cherono |
| Lenana School | `LEN-1234` | Joseph Kamau |

## Payment Flow

1. Visit `/schools`.
2. Select a school and enter a matching admission number.
3. Enter the parent/payer name and the amount in KES.
4. Submit the form to create a Lightning invoice.
5. Track the payment using the returned payment ID.

For local mock mode, settle a payment from the backend:

```bash
curl -X POST http://localhost:8080/api/payments/<payment_id>/settle
```

The status component listens over WebSocket at `/api/payments/{paymentID}/ws`. If the socket cannot connect, it polls `GET /api/payments/{paymentID}` every few seconds.

## Scripts

```bash
npm run dev      # Start local development server
npm run build    # Create production build
npm start        # Serve production build
npm run lint     # Run Next.js linting
```

## Backend Contract

The frontend currently uses these backend endpoints:

- `GET /api/schools`
- `POST /api/schools`
- `GET /api/schools/{schoolID}/students/{admissionNumber}`
- `POST /api/payments`
- `GET /api/payments/{paymentID}`
- `GET /api/payments/{paymentID}/ws`
