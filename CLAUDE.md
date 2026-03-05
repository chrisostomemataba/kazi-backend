# KAZI - Home Services Marketplace

## Project Overview
KAZI is a mobile-first platform connecting customers with verified home service workers (maids) in Tanzania. The app handles booking, payments via AzamPay (M-Pesa, Tigo Pesa, Airtel Money, Halopesa), escrow management, and dual-role user support (one phone number can be both customer and worker).

## Core Business Logic
- **Phone-only authentication**: No emails, SMS OTP verification
- **Dual roles**: Single user can be customer AND maid simultaneously
- **Payment flow**: Customer pays → AzamPay holds in escrow → Service completes → Customer confirms → Maid receives payout (minus 15% commission)
- **Currency**: All amounts in TZS (Tanzanian Shillings), stored as integers (no decimals)
- **Verification**: Manual admin review of maid selfie videos (15sec) + ID photos
- **MinIO storage**: All documents, photos, videos stored in MinIO buckets

## Tech Stack
**Backend:**
- Go with Fiber framework (lightweight, fast)
- PostgreSQL database
- MinIO for object storage
- AzamPay API integration
- SMS gateway integration (for OTP)

**Frontend:**
- Flutter (cross-platform iOS/Android)
- Rich UI libraries matching iOS design aesthetic
- State management: Provider or Riverpod
- HTTP client for API calls

## Key Terminology
- **Msaidizi/Maid**: The worker providing services
- **Customer**: Person hiring services
- **Booking**: A scheduled service appointment
- **Escrow**: Platform wallet holding customer payment until service completion
- **Disbursement**: Payout to maid after job completion
- **Verification**: Manual admin approval process for new maids

## Database Principles
- Use UUIDs for all primary keys (not auto-increment integers)
- All timestamps use PostgreSQL TIMESTAMP type
- Phone numbers stored as VARCHAR(13) in format: 255712345678 (no spaces, dashes)
- Money stored as INTEGER (amount in TZS cents/smallest unit)
- Use ENUM types for status fields where applicable
- Proper foreign key constraints with ON DELETE CASCADE where appropriate

## Payment Flow States
**Booking States:**
pending_maid → maid_accepted → payment_pending → confirmed → in_progress → completed

**Payment States:**
unpaid → payment_initiated → paid_held_escrow → released_to_maid

**Escrow States:**
holding → released_to_maid OR refunded_to_customer

## File Storage Structure (MinIO)
```
taskmaid-tz/
├── profiles/original/{user_id}.jpg
├── profiles/thumbnails/{user_id}_thumb.jpg
├── verification/videos/{maid_id}_{timestamp}.mp4
├── verification/ids/{maid_id}_{timestamp}.jpg
└── disputes/{booking_id}/evidence_{n}.jpg
```

## API Design Principles
- RESTful endpoints
- JSON request/response bodies
- Bearer token authentication (JWT)
- Consistent error response format
- Phone number in format 255XXXXXXXXX (no + prefix in API)
- All datetime in ISO 8601 format
- Pagination for lists (limit, offset query params)

## Business Rules
- Minimum withdrawal: TZS 10,000
- Platform commission: 15% of booking total
- OTP expiry: 10 minutes
- Payment pending timeout: 24 hours (auto-cancel if not paid)
- Auto-complete timeout: 24 hours after maid marks complete (if customer doesn't confirm)
- Verification review SLA: 24-48 hours

## Localization
- Primary language: Swahili
- All user-facing text should support Swahili translations
- Store translations in JSON files, not hardcoded
- Date/time formats: DD/MM/YYYY, 12-hour format with AM/PM

## Security Considerations
- Rate limiting on OTP generation (max 3 per 10 minutes per phone)
- Hash sensitive data at rest
- Use prepared statements (SQL injection prevention)
- Validate all file uploads (type, size, content)
- Implement request signing for webhook callbacks
- Store AzamPay credentials in environment variables only

## Performance Guidelines
- Database queries should use indexes
- Implement connection pooling (Postgres, MinIO)
- Cache frequently accessed data (maid listings, platform settings)
- Optimize images before storage (thumbnails for profiles)
- Use background jobs for non-critical tasks (notifications, analytics)

## Testing Approach
- Unit tests for business logic functions
- Integration tests for API endpoints
- Mock external services (AzamPay, SMS gateway)
- Test database migrations rollback capability
- E2E tests for critical flows (booking, payment, withdrawal)

## Deployment Context
- Low-cost VPS hosting (DigitalOcean, Linode)
- Single server initially (backend + postgres + minio)
- Frontend deployed as mobile apps (App Store, Play Store)
- No docker initially (direct systemd service)
- Backup strategy: Daily postgres dumps, weekly MinIO snapshots
