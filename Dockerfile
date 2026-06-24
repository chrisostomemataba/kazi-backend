# --- Step 1: Build the Go binary ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile a completely static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /kazi-server ./cmd/server

# --- Step 2: Final Secure Distroless Runtime ---
# gcr.io/distroless/static-debian12 contains system certificates, 
# tzdata for timezones, and a /tmp directory, but NO shell, package manager, or extra fluff.
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy the compiled static binary from the builder stage
COPY --from=builder /kazi-server /app/kazi-server

# Expose Go Fiber's port
EXPOSE 8080

# Directly invoke the binary without a shell wrapper
ENTRYPOINT ["/app/kazi-server"]