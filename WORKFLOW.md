# KAZI Application Workflows

This document is the source of truth for every user-facing flow. It reflects the code as it exists today: payments run through the **KAZI payment microservice** (`PAYMENT_SERVICE_URL`), which talks to **Snippe** (Tanzanian gateway). KAZI authenticates to the microservice with a **Casdoor machine-to-machine JWT**. There is no AzamPay anywhere anymore.

Every workflow lists the mobile screen driving each step so the frontend and backend stay in lockstep.

---

## Payment architecture (read this first)

```
Customer phone ──USSD/PIN──┐
                           ▼
KAZI backend ──Casdoor JWT──▶ Payment microservice ──▶ Snippe ──▶ Mobile money / Card
     ▲                                 │
     └──── webhooks (shared secret) ◀──┘
           POST /internal/payment-webhook
           events: payment.completed | payment.failed | payout.completed
```

- Auth to the microservice: `POST {CASDOOR_ENDPOINT}/api/login/oauth/access_token` with `grant_type=client_credentials` (`CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET`). Token is cached and refreshed 30s before expiry (`internal/payment/casdoor_token.go`).
- Webhooks to KAZI are verified with the `X-Webhook-Secret` header (`PAYMENT_WEBHOOK_SECRET`).
- Money never sits in KAZI's database. Escrow lives in the microservice/Snippe. The `maid_wallets` table is a **display ledger only**.

### Payment status state machine (bookings.payment_status)

```
unpaid → collection_pending → paid_held_escrow → disbursement_pending → released_to_maid
                    │
                    ├→ failed   (payment.failed webhook → booking cancelled_payment_failed)
                    └→ expired  (no webhook + microservice says dead after 24h → booking cancelled_by_system)
```

### Booking status state machine (bookings.booking_status)

```
pending_maid → maid_accepted → confirmed → in_progress → completed
      │              │
      │              └→ cancelled_payment_failed / cancelled_by_system
      └→ cancelled_by_maid
```

---

## WORKFLOW A: Registration & Authentication

**Screens:** `WelcomeScreen` → `PhoneEntryScreen` → `OtpScreen` → (`RoleSelectionScreen` → `ProfileSetupScreen` for new users only)

1. Welcome shows a single **"Anza Sasa / Get Started"** button (no pre-login role choice — role is an account property chosen once at signup).
2. `PhoneEntryScreen`: 255XXXXXXXXX phone → `POST /api/v1/auth/request-otp`.
3. `OtpScreen`: 6 boxes, 10-min expiry countdown, resend after expiry. **3 wrong entries locks input for 10 minutes with a visible countdown.** → `POST /api/v1/auth/verify-otp`.
4. New user → `RoleSelectionScreen` (customer / maid / both) → `ProfileSetupScreen` (name + optional photo) → `POST /api/v1/auth/complete-profile`. Maid role continues to Workflow B.
5. Returning user → straight to the role-based home (see Workflow J). JWT stored in `flutter_secure_storage`.
6. **Auto-login:** the router guard redirects any logged-in user away from welcome/phone/OTP directly to their home. A day-to-day maid opens the app and lands on `MaidHomeScreen` with zero taps.

## WORKFLOW B: Maid Verification

**Screens:** `VerificationIntroScreen` (profile form) → `VerificationVideoScreen` (15s selfie, real camera) → `VerificationIDScreen` (ID photo + number) → `VerificationPendingScreen` (polls status)

- Endpoints: `POST /api/v1/maid/verification/upload-video`, `upload-id`, `submit` (MinIO storage).
- Admin approves/rejects via `POST /api/v1/admin/verifications/approve|reject`.
- Pending screen auto-navigates to home on approval, back to the form on rejection. An unverified maid is always redirected to `VerificationPendingScreen` by the router guard.

## WORKFLOW C: Booking Creation

**Screens:** `HomeScreen` (preview of 5) / `MaidListScreen` (full browse + search + filter) → `MaidDetailScreen` → `BookingFormScreen`

1. Browse: `GET /api/v1/maids/search` (public). Detail: `GET /api/v1/maids/:maid_id` + `GET /api/v1/reviews/maid/:maid_id`.
2. `BookingFormScreen`: service, date/time pickers, duration, map location picker, live price (subtotal + 15% fee) → `POST /api/v1/bookings/create` → booking `pending_maid`, maid notified.
3. **Maid side** (`MaidHomeScreen` → `MaidJobRequestScreen`): 2-hour countdown, Accept / Decline (with reason sheet).
   - Accept: `POST /api/v1/maid/bookings/:id/accept` → `maid_accepted`, customer notified to pay.
   - **Concurrency guard:** accept re-checks the maid's calendar — if another booking in `maid_accepted`/`confirmed`/`in_progress` overlaps the same date/time window, accept is rejected ("you already have another booking at this time"). Two customers can *request* the same slot; the first accepted one wins.
   - Decline: `POST /api/v1/maid/bookings/:id/decline` → `cancelled_by_maid`, customer rebooks.

## WORKFLOW D: Payment (Snippe via payment microservice)

**Screen:** `PaymentScreen` (from `BookingDetailScreen` "Pay Now" when `maid_accepted`)

### D1 — Mobile money (default)
1. Customer picks provider (M-Pesa / Tigo / Airtel / Halopesa), enters number → `POST /api/v1/bookings/:id/initiate-payment` `{payment_method:"mobile", provider, phone_number}`.
2. Backend calls microservice `POST /v1/collect/mobile` → stores `payment_collection_transaction_id`, sets `collection_pending`.
3. App shows the USSD instruction view and **polls `GET /bookings/:id` every 5s (max 3 minutes)**.
4. Customer enters PIN on their phone; Snippe processes.

### D2 — Card
1. Customer picks **"Lipa kwa Kadi"** → same endpoint with `{payment_method:"card"}` (+ optional billing fields; defaults: Dar es Salaam / TZ, deep links `kazi://payment/success|cancel`).
2. Backend calls `POST /v1/collect/card` → returns `payment_url` (hosted Snippe checkout).
3. **The app opens `payment_url` in an external browser/WebView** and shows a card-pending view with a "reopen payment page" button. Polling continues in the background exactly like mobile money — the redirect back to the app is cosmetic; the webhook is the source of truth.

### D3 — Webhook confirmation
`POST /internal/payment-webhook` (header `X-Webhook-Secret`), body `{transaction_id, event_type}`:
- `payment.completed` → find booking by collection tx id → microservice `POST /v1/escrow/hold` → `payment_status=paid_held_escrow`, `booking_status=confirmed` → maid notified.
- `payment.failed` → `payment_status=failed`, `booking_status=cancelled_payment_failed` → customer notified.

### D4 — What the customer sees
- Poll finds `paid_held_escrow` → green "Malipo yamekamilika!" + "View booking".
- Poll finds `failed`/`expired` → error + retry (returns to provider selection).
- 3-minute poll timeout → timeout message + retry. **The pending screen can never get stuck.**

### D5 — Lost webhook / timeout safety net (background job, hourly)
- Bookings in `collection_pending` older than 30 min: backend asks the microservice `GET /v1/transactions/:id`.
  - completed → escrow held, booking confirmed (recovers a lost webhook).
  - failed → cleanup as in payment.failed.
  - still unknown after 24h → `payment_status=expired`, `booking_status=cancelled_by_system`, both parties notified, maid slot freed.

## WORKFLOW E: Service Day

**Maid screen:** `MaidActiveJobScreen` · **Customer screens:** `BookingDetailScreen` + `TrackingScreen`

1. **"Ninaelekea" (I'm on my way):** maid taps → GPS permission → location POSTed every 15s to `POST /api/v1/bookings/:id/location`. Button then swaps to the Arrive banner.
2. **Customer tracking:** `TrackingScreen` polls `GET /bookings/:id/location` every 10s (map, pulsing `MaidAvatar` marker, ETA card = distance/4 km/h walking speed) and `GET /bookings/:id` every 15s for status.
3. **"Nimefikisha" (Arrived):** `POST /api/v1/maid/bookings/:id/arrive` → `in_progress`. Customer's tracking map swaps the maid marker for a green home icon + "Msaidizi amefika!" banner.
4. **"Nimekamilisha Kazi" (Work done):** confirm dialog → `POST /api/v1/maid/bookings/:id/complete` → booking **stays `in_progress`**, `service_completed_at` set, timeline event logged, customer notified "maid has finished, please confirm". Maid screen shows "Waiting for customer confirmation" (disabled state).

## WORKFLOW F: Completion Confirmation & Payout Release

**Customer screen:** `BookingDetailScreen` (polls every 15s while in_progress, so the confirm button appears without a manual refresh)

1. Customer taps **"Thibitisha Kukamilika" (Confirm & Release Payment)** → `POST /api/v1/bookings/:id/confirm`:
   - `booking_status=completed`, timeline event.
   - Microservice `POST /v1/escrow/release` `{collection_transaction_id, recipient_phone: maid phone, recipient_name, narration: "Job payment BK..."}`.
   - Stores returned `disbursement_transaction_id`, sets `payment_status=disbursement_pending`.
   - **Returns success to the customer immediately — does not wait for the payout.**
2. Microservice sends `payout.completed` webhook → KAZI finds the booking by `payment_disbursement_transaction_id` → `payment_status=released_to_maid` → **credits `maid_wallets` (display only — real money already sent by Snippe)** → maid notified "payment of TZS X has been sent to your phone".
3. Maid sees the wallet balance rise on `MaidHomeScreen` and gets the in-app notification.
4. **Auto-confirm safety net (background job, hourly):** `in_progress` bookings with `service_completed_at` older than 24h are auto-completed, escrow released, customer notified — the maid is never left unpaid because a customer forgot.

## WORKFLOW G: Maid Withdrawal — display wallet note

With Snippe escrow release paying the maid **directly to her phone** on every job, the old wallet-withdrawal flow is no longer the payout path. `GET /api/v1/maid/wallet` shows lifetime earnings (display ledger). The frontend `WithdrawalFormScreen`/`WithdrawalOtpScreen` exist and call `POST /api/v1/payouts/request` / `confirm-otp` — **these backend endpoints are not implemented** and are only needed if a hold-then-withdraw model is reintroduced. See TASKS.

## WORKFLOW H: Disputes — frontend ready, backend pending

`DisputeCreateScreen` (type + description + evidence photos) posts to `POST /api/v1/disputes`, and `BookingDetailScreen` renders a dedicated `disputed` banner. The backend routes/handlers are **not implemented yet**. See TASKS.

## WORKFLOW I: Reviews

**Screen:** `ReviewSubmitScreen` (from BookingDetail "Write a review" after completion) — `StarPicker`, comment, tag chips (backend enum: punctual/thorough/friendly/professional/fast) → `POST /api/v1/reviews`. Duplicate returns "already reviewed" and the button becomes "Reviewed ✓". Known gap: maid `average_rating` is not recalculated on new reviews (see TASKS).

## WORKFLOW J: Roles & Sessions

- Single login for everyone; roles are account properties.
- Maid-only → `MaidHomeScreen`. Customer-only → `HomeScreen`. Dual-role → last active role (persisted in secure storage as `active_role`).
- Dual-role users switch modes via a single **Settings tile** ("Switch to Work/Customer Mode") — there is no separate role-switcher screen.
- Logout clears secure storage and the in-memory session.

## WORKFLOW K: Background Jobs (implemented, hourly ticker in `internal/booking/jobs.go`)

| Job | Trigger | Action |
|---|---|---|
| Lost-webhook recovery | `collection_pending` > 30 min | Ask microservice for real status; hold escrow or fail accordingly |
| Payment expiry | `collection_pending` > 24 h and still unknown | `expired` + `cancelled_by_system`, notify both, free the slot |
| Auto-confirm | `in_progress` + `service_completed_at` > 24 h | Complete booking, release escrow, notify customer |

## Cross-cutting

- **Logging:** everything payment/booking related logs via `slog` with human-readable messages and structured fields (`booking_reference`, `transaction_id`, `amount_tzs`, ...). Webhook rejections log the remote IP; wallet-credit failures log an explicit "fix manually" warning.
- **Notifications:** in-app rows (`GET /api/v1/notifications`) + WebSocket hub at `/ws`.
- **Frontend feedback:** all user feedback uses the branded KAZI toast (top-slide banner with accent bar + icon badge) — success 3s, error 4s, tap to dismiss.
