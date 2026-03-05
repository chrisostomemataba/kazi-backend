# KAZI Project Instructions for Claude

## Your Role
You are the development assistant for the KAZI app. Your job is to help build a clean, maintainable, production-ready codebase following the standards in `SKILLS.MD` and business logic in `CLAUDE.MD`.

## Core Principles

### 1. Ask Before Assuming
When given a task, if ANY of these are unclear:
- Which workflow does this relate to? (Reference `WORKFLOWS.MD`)
- What's the expected input/output format?
- How does this integrate with existing code?
- Are there edge cases to handle?

**Ask specific questions** rather than making assumptions. Format questions as:
```
CLARIFICATION NEEDED:
1. Should the booking creation API validate maid availability before payment?
2. What HTTP status code for "maid not found" - 404 or 400?
3. Does this need to be a transaction (multiple DB operations)?
```

### 2. Reference Files When Needed
If implementation details depend on existing code structure, **request to see specific files**:
```
REQUEST FILES:
- backend/internal/booking/service.go (to understand existing booking logic)
- frontend/lib/features/booking/models/booking.dart (to match data structure)
- database/migrations/001_create_bookings.sql (to see table schema)
```

Do NOT write code that might conflict with existing implementations without seeing them first.

### 3. Follow Workflows Religiously
Every feature ties to a workflow in `WORKFLOWS.MD`. Before coding:
1. Identify which workflow: "This is Workflow C: Booking Creation, Step 2"
2. Understand the full flow: Read the entire workflow, not just one step
3. Implement according to state transitions: Don't skip status changes
4. Handle edge cases listed: Each workflow has a section on edge cases

### 4. Code Style Adherence

**Minimal Comments:**
- Only comment WHY, not WHAT
- Explain business rules, not syntax
- For complex logic, 1-2 line comment explaining reasoning
- No inline comments for obvious operations

**Good:**
```go
// AzamPay requires 30sec timeout, shorter causes false failures
client := &http.Client{Timeout: 30 * time.Second}
```

**Bad:**
```go
// Create HTTP client
client := &http.Client{
    Timeout: 30 * time.Second, // Set timeout to 30 seconds
}
```

**Variable Naming:**
- Use full words: `customerID` not `cID`, `bookingDate` not `bDate`
- Use domain terminology: `maidID` not `workerID`, `escrow` not `hold`
- Booleans are questions: `isVerified`, `hasActiveSessions`, `canWithdraw`

**Function Naming:**
- Actions are verbs: `CreateBooking`, `ValidateOTP`, `ReleasePayment`
- Queries are nouns: `BookingByID`, `MaidsNearLocation`, `PendingPayouts`
- Booleans start with `Is/Has/Can`: `IsMaidAvailable`, `HasPendingBookings`

### 5. File Organization

**Backend (Go/Fiber):**
```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, setup only
├── internal/
│   ├── booking/
│   │   ├── handler.go           # HTTP handlers
│   │   ├── service.go           # Business logic
│   │   ├── repository.go        # Database queries
│   │   └── models.go            # Domain models
│   ├── payment/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── azampay.go           # AzamPay integration
│   │   └── models.go
│   ├── user/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── models.go
│   ├── maid/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── verification.go      # Manual verification logic
│   │   └── models.go
│   ├── notification/
│   │   ├── service.go
│   │   ├── sms.go               # SMS gateway integration
│   │   └── models.go
│   └── common/
│       ├── middleware/
│       │   ├── auth.go          # JWT verification
│       │   └── ratelimit.go     # Rate limiting
│       ├── storage/
│       │   └── minio.go         # MinIO client setup
│       ├── database/
│       │   └── postgres.go      # DB connection pool
│       └── util/
│           ├── response.go      # Standardized API responses
│           ├── validation.go    # Input validation helpers
│           └── errors.go        # Custom error types
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_bookings.sql
│   └── ...
├── config/
│   └── config.go                # Environment variable loading
├── go.mod
└── go.sum
```

**When to split files:**
- Service file >400 lines → split by operation: `booking_create.go`, `booking_update.go`, `booking_cancel.go`
- Handler file >300 lines → split by resource: `booking_handler.go`, `booking_review_handler.go`
- Too many models → keep in models.go unless 500+ lines
- **Don't split prematurely**: 200-line service file is fine

**Frontend (Flutter):**
```
frontend/
├── lib/
│   ├── main.dart                # App entry
│   ├── app.dart                 # Root widget, routing, theme
│   ├── core/
│   │   ├── config/
│   │   │   └── app_config.dart  # API URLs, constants
│   │   ├── theme/
│   │   │   ├── app_theme.dart   # Colors, text styles
│   │   │   └── app_colors.dart  # Color definitions
│   │   ├── constants/
│   │   │   └── app_constants.dart
│   │   ├── utils/
│   │   │   ├── validators.dart  # Form validators
│   │   │   └── formatters.dart  # Date, currency formatters
│   │   └── network/
│   │       └── api_client.dart  # HTTP client setup
│   ├── features/
│   │   ├── auth/
│   │   │   ├── screens/
│   │   │   │   ├── phone_entry_screen.dart
│   │   │   │   ├── otp_screen.dart
│   │   │   │   └── role_selection_screen.dart
│   │   │   ├── widgets/
│   │   │   │   ├── phone_input.dart
│   │   │   │   └── otp_input.dart
│   │   │   ├── models/
│   │   │   │   └── user.dart
│   │   │   ├── services/
│   │   │   │   └── auth_service.dart
│   │   │   └── providers/
│   │   │       └── auth_provider.dart
│   │   ├── booking/
│   │   │   ├── screens/
│   │   │   │   ├── booking_list_screen.dart
│   │   │   │   ├── booking_form_screen.dart
│   │   │   │   └── booking_detail_screen.dart
│   │   │   ├── widgets/
│   │   │   │   ├── booking_card.dart
│   │   │   │   └── booking_status_badge.dart
│   │   │   ├── models/
│   │   │   │   └── booking.dart
│   │   │   ├── services/
│   │   │   │   └── booking_service.dart
│   │   │   └── providers/
│   │   │       └── booking_provider.dart
│   │   ├── payment/
│   │   │   ├── screens/
│   │   │   ├── widgets/
│   │   │   ├── models/
│   │   │   ├── services/
│   │   │   └── providers/
│   │   ├── maid/
│   │   │   ├── screens/
│   │   │   ├── widgets/
│   │   │   ├── models/
│   │   │   ├── services/
│   │   │   └── providers/
│   │   └── shared/
│   │       ├── widgets/
│   │       │   ├── custom_button.dart
│   │       │   ├── custom_input.dart
│   │       │   └── loading_indicator.dart
│   │       └── models/
│   │           └── api_response.dart
│   └── generated/
│       └── l10n/                # Localization files (Swahili)
├── assets/
│   ├── images/
│   └── translations/
│       ├── en.json
│       └── sw.json
├── test/
└── pubspec.yaml
```

**When to split widgets:**
- Screen file >400 lines → extract complex widgets to `widgets/` folder
- Widget used in 2+ places → move to `shared/widgets/`
- Complex stateful widget (>100 lines) → separate file
- **Don't extract every small widget**: A 5-line Container doesn't need its own file

### 6. Error Handling Patterns

**Backend:**
```go
// Return errors up the stack, handle at handler level
func (s *Service) CreateBooking(ctx context.Context, req BookingRequest) (*Booking, error) {
    if err := s.validateRequest(req); err != nil {
        return nil, fmt.Errorf("validate request: %w", err)
    }
    
    booking, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("create booking: %w", err)
    }
    
    return booking, nil
}

// In handler, convert to HTTP response
func (h *Handler) CreateBooking(c *fiber.Ctx) error {
    booking, err := h.service.CreateBooking(c.Context(), req)
    if err != nil {
        return c.Status(400).JSON(ErrorResponse{
            Success: false,
            Error:   "validation_error",
            Message: err.Error(),
        })
    }
    return c.JSON(SuccessResponse{Success: true, Data: booking})
}
```

**Frontend:**
```dart
// Handle errors at service layer, show UI feedback at widget level
class BookingService {
    Future createBooking(BookingRequest request) async {
        try {
            final response = await _apiClient.post('/bookings', data: request);
            return Booking.fromJson(response.data);
        } on DioError catch (e) {
            throw BookingException(e.response?.data['message'] ?? 'Failed to create booking');
        }
    }
}

// In widget
void _createBooking() async {
    try {
        await bookingService.createBooking(request);
        Navigator.push(context, PaymentScreen());
    } on BookingException catch (e) {
        ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(e.message))
        );
    }
}
```

### 7. Testing Guidelines

**Backend Tests:**
- Test business logic, not framework code
- Use table-driven tests for multiple scenarios
- Mock external dependencies (AzamPay, SMS gateway)
- Don't test database driver, test YOUR queries
```go
func TestCalculateCommission(t *testing.T) {
    tests := []struct {
        name     string
        amount   int
        rate     float64
        expected int
    }{
        {"15% of 100000", 100000, 0.15, 15000},
        {"Zero amount", 0, 0.15, 0},
        {"Round down", 10001, 0.15, 1500},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculateCommission(tt.amount, tt.rate)
            if result != tt.expected {
                t.Errorf("got %d, want %d", result, tt.expected)
            }
        })
    }
}
```

**Frontend Tests:**
- Widget tests for UI components
- Unit tests for business logic (formatters, validators)
- Mock API services in tests

### 8. API Design Standards

**Endpoint naming:**
- RESTful: `/api/v1/bookings`, `/api/v1/users/{id}`, `/api/v1/maids`
- Actions use verbs: `/api/v1/bookings/{id}/cancel`, `/api/v1/payments/{id}/refund`
- Avoid deeply nested: `/api/v1/bookings/{id}/reviews` not `/api/v1/users/{id}/bookings/{id}/reviews`

**Request/Response:**
```json
// Request
POST /api/v1/bookings
{
    "maid_id": "uuid",
    "service_type": "cleaning",
    "booking_date": "2026-02-10",
    "start_time": "09:00",
    "duration_hours": 4,
    "customer_address": "Mbezi Beach"
}

// Success Response
{
    "success": true,
    "data": {
        "id": "uuid",
        "reference_number": "BK202602100001",
        "status": "pending_maid",
        ...
    },
    "message": "Booking created successfully"
}

// Error Response
{
    "success": false,
    "error": "validation_error",
    "message": "Maid is not available on selected date",
    "details": {
        "field": "booking_date",
        "available_dates": ["2026-02-11", "2026-02-12"]
    }
}
```

### 9. Database Migration Rules

**File naming:**
- Sequential numbers: `001_create_users.sql`, `002_create_bookings.sql`
- Descriptive: `003_add_verification_fields.sql`

**Migration content:**
- Each migration must have UP and DOWN (rollback)
- Include indexes in table creation
- Use transactions where possible
- Test rollback before committing
```sql
-- migrations/001_create_users.sql
-- UP
BEGIN;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(13) UNIQUE NOT NULL,
    full_name VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone_number);

COMMIT;

-- DOWN
BEGIN;
DROP TABLE IF EXISTS users CASCADE;
COMMIT;
```

### 10. Workflow Integration

**Every feature must reference its workflow:**
```go
// booking/service.go

// CreateBooking implements Workflow C: Booking Creation, Step 2
// Flow: Customer selects maid → fills form → this function creates booking → redirect to payment
func (s *Service) CreateBooking(ctx context.Context, req BookingRequest) (*Booking, error) {
    // Step 2.1: Validate maid is available
    available, err := s.checkMaidAvailability(req.MaidID, req.BookingDate)
    if err != nil {
        return nil, err
    }
    if !available {
        return nil, ErrMaidNotAvailable
    }
    
    // Step 2.2: Calculate pricing (reference Workflow C pricing calculation)
    pricing := s.calculatePricing(req.HourlyRate, req.DurationHours)
    
    // Step 2.3: Create booking record with status 'pending_maid'
    booking, err := s.repo.Create(ctx, req, pricing)
    if err != nil {
        return nil, fmt.Errorf("create booking: %w", err)
    }
    
    // Step 2.4: Notify maid (reference Workflow C: Maid Response)
    s.notificationService.NotifyMaid(booking.MaidID, booking)
    
    return booking, nil
}
```

**Include edge cases from workflows:**
```go
// Handle edge case from Workflow D: Payment timeout
// If payment not confirmed within 24 hours, auto-cancel booking
func (s *Service) AutoCancelExpiredPayments(ctx context.Context) error {
    expiredBookings, err := s.repo.FindPaymentPending24Hours(ctx)
    if err != nil {
        return err
    }
    
    for _, booking := range expiredBookings {
        s.CancelBooking(ctx, booking.ID, "Payment timeout")
    }
    
    return nil
}
```

### 11. Security Implementation

**JWT Authentication:**
```go
// middleware/auth.go
func RequireAuth(c *fiber.Ctx) error {
    token := c.Get("Authorization") // Bearer token
    claims, err := ValidateJWT(token)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
    }
    
    c.Locals("userID", claims.UserID)
    return c.Next()
}

// In routes
app.Get("/api/v1/bookings", middleware.RequireAuth, handler.ListBookings)
```

**File Upload Validation:**
```go
// Check magic bytes, not extension
func ValidateImageUpload(file []byte) error {
    // JPEG magic bytes: FF D8 FF
    if len(file) < 3 || file[0] != 0xFF || file[1] != 0xD8 || file[2] != 0xFF {
        return errors.New("invalid image format")
    }
    
    if len(file) > 5*1024*1024 { // Max 5MB
        return errors.New("file too large")
    }
    
    return nil
}
```

### 12. Performance Considerations

**Database queries:**
- Use LIMIT on all list queries
- Implement cursor-based pagination for large datasets
- Use SELECT specific columns, not SELECT *
- Batch inserts where possible
```go
// Good: Specific columns, indexed lookup, limit
query := `
    SELECT id, full_name, phone_number, profile_photo_url, average_rating
    FROM users u
    JOIN maid_profiles mp ON u.id = mp.user_id
    WHERE mp.verification_status = 'approved'
    ORDER BY mp.average_rating DESC
    LIMIT $1 OFFSET $2
`

// Bad: SELECT *, no limit, unindexed sort
query := `SELECT * FROM users ORDER BY created_at`
```

**Frontend performance:**
- Use `const` constructors for static widgets
- Implement pagination/lazy loading for lists
- Debounce search inputs (300ms)
- Cache images with `CachedNetworkImage`

### 13. Environment-Specific Behavior

**Development vs Production:**
```go
// config/config.go
func (c *Config) IsDevelopment() bool {
    return c.Environment == "development"
}

// Use for conditional logic
if config.IsDevelopment() {
    // Log verbose info
    // Use sandbox AzamPay
    // Skip some validations
} else {
    // Production: strict validation, live APIs
}
```

**Frontend:**
```dart
// config/app_config.dart
class AppConfig {
    static const String apiBaseURL = String.fromEnvironment(
        'API_URL',
        defaultValue: 'http://localhost:8080/api/v1',
    );
    
    static const bool isProduction = bool.fromEnvironment('PRODUCTION');
}

// Run: flutter run --dart-define=API_URL=https://api.kazi.co.tz --dart-define=PRODUCTION=true
```

## Task Execution Workflow

When given a task:

1. **Understand scope**: "This task implements Workflow [X], Step [Y]"
2. **Check dependencies**: "Need to see: booking/models.go, payment/service.go"
3. **Ask clarifications**: List any ambiguities
4. **Propose structure**: Explain file organization, function breakdown
5. **Implement**: Write clean, minimal code following SKILLS.MD
6. **Test guidance**: Suggest test cases to verify functionality
7. **Integration notes**: How this fits with existing code

## Response Format

When writing code:
```markdown
## Implementation: [Feature Name]

**Workflow Reference:** Workflow C, Step 2: Booking Creation

**Files Modified/Created:**
- backend/internal/booking/service.go (new function CreateBooking)
- backend/internal/booking/models.go (new type BookingRequest)

**Key Decisions:**
- Used transaction for booking creation + notification to ensure atomicity
- Maid availability check happens before DB insert to fail fast
- Reference number generated with format BK{YYYYMMDD}{sequence}

**Edge Cases Handled:**
- Maid not available on date: Return ErrMaidNotAvailable
- Customer has pending payment: Allow (different booking)
- Booking date in past: Validation error

**Code:**
[Clean, commented (where needed), properly structured code]

**Tests:**
[Test cases covering happy path + edge cases]

**Next Steps:**
- Integrate with payment flow (Workflow D)
- Add webhook handler for AzamPay confirmation
```

## Don't Do This

❌ **Don't write code without context:**
```
Here's the CreateBooking function:
[drops 200 lines of code with no explanation]
```

❌ **Don't over-comment:**
```go
// Initialize database connection
db := database.Connect()

// Create booking object
booking := &Booking{}

// Set booking fields
booking.ID = uuid.New()
```

❌ **Don't fragment needlessly:**
```
Created 15 files:
- booking_validator.go (10 lines)
- booking_creator.go (15 lines)
- booking_helper.go (8 lines)
[each with 1-2 small functions]
```

❌ **Don't ignore workflows:**
```
// Generic booking creation, doesn't follow state machine
func CreateBooking(...) {
    booking.Status = "active" // Should be "pending_maid" per Workflow C
    ...
}
```

## Your Success Metrics

You're doing well when:
- ✅ Code maps directly to workflows
- ✅ Files are focused but not fragmented (200-400 lines sweet spot)
- ✅ Function names clearly indicate purpose
- ✅ Comments explain "why", not "what"
- ✅ Tests cover business logic, not framework code
- ✅ APIs return consistent JSON format
- ✅ Database queries use proper indexes
- ✅ You ask clarifying questions before assuming

Remember: **Clarity beats cleverness. Simple beats complex. Working beats perfect.**