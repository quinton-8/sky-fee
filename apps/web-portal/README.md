# SkyFee Web Portal

The SkyFee web portal is a Next.js App Router frontend for school-fee payments. The current app opens on a dashboard that lets a parent select a school, verify a student, generate a KES-denominated Lightning invoice, and watch settlement progress through to M-Pesa disbursement.

## Tech Stack

- Next.js 14 App Router
- React 18
- TypeScript
- Tailwind CSS
- Lucide React icons
- Native `fetch` for backend and price feed calls
- Polling-based payment status updates, with backend WebSocket support available

## Project Structure

```text
app/layout.tsx             Root layout, metadata, and top navigation
app/page.tsx               Redirects `/` to `/dashboard`
app/dashboard/page.tsx     Main school-fee payment dashboard and checkout modal
app/register/page.tsx      Institution registration/provisioning screen
app/globals.css            Global Tailwind imports and base body colors
lib/finance.ts             Fee and fiat-to-satoshi helper logic
package.json               Next.js scripts and dependencies
```

## Run Locally

Start the backend first:

```bash
cd services/payment-processor
go run ./cmd/server
```

Then start the frontend:

```bash
cd apps/web-portal
npm install
npm run dev
```

Open `http://localhost:3000`.

## Environment Variables

Set this in your shell or in `apps/web-portal/.env.local` when you need a non-default backend URL:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

If `NEXT_PUBLIC_API_URL` is not set, the dashboard currently defaults to `http://localhost:8080`.

## Main Screens

- `/` - redirects to `/dashboard`.
- `/dashboard` - primary payment operations screen with system stats, school/student verification, invoice creation, QR code display, invoice copy action, status polling, and mock settlement trigger.
- `/register` - institution registration form that posts onboarding data to `http://localhost:8080/api/v1/schools/register`.

Note: the current Go backend does not expose `/api/v1/schools/register`, so the registration page is an integration placeholder until that endpoint is implemented.

## Dashboard Flow

1. Load schools from `GET /api/schools`; if unavailable, use the local seeded school list.
2. Fetch BTC-KES spot pricing from Coinbase; if unavailable, use the cached fallback rate.
3. Select a school and verify an admission number with `GET /api/schools/{schoolID}/students/{admissionNumber}`.
4. Enter the payer name and KES amount.
5. Create a payment through `POST /api/payments`.
6. Display the returned Lightning invoice as text and a QR code.
7. Poll `GET /api/payments/{paymentID}` roughly every two seconds until settlement completes.
8. In local mock mode, the checkout modal's `Continue (I've Paid)` action calls `POST /api/payments/{paymentID}/settle`.

## Local Demo Data

When the backend runs without `DATABASE_URL`, it uses in-memory seed data:

| School | Admission number | Student | Grade |
| --- | --- | --- | --- |
| Alliance High School | `AHS-8899` | John Kiprop | Form 3 Green |
| Alliance High School | `AHS-9012` | David Mwangi | Form 1 Blue |
| Kenya High School | `KHS-4455` | Sarah Cherono | Form 4 East |
| Lenana School | `LEN-1234` | Joseph Kamau | Form 2 West |

## Backend Contract Used By The Frontend

- `GET /api/schools`
- `GET /api/schools/{schoolID}/students/{admissionNumber}`
- `POST /api/payments`
- `GET /api/payments/{paymentID}`
- `POST /api/payments/{paymentID}/settle`

The backend also exposes `GET /api/payments/{paymentID}/ws`, but the current dashboard implementation polls the payment endpoint instead.

## Scripts

```bash
npm run dev      # Start local development server on port 3000
npm run build    # Create production build
npm start        # Serve production build on port 3000
npm run lint     # Run Next.js linting
```
