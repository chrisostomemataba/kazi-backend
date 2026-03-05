# KAZI Development Skills & Standards

## Code Style Philosophy
**Principle:** Write code that explains itself through clear naming and structure, not comments.

### General Rules
- **Clarity over cleverness**: Choose readable code over "clever" optimizations
- **Single Responsibility**: Each function/method does one thing well
- **Fail fast**: Validate inputs early, return errors immediately
- **No magic numbers**: Use named constants (e.g., `MinWithdrawalAmount` not `10000`)
- **Comments only for "why"**: Explain business logic reasoning, not "what" the code does
- **Consistent naming**: Use project glossary terms (maid not worker, booking not reservation)

## Naming Conventions

### Go Backend
**Variables:**
- camelCase for local variables: `userID`, `bookingDate`, `totalAmount`
- PascalCase for exported types/functions: `CreateBooking`, `PaymentStatus`
- Acronyms uppercase when appropriate: `userID` not `userId`, `apiKey` not `apikey`

**Functions:**
- Verb-first for actions: `CreateBooking`, `ValidateOTP`, `ReleaseEscrow`
- Noun-first for getters: `UserByPhone`, `BookingByID`
- Boolean returns prefix with `Is`, `Has`, `Can`: `IsVerified`, `HasPendingPayment`

**Files:**
- Lowercase with underscores: `booking_service.go`, `payment_handler.go`
- Test files: `booking_service_test.go`
- Group by domain not layer: `booking/service.go`, `booking/repository.go`, `booking/handler.go`

**Constants:**
- PascalCase for exported: `MaxOTPAttempts`, `PlatformCommissionRate`
- camelCase for package-private: `defaultTimeout`, `minPasswordLength`

### Flutter Frontend
**Variables:**
- camelCase: `userName`, `bookingId`, `totalAmount`
- Prefix booleans: `isLoading`, `hasError`, `canSubmit`
- Prefix private with underscore: `_controller`, `_fetchData`

**Classes:**
- PascalCase: `BookingCard`, `PaymentService`, `MaidProfileScreen`
- Suffix widgets with purpose: `BookingListScreen`, `PaymentButton`, `LoadingIndicator`

**Files:**
- snake_case: `booking_card.dart`, `payment_service.dart`, `maid_profile_screen.dart`
- One major widget/class per file
- Group by feature: `features/booking/`, `features/payment/`

## File Organization Rules

### Backend (Go)
**Keep files focused:**
- Max 300 lines per file (guideline not hard rule)
- If file grows beyond 400 lines, consider splitting by responsibility
- Group related functions together, separate with blank line + comment header

**Module boundaries:**
- Each domain (booking, payment, user, maid) is a package
- Shared utilities in `internal/util`
- Database models in each domain package (not centralized models package)
- API handlers in each domain package

**When to split:**
- ✅ Split when file handles multiple sub-domains (e.g., `booking_service.go` into `booking_create.go`, `booking_update.go`)
- ✅ Split when file has multiple large functions (300+ lines total)
- ❌ Don't split just to have more files (3 files with 20 lines each is worse than 1 file with 60 lines)
- ❌ Don't fragment so much that you're jumping between 10 files to understand one flow

### Frontend (Flutter)
**Screen structure:**
```
features/booking/
├── screens/
│   ├── booking_list_screen.dart (main screen)
│   └── booking_detail_screen.dart
├── widgets/
│   ├── booking_card.dart (reusable card)
│   └── booking_status_badge.dart
├── models/
│   └── booking.dart (data model)
├── services/
│   └── booking_service.dart (API calls)
└── providers/
└── booking_provider.dart (state management)
```

**Widget split rules:**
- ✅ Extract widget if used in 2+ places
- ✅ Extract if widget has complex logic (>50 lines)
- ✅ Extract if it improves screen readability
- ❌ Don't extract every tiny widget (e.g., a single Text with style)

## Comment Guidelines

### When to Comment
**DO comment:**
- Complex business rules: `// AzamPay requires amount in TZS (no decimals), so we multiply by 100`
- Non-obvious decisions: `// We use pessimistic locking here to prevent double-withdrawal`
- External API quirks: `// AzamPay webhook can arrive before API response, handle both cases`
- Temporary workarounds: `// TODO: Remove after AzamPay adds refund API (Q2 2026)`
- Public API functions: `// CreateBooking initiates a new service booking and triggers payment flow`

**DON'T comment:**
- Obvious code: `// Set user name` (we can see that)
- Variable declarations: `userID := 123 // user ID` (redundant)
- Every function: Only comment exported functions or complex internal ones
- What code does: Comment *why* it does it, not what

### Comment Format
**Go:**
```go
// CreateBooking initiates a booking and sends notification to maid.
// Returns error if maid is unavailable or customer has insufficient funds.
func CreateBooking(ctx context.Context, req BookingRequest) (*Booking, error) {
// Business logic here...
}
```

**Flutter:**
```dart
/// Creates a booking and navigates to payment screen.
///
/// Throws [BookingException] if maid is unavailable.
Future<void> createBooking(BookingRequest request) async {
// Implementation...
}
```

## Error Handling Standards

### Go Backend
**Error types:**
```go
// Define domain errors as variables
var (
ErrUserNotFound      = errors.New("user not found")
ErrInsufficientFunds = errors.New("insufficient funds")
ErrInvalidOTP        = errors.New("invalid OTP code")
)// Wrap errors with context
return fmt.Errorf("create booking: %w", err)
```

**HTTP errors:**
```go
// Return consistent JSON error format
{
"error": "invalid_request",
"message": "Phone number is required",
"code": 400
}
```

### Flutter Frontend
**Try-catch blocks:**
```dart
// Minimal try-catch, handle at service layer
try {
await bookingService.create(booking);
} on BookingException catch (e) {
showError(e.message);
} catch (e) {
showError('Something went wrong');
}
```

## Testing Principles

### Backend Tests
**What to test:**
- Business logic functions (booking creation, payment calculation)
- Database queries (ensure correct SQL, handle edge cases)
- API endpoints (request validation, response format)

**What NOT to test:**
- Trivial getters/setters
- Third-party libraries (trust Go stdlib, Fiber, etc.)
- Database driver itself

**Test naming:**
```go
func TestCreateBooking_ValidInput_Success(t *testing.T)
func TestCreateBooking_InvalidDate_ReturnsError(t *testing.T)
func TestCalculateCommission_ZeroAmount_ReturnsZero(t *testing.T)
```

### Frontend Tests
**Widget tests:**
- Test user interactions (button taps, form submissions)
- Test conditional rendering (loading states, error states)
- Test navigation flows

**Test naming:**
```dart
testWidgets('BookingCard displays maid name and price', (tester) async { ... })
testWidgets('Payment button disabled when form invalid', (tester) async { ... })
```

## API Response Formats

### Success Response
```json{
"success": true,
"data": { ... },
"message": "Booking created successfully"
}
```

### Error Response
```json{
"success": false,
"error": "validation_error",
"message": "Phone number is required",
"details": {
"field": "phone_number",
"code": "required"
}
}
```

### Paginated Response
```json{
"success": true,
"data": [ ... ],
"pagination": {
"total": 150,
"limit": 20,
"offset": 40,
"has_more": true
}
}
```

## Database Query Patterns

### Use prepared statements
```go
// Good
db.Query("SELECT * FROM users WHERE phone_number = $1", phone)// Bad
db.Query(fmt.Sprintf("SELECT * FROM users WHERE phone_number = '%s'", phone))
```

### Use transactions for multi-step operations
```go
// Payment release involves multiple tables
tx, _ := db.Begin()
defer tx.Rollback() // Auto-rollback if commit not called// Update escrow
tx.Exec("UPDATE platform_escrow_wallet SET status = 'released' WHERE booking_id = $1", bookingID)// Credit maid wallet
tx.Exec("UPDATE maid_wallets SET available_balance = available_balance + $1 WHERE maid_id = $2", amount, maidID)// Log transaction
tx.Exec("INSERT INTO wallet_transactions (...) VALUES (...)")tx.Commit() // All or nothing
```

### Use indexes wisely
```sql
-- Index phone lookups (used in every login)
CREATE INDEX idx_users_phone ON users(phone_number);-- Compound index for common query
CREATE INDEX idx_bookings_customer_date ON bookings(customer_id, booking_date DESC);-- Don't over-index (every index slows writes)
```

## Configuration Management

### Environment Variables (Required)
```bash
## Database
DATABASE_URL=postgres://user:pass@localhost:5432/kaziMinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123AzamPay
AZAMPAY_APP_NAME=KAZI
AZAMPAY_CLIENT_ID=xxx
AZAMPAY_CLIENT_SECRET=xxx
AZAMPAY_BASE_URL=https://sandbox.azampay.co.tz/api/v1SMS Gateway
SMS_API_KEY=xxx
SMS_SENDER_ID=KAZIJWT
JWT_SECRET=random-secret-key-min-32-charsApp
PORT=8080
ENV=development
```

### Config loading (Go)
```go
// Use struct for config, validate on startup
type Config struct {
DatabaseURL    string env:"DATABASE_URL,required"
MinIOEndpoint  string env:"MINIO_ENDPOINT,required"
AzamPayBaseURL string env:"AZAMPAY_BASE_URL,required"
Port           int    env:"PORT" envDefault:"8080"
}// Load and validate early in main()
cfg := LoadConfig()
if err := cfg.Validate(); err != nil {
log.Fatal(err)
}
```

## Performance Optimization

### Backend
- **Connection pooling**: Set max connections for Postgres (20-50 for small server)
- **Query optimization**: Use EXPLAIN ANALYZE to check slow queries
- **Caching**: Cache platform settings, maid listings (Redis later, in-memory initially)
- **Batch operations**: Insert multiple notifications in single query
- **Background jobs**: Send emails/SMS async (don't block HTTP response)

### Frontend
- **Image optimization**: Resize images before upload, use thumbnails in lists
- **Pagination**: Load 20 items at a time, infinite scroll
- **Debouncing**: Debounce search input (wait 300ms after typing stops)
- **Lazy loading**: Load tabs/screens on demand, not upfront
- **State management**: Don't rebuild entire tree, use targeted updates

## Security Checklist

### Backend
- [ ] SQL injection prevention (use parameterized queries)
- [ ] XSS prevention (sanitize user input)
- [ ] Rate limiting (max 100 req/min per IP on auth endpoints)
- [ ] JWT expiry (24 hours, refresh token for longer sessions)
- [ ] File upload validation (check magic bytes, not just extension)
- [ ] Webhook signature verification (HMAC for AzamPay callbacks)
- [ ] HTTPS only in production
- [ ] CORS properly configured (whitelist mobile app domains)

### Frontend
- [ ] Secure storage for tokens (flutter_secure_storage)
- [ ] No sensitive data in logs
- [ ] Certificate pinning for API calls (production)
- [ ] Biometric auth for sensitive actions (withdrawals)
- [ ] Input validation on client side (UX, not security)

## Deployment Checklist

### Backend
- [ ] Environment variables set
- [ ] Database migrations applied
- [ ] MinIO buckets created with proper policies
- [ ] Systemd service configured with auto-restart
- [ ] Log rotation enabled
- [ ] Postgres backups scheduled (daily)
- [ ] SSL certificate configured (Let's Encrypt)
- [ ] Monitoring setup (Uptime checks, error alerts)

### Frontend
- [ ] API endpoints point to production
- [ ] App signing keys configured
- [ ] Privacy policy and terms links updated
- [ ] App store assets prepared (screenshots, descriptions)
- [ ] Push notification certificates configured
- [ ] Analytics initialized (if using)