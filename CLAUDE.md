# KAZI Backend — Live Tracking Context

## What this task is

Add live maid location tracking to the existing booking flow. The maid sends GPS coordinates to the backend while en route. The customer polls those coordinates to display a live map. Nothing else changes.

## Scope boundary

Touch only:
- one new migration file
- `internal/booking/handler.go`
- `internal/booking/service.go`
- `internal/booking/repository.go`
- `internal/routes/routes.go`

Do not touch payment, auth, notification, or any other domain.

## Existing booking state machine

```
pending_maid → maid_accepted → payment_pending → confirmed → in_progress → completed
```

Tracking is only active when `booking_status = 'confirmed'` (maid accepted, payment done, maid en route) or `booking_status = 'in_progress'` (maid has arrived and started work).

The maid taps "Ninaelekea" (I'm on my way) — this is when location sharing starts. The maid taps "Nimefikisha" (I've arrived) — this is Workflow E Step 1, which transitions to `in_progress`. Location sharing stops after that transition.

## Domain module pattern

Each domain owns its own `setup.go` with `repo → service → handler` wiring. `main.go` does infrastructure only. Routes live in `internal/routes/routes.go`.

When you add a function to service or repository, follow the existing constructor pattern — do not add package-level variables or init functions.

## Authorization rules

- `POST /api/v1/bookings/:id/location` — JWT required, caller must be the maid assigned to that booking
- `GET /api/v1/bookings/:id/location` — JWT required, caller must be the customer of that booking

Check both conditions in the service layer, not just the handler.

## ETA calculation

Use the haversine formula. Walking speed constant: 4.0 km/h. Return `distance_km` (float, 2 decimal places) and `eta_minutes` (int). No external routing API.

The customer's coordinates come from `bookings.customer_lat` and `bookings.customer_lng` which already exist on the booking record.

## Database column types

```
maid_current_lat          DECIMAL(10,8)   NULLABLE
maid_current_lng          DECIMAL(11,8)   NULLABLE
maid_location_updated_at  TIMESTAMP       NULLABLE
```

Add these to the `bookings` table via a migration. Do not create a separate table.

## Docker

The project runs via Docker Compose. After adding the migration file, the migration must be applied. Run:

```
docker compose exec app go run ./cmd/migrate/main.go up
```

or whatever the existing migration runner command is in the project. Check `Makefile` or `docker-compose.yml` for the exact command before running.

## Response format

All responses follow the existing pattern in `internal/common/util/response.go`. Use whatever `SuccessResponse` and `ErrorResponse` helpers already exist — do not invent new ones.

## No new dependencies

Do not add any Go packages. `math` is part of the standard library and covers haversine.