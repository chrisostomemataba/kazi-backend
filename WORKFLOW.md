# KAZI Application Workflows

This document describes all user-facing workflows in the KAZI app. Every feature implementation should reference the relevant workflow.

---

## WORKFLOW A: User Registration & Authentication

### A1: New User Registration
**Actors:** Any user (Customer or Maid)

**Steps:**
1. User enters phone number (255XXXXXXXXX format)
2. System checks if phone exists in database
3. If new: Generate 6-digit OTP, store in `otp_codes` table with 10-minute expiry
4. Send OTP via SMS gateway
5. User enters OTP
6. System validates OTP (check code, expiry, not already used)
7. Mark OTP as used
8. User selects role: "I need help" (customer), "I want to work" (maid), or "Both"
9. Create user record in `users` table
10. Create role record(s) in `user_roles` table
11. If maid role selected: Redirect to verification workflow (Workflow B)
12. Generate JWT token with userID + roles
13. Return token to app, store securely

**Database Tables:** users, user_roles, otp_codes

**States:**
- Phone verification: unverified → OTP sent → verified
- User creation: not exists → exists with roles

**Edge Cases:**
- Phone already registered: Send OTP for login (no new user creation)
- OTP expired: Allow resend (max 3 per 10 minutes)
- Invalid OTP (3 attempts): Temporarily block phone (10 minutes)
- User wants both roles: Create TWO rows in user_roles

**API Endpoints:**
- POST /api/v1/auth/request-otp
- POST /api/v1/auth/verify-otp
- POST /api/v1/auth/complete-profile

---

## WORKFLOW B: Maid Verification

### B1: Verification Submission
**Actors:** Maid (user with maid role)

**Steps:**
1. Maid fills profile form: bio, hourly_rate, services_offered[]
2. Maid records 15-second selfie video:
   - Speaks: "Jina langu ni [name], tarehe leo ni [date], namba ya ID ni [ID number]"
   - Video uploaded to MinIO: `/verification/videos/{maid_id}_{timestamp}.mp4`
3. Maid uploads ID photo (NIDA/Voters ID/License)
   - Photo uploaded to MinIO: `/verification/ids/{maid_id}_{timestamp}.jpg`
4. Maid enters: id_number, id_type
5. Create/update record in `maid_profiles`:
   - verification_status = 'pending'
   - Store MinIO paths for video and ID
6. Create notification for admin dashboard
7. Show "Pending Verification" screen to maid

**Database Tables:** maid_profiles, notifications (admin), admin_logs

**MinIO Buckets:** taskmaid-tz/verification/

**States:**
- Verification: not_started → pending → approved/rejected

### B2: Admin Review (Manual)
**Actors:** Admin (you)

**Steps:**
1. Admin logs into web dashboard
2. Views list of pending verifications (filter: verification_status='pending')
3. Admin opens maid profile
4. Admin watches selfie video:
   - Verify face matches ID photo
   - Verify speech is clear, name/date/ID stated correctly
5. Admin views ID photo:
   - Check ID is valid (not expired, clear image, no tampering signs)
   - Cross-reference name on ID with spoken name
6. Admin makes decision:

**If Approved:**
- Update `maid_profiles`: verification_status='approved', verified_at=NOW()
- Create notification: "Hongera! Account yako imethibitishwa"
- Log in `admin_action_logs`: action='verified_maid', target_user_id
- Maid can now receive bookings

**If Rejected:**
- Update `maid_profiles`: verification_status='rejected', rejection_reason="[reason]"
- Create notification with reason: "ID photo is blurry, please resubmit"
- Log in `admin_action_logs`: action='rejected_maid'
- Maid can resubmit

**Edge Cases:**
- Suspicious documents: Flag for further review, request additional docs
- Multiple submissions: Only latest submission counts
- Resubmission after rejection: Treated as new verification

**API Endpoints:**
- POST /api/v1/admin/verifications/{maid_id}/approve
- POST /api/v1/admin/verifications/{maid_id}/reject
- GET /api/v1/admin/verifications/pending

---

## WORKFLOW C: Booking Creation (Customer Side)

### C1: Browse Maids
**Actors:** Customer

**Steps:**
1. Customer opens app, sees list of verified maids
2. System queries:
```sql
   SELECT u.*, mp.* FROM users u
   JOIN maid_profiles mp ON u.id = mp.user_id
   WHERE mp.verification_status = 'approved'
     AND mp.is_available_now = true
   ORDER BY mp.average_rating DESC
```
3. Customer can filter by: service type, price range, rating, distance
4. Customer taps maid to view full profile

**Database Tables:** users, maid_profiles, reviews (for ratings)

### C2: Create Booking
**Actors:** Customer

**Steps:**
1. Customer fills booking form:
   - service_type (cleaning, cooking, laundry, etc.)
   - booking_date, start_time, duration_hours
   - customer_address, location coordinates (optional GPS)
   - special_instructions (optional)
2. System validates:
   - Date not in past
   - Maid available on selected date/time (check `maid_availability`, `maid_blocked_dates`)
   - Duration between 2-12 hours
3. System calculates pricing:
   - hourly_rate = maid's current rate (snapshot)
   - subtotal_amount = hourly_rate × duration_hours
   - platform_commission_rate = 15% (from `platform_settings`)
   - platform_commission_amount = subtotal_amount × 0.15
   - total_amount = subtotal_amount
   - maid_payout_amount = subtotal_amount - platform_commission_amount
4. Create booking record:
   - booking_status = 'pending_maid'
   - payment_status = 'unpaid'
   - Generate reference_number: BK{YYYYMMDD}{sequence}
5. Insert into `booking_status_history`: from=null, to='pending_maid'
6. Send notification to maid: "New booking request from [customer_name]"
7. Redirect customer to wait for maid acceptance

**Database Tables:** bookings, booking_status_history, notifications

**States:**
- Booking: none → pending_maid

**Edge Cases:**
- Maid not available: Show error, suggest alternate dates
- Customer has unpaid booking: Allow (different booking)
- Maid blocked customer (future feature): Show "Unavailable"

### C3: Maid Response
**Actors:** Maid

**Steps:**
1. Maid receives notification of booking request
2. Maid views booking details: date, time, location, service, price they'll earn
3. Maid decides:

**If Accept:**
- Update booking: booking_status='maid_accepted', maid_accepted_at=NOW()
- Insert `booking_status_history`: from='pending_maid', to='maid_accepted'
- Notify customer: "Mary accepted your booking. Please proceed to payment."
- Customer redirected to payment screen (Workflow D)

**If Decline:**
- Update booking: booking_status='cancelled_by_maid', cancellation_reason
- Notify customer: "Sorry, Mary is not available. Please select another maid."
- Customer can rebook with different maid

**Database Tables:** bookings, booking_status_history, notifications

**States:**
- Booking: pending_maid → maid_accepted (accept) or cancelled_by_maid (decline)

**Edge Cases:**
- Maid doesn't respond within 2 hours: Auto-decline, notify customer
- Multiple booking requests at same time: First-come-first-served
- Maid declines after initial accept (before payment): Allowed with penalty flag

**API Endpoints:**
- GET /api/v1/bookings (list for maid)
- POST /api/v1/bookings/{id}/accept
- POST /api/v1/bookings/{id}/decline

---

## WORKFLOW D: Payment Flow (AzamPay Checkout)

### D1: Customer Initiates Payment
**Actors:** Customer

**Steps:**
1. Customer clicks "Lipa Sasa" (Pay Now)
2. App shows amount breakdown: subtotal, platform fee, total
3. Customer selects provider: M-Pesa, Tigo Pesa, Airtel Money, Halopesa
4. Customer enters mobile money number
5. System creates payment record:
   - transaction_type = 'customer_checkout'
   - amount = booking.total_amount
   - status = 'initiated'
   - booking_id, provider, account_number
6. System calls AzamPay PostCheckout API:
```json
   {
       "appName": "KAZI",
       "vendorId": "...",
       "currency": "TZS",
       "amount": "36800",
       "externalId": "booking_uuid",
       "provider": "Mpesa",
       "accountNumber": "255712345678"
   }
```
7. AzamPay returns transaction_id
8. Update payment: azampay_transaction_id, status='pending_confirmation'
9. Update booking: payment_status='payment_initiated', payment_initiated_at=NOW()
10. Show "Processing" screen to customer

**Database Tables:** payments, bookings

**States:**
- Payment: none → initiated → pending_confirmation
- Booking payment: unpaid → payment_initiated

### D2: Customer Confirms Payment
**Actors:** Customer (via phone)

**Steps:**
1. Customer receives USSD push on phone: "Lipia KAZI TZS 36,800"
2. Customer enters M-Pesa/Tigo/Airtel PIN
3. Mobile money provider processes payment
4. Money transferred from customer account to AzamPay account (your merchant account)

### D3: Webhook Confirmation
**Actors:** AzamPay (webhook) → KAZI backend

**Steps:**
1. AzamPay calls webhook: POST /api/v1/webhooks/azampay
2. Webhook receives:
```json
   {
       "transactionId": "AZ123456",
       "externalId": "booking_uuid",
       "status": "SUCCESS",
       "amount": "36800",
       "provider": "Mpesa",
       "msisdn": "255712345678"
   }
```
3. System validates webhook signature (HMAC if implemented)
4. System processes based on status:

**If SUCCESS:**
- Update payment: status='completed', completed_at=NOW()
- Update booking: payment_status='paid_held_escrow', booking_status='confirmed', payment_confirmed_at=NOW()
- Insert into `platform_escrow_wallet`:
  - booking_id, amount_held=36800, status='holding', held_at=NOW()
- Send SMS to customer: "Malipo yamekamilika! Mary atakuja 10 Feb saa 9."
- Send SMS to maid: "Booking confirmed! Kwenda: Mbezi Beach, 10 Feb, 9am"
- Create notifications for both users
- Return success response to AzamPay

**If FAILED:**
- Update payment: status='failed', failure_reason
- Update booking: payment_status='failed', booking_status='payment_failed'
- Notify customer: "Malipo hayajakamilika. Jaribu tena."
- Booking remains in failed state for 24 hours, then auto-cancels

**Database Tables:** payments, bookings, platform_escrow_wallet, notifications

**States:**
- Payment: pending_confirmation → completed/failed
- Booking: payment_initiated → confirmed (success) or payment_failed (fail)
- Escrow: none → holding

**Edge Cases:**
- Webhook arrives before API response: Handle both, prevent duplicate processing
- Webhook arrives twice (AzamPay retry): Idempotent processing (check if payment already completed)
- Webhook never arrives: Polling fallback (check payment status after 5 minutes)
- Customer cancels USSD prompt: Payment fails, booking cancellable

**API Endpoints:**
- POST /api/v1/payments/initiate
- POST /api/v1/webhooks/azampay (public, no auth)

---

## WORKFLOW E: Service Day (Job Execution)

### E1: Maid Arrival
**Actors:** Maid

**Steps:**
1. On booking day, maid opens app
2. Maid sees "Today" bookings in dashboard
3. Maid clicks "Nimefikisha" (I've arrived)
4. App requests location permission
5. App captures GPS coordinates
6. System updates booking:
   - booking_status = 'in_progress'
   - service_started_at = NOW()
   - Store GPS coordinates (optional verification)
7. Notify customer: "Mary ameanza kazi" (Mary started work)

**Database Tables:** bookings, booking_status_history

**States:**
- Booking: confirmed → in_progress

### E2: Work Completion
**Actors:** Maid → Customer

**Steps:**
1. Maid completes work, clicks "Nimekamilisha Kazi"
2. System captures end GPS coordinates (optional)
3. Update booking: service_completed_at=NOW()
4. Maid can optionally add notes, upload before/after photos
5. Notify customer: "Mary amesema kazi imekamilika. Tafadhali confirm."
6. Customer reviews work
7. Customer clicks "Kazi Imekamilika" (Confirm completion)
8. System updates booking: booking_status='completed'
9. Trigger payment release workflow (Workflow F)

**Database Tables:** bookings, booking_status_history

**States:**
- Booking: in_progress → completed

**Edge Cases:**
- Customer doesn't confirm within 24 hours: Auto-confirm, release payment
- Customer disputes completion: Create dispute (Workflow H), hold payment
- Maid marks complete prematurely: Customer can report, admin reviews

---

## WORKFLOW F: Payment Release (Maid Payout)

### F1: Release from Escrow
**Actors:** System (automatic after booking completion)

**Steps:**
1. Booking status changed to 'completed' (Workflow E, Step 7)
2. System retrieves escrow record: `WHERE booking_id = X AND status = 'holding'`
3. System calculates:
   - amount_held = 36800 (from escrow table)
   - platform_commission = 6800 (already calculated, stored in booking)
   - maid_payout = 30000 (stored in booking.maid_payout_amount)
4. Update escrow: status='released_to_maid', released_at=NOW()
5. Update booking: payment_status='released_to_maid'
6. Credit maid's wallet:
   - INSERT or UPDATE `maid_wallets`: available_balance += 30000, total_earned += 30000
   - INSERT `wallet_transactions`: transaction_type='job_completed_credit', amount=30000, related_booking_id
7. Update maid stats:
   - `maid_profiles`: total_jobs_completed += 1, total_earnings += 30000
8. Notify maid: "Hongera! Umepokea TZS 30,000. Balance yako: 30,000. Unaweza kutoa pesa."

**Database Tables:** platform_escrow_wallet, maid_wallets, wallet_transactions, maid_profiles, bookings, notifications

**States:**
- Escrow: holding → released_to_maid
- Booking payment: paid_held_escrow → released_to_maid
- Maid wallet: balance increased

**Edge Cases:**
- Escrow already released (duplicate trigger): Check status first, skip if already released
- Database transaction fails mid-way: Use DB transaction, rollback all or nothing

**API Endpoints:**
- Internal function, no direct API endpoint

---

## WORKFLOW G: Maid Withdrawal (Disbursement)

### G1: Request Withdrawal
**Actors:** Maid

**Steps:**
1. Maid opens Wallet screen
2. Sees: Available Balance: 30,000 TZS
3. Maid clicks "Toa Pesa" (Withdraw)
4. System checks minimum: TZS 10,000 (from `platform_settings`)
5. Maid enters:
   - amount: 25000
   - provider: M-Pesa
   - phone_number: 255787654321
6. System validates:
   - amount <= available_balance
   - amount >= minimum_withdrawal
7. Generate 6-digit OTP for security
8. Insert `otp_codes`: purpose='payout_confirm', expires in 10 minutes
9. Send SMS: "KAZI: Weka OTP 123456 kukubali kutoa TZS 25,000"
10. Maid enters OTP
11. System verifies OTP

**Database Tables:** maid_wallets, otp_codes, payout_requests

### G2: Process Disbursement
**Actors:** System → AzamPay

**Steps:**
1. Create payout request record:
   - maid_id, amount_requested=25000, provider='Mpesa', phone_number
   - status='processing'
   - Generate reference_number: PO{YYYYMMDD}{sequence}
2. Immediately deduct from wallet (prevents double withdrawal):
   - UPDATE `maid_wallets`: available_balance -= 25000
   - INSERT `wallet_transactions`: transaction_type='withdrawal_debit', amount=-25000
3. Call AzamPay PostDisbursement API:
```json
   {
       "source": {
           "accountNumber": "YOUR_VENDOR_ID",
           "currency": "TZS"
       },
       "destination": {
           "phoneNumber": "255787654321",
           "bankName": "Mpesa",
           "accountNumber": "255787654321",
           "currency": "TZS"
       },
       "transferDetails": {
           "type": "B2C",
           "amount": "25000"
       },
       "externalReferenceId": "payout_uuid"
   }
```
4. AzamPay returns transaction_id
5. Update payout_request: azampay_transaction_id, status='processing'
6. Create payment record:
   - transaction_type='maid_disbursement', flow_direction='outbound', status='pending'

### G3: Disbursement Confirmation
**Actors:** AzamPay → KAZI backend

**Steps:**
1. AzamPay processes transfer (usually within 5 minutes)
2. Maid receives SMS from M-Pesa: "Umepokea TZS 25,000..."
3. AzamPay calls webhook or polling confirms status

**If SUCCESS:**
- Update payout_request: status='completed', completed_at=NOW()
- Update payment: status='completed', completed_at=NOW()
- Update maid_wallets: total_withdrawn += 25000, last_withdrawal_at=NOW()
- Notify maid: "Pesa zimetumwa! Angalia M-Pesa yako."

**If FAILED:**
- Update payout_request: status='failed', failure_reason
- REFUND to wallet:
  - UPDATE maid_wallets: available_balance += 25000 (add back)
  - INSERT wallet_transactions: transaction_type='withdrawal_refund', amount=+25000
- Notify maid: "Samahani, pesa hazijatumwa. Balance yako imerudi. Jaribu tena baadaye."
- Common failure reasons: Insufficient funds in KAZI AzamPay account, invalid phone number

**Database Tables:** payout_requests, payments, maid_wallets, wallet_transactions, notifications

**States:**
- Payout: none → pending → processing → completed/failed
- Payment: initiated → pending → completed/failed
- Wallet: balance deducted immediately → refunded if failed

**Edge Cases:**
- Maid tries to withdraw while payout pending: Block, show "You have pending withdrawal"
- Maid's M-Pesa account full/blocked: Disbursement fails, refund to KAZI wallet
- KAZI AzamPay account has insufficient funds: Disbursement fails, admin alert, manual top-up needed

**API Endpoints:**
- POST /api/v1/payouts/request
- POST /api/v1/payouts/confirm-otp

---

## WORKFLOW H: Dispute Handling

### H1: Customer Raises Dispute
**Actors:** Customer

**Steps:**
1. Customer unhappy with service, clicks "Report Problem"
2. Customer fills dispute form:
   - dispute_type: 'service_not_completed', 'poor_quality', 'payment_issue', 'safety_concern'
   - description: "Maid left after 2 hours, didn't finish cleaning"
   - Upload evidence photos to MinIO: `/disputes/{booking_id}/evidence_1.jpg`
3. Create dispute record:
   - booking_id, raised_by=customer_id, status='pending'
   - Store evidence_urls (MinIO paths)
4. HOLD payment release (escrow remains frozen, don't release to maid)
5. Update booking: booking_status='disputed'
6. Notify maid: "Customer amereport issue. Tafadhali jibu."
7. Notify admin for review

**Database Tables:** disputes, bookings, platform_escrow_wallet (status remains 'holding')

**States:**
- Booking: completed/in_progress → disputed
- Escrow: holding (no change, prevent release)

### H2: Admin Resolution
**Actors:** Admin

**Steps:**
1. Admin reviews dispute details:
   - Customer complaint + evidence
   - Maid's response (if any)
   - Booking details, messages between parties
2. Admin makes decision:

**Option A: Customer Wins (Full Refund)**
- Update dispute: status='resolved_customer_favor', resolved_at=NOW()
- Update escrow: status='refunded_to_customer', refunded_at=NOW()
- Create payment record: transaction_type='refund_to_customer', amount=36800
- Initiate AzamPay refund (if supported) OR manual bank transfer
- Update booking: payment_status='refunded'
- Notify customer: "Pesa zako zimerudi: TZS 36,800"
- Notify maid: "Dispute decided against you. Hakuna malipo."
- Flag maid profile (if multiple disputes, consider suspension)

**Option B: Maid Wins (Full Payment)**
- Update dispute: status='resolved_maid_favor', resolved_at=NOW()
- Release payment normally (Workflow F)
- Notify both parties of decision

**Option C: Partial Refund (Compromise)**
- Update dispute: status='resolved_partial_refund', refund_amount=15000
- Split escrow:
  - Customer refund: 15000
  - Maid payout: 21800 (36800 - 15000, then minus commission)
- Process partial refund and partial payout
- Notify both parties

**Database Tables:** disputes, platform_escrow_wallet, payments, bookings, maid_wallets, notifications

**States:**
- Dispute: pending → under_review → resolved_*
- Escrow: holding → released/refunded/split

**Edge Cases:**
- Customer raises dispute after payment released: More complex, may require reversing maid payout
- Both parties submit evidence: Admin weighs both sides
- Maid doesn't respond: Decision based on customer evidence only
- Fraudulent disputes: Flag customer, pattern analysis

**API Endpoints:**
- POST /api/v1/disputes
- POST /api/v1/admin/disputes/{id}/resolve

---

## WORKFLOW I: Review & Rating

### I1: Customer Reviews Maid
**Actors:** Customer

**Steps:**
1. After booking completion, customer prompted to review
2. Customer submits:
   - rating: 5 stars
   - comment: "Very good work, very clean!"
   - review_tags: ['punctual', 'thorough', 'friendly']
   - Optional: Photos of completed work
3. Create review record:
   - booking_id (ensure one review per booking)
   - reviewer_id=customer_id, reviewee_id=maid_id
   - rating, comment, review_tags
4. Recalculate maid's average rating:
```
   new_average = (old_average × total_reviews + new_rating) / (total_reviews + 1)
```
5. Update maid_profiles:
   - average_rating = new calculated value
   - total_reviews += 1
6. Notify maid: "Umepata review mpya: 5 stars!"

**Database Tables:** reviews, maid_profiles, notifications

**Edge Cases:**
- Customer tries to review twice: Prevent (UNIQUE constraint on booking_id)
- Review submitted long after completion: Allow (no time limit)
- Offensive review content: Admin can hide (is_visible=false), not delete

**API Endpoints:**
- POST /api/v1/reviews
- GET /api/v1/maids/{id}/reviews

---

## WORKFLOW J: Dual-Role Account Management

### J1: Role Switching
**Actors:** User with both customer and maid roles

**Database State:**
```
users: id=xyz789, phone=255712345678
user_roles: 
  - (user_id=xyz789, role_type='customer')
  - (user_id=xyz789, role_type='maid')
maid_profiles: user_id=xyz789, verification_status='approved'
```

**App Behavior:**
1. User opens app
2. App checks user_roles table for this user
3. If multiple roles found, show role switcher UI
4. User selects "Customer Mode":
   - App shows: Browse maids, My bookings (as customer), Payment history
   - Queries filter: `WHERE customer_id = user_id`
5. User switches to "Maid Mode":
   - App shows: Incoming requests, My earnings, Calendar, Wallet
   - Queries filter: `WHERE maid_id = user_id`

**Constraints:**
- User cannot book themselves: `WHERE maid_id != customer_id`
- User can have simultaneous bookings in both roles (as customer at one place, as maid at another)
- Wallet only visible in Maid mode
- Booking history separated by role

**Database Queries:**
```sql
-- Get user's roles
SELECT role_type FROM user_roles WHERE user_id = 'xyz789'

-- Get bookings as customer
SELECT * FROM bookings WHERE customer_id = 'xyz789'

-- Get bookings as maid
SELECT * FROM bookings WHERE maid_id = 'xyz789'

-- Prevent self-booking
INSERT INTO bookings (...) WHERE maid_id != customer_id
```

**Edge Cases:**
- User with only customer role adds maid role later: Add new row to user_roles, redirect to verification
- User verified as maid, then adds customer role: Instant (no verification needed for customer)
- User tries to book self: Validation error, "You cannot book yourself"

---

## WORKFLOW K: Automated Jobs & Background Tasks

### K1: Auto-Cancel Expired Payments
**Trigger:** Cron job runs every hour

**Logic:**
```sql
SELECT * FROM bookings 
WHERE payment_status = 'payment_initiated' 
  AND payment_initiated_at < NOW() - INTERVAL '24 hours'
  AND booking_status NOT IN ('cancelled_by_customer', 'cancelled_by_maid', 'completed')
```

**Actions:**
1. For each expired booking:
   - Update: booking_status='cancelled_by_system', cancellation_reason='Payment timeout'
   - Update: payment_status='expired'
   - Notify customer: "Your booking expired. Please book again."
   - Notify maid: "Booking cancelled due to payment timeout."

### K2: Auto-Confirm Completed Jobs
**Trigger:** Cron job runs every hour

**Logic:**
```sql
SELECT * FROM bookings 
WHERE booking_status = 'in_progress' 
  AND service_completed_at IS NOT NULL
  AND service_completed_at < NOW() - INTERVAL '24 hours'
```

**Actions:**
1. For each booking:
   - Update: booking_status='completed'
   - Trigger payment release (Workflow F)
   - Notify customer: "Auto-confirmed after 24 hours. Please review Mary."
   - Notify maid: "Customer auto-confirmed. You've been paid."

### K3: Daily Statistics Snapshot
**Trigger:** Cron job runs at midnight

**Actions:**
1. Calculate daily metrics:
   - Total bookings created
   - Completed bookings
   - Cancelled bookings
   - Total platform revenue (sum of commission)
   - Total disbursed to maids
   - New customers registered
   - New maids registered
   - Active maids (had at least 1 booking)
2. Insert into `system_stats_daily` table
3. Use for analytics dashboard

---

## CROSS-CUTTING CONCERNS

### Authentication & Authorization
- All API endpoints (except auth, webhooks) require JWT token
- Token includes: userID, roles (customer/maid)
- Role-based access: Maid endpoints check token has 'maid' role
- Token expiry: 24 hours, refresh tokens for longer sessions (optional)

### Notifications
- SMS: For OTP, payment confirmations, important alerts
- Push: For booking requests, messages, reminders
- In-app: For all notifications, stored in database
- Email: Future enhancement (not MVP)

### File Uploads
- Max file size: 5MB for photos, 50MB for videos
- Allowed formats: JPG, PNG for photos; MP4 for videos
- Validation: Check magic bytes (not just extension)
- Storage: MinIO with organized bucket structure
- Access: Private URLs (presigned) for verification docs, public for profile photos

### Rate Limiting
- OTP generation: Max 3 per 10 minutes per phone
- API calls: Max 100 req/min per IP (configurable)
- Payment initiation: Max 3 per hour per user (prevent abuse)

### Logging & Monitoring
- Log all payment transactions (request, response, status changes)
- Log all admin actions (verification, dispute resolution)
- Error tracking: Log all API errors with stack traces
- Webhook logs: Store raw webhook payloads for debugging

### Data Privacy
- Phone numbers: Masked in logs (show 255712***678)
- ID photos: Only admins can view (presigned URLs)
- Customer addresses: Visible only to assigned maid
- Payment details: PCI compliance (don't log card numbers, though we use mobile money)

---
