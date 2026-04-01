FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared events package first (needed for replace directive)
COPY pkg/events/ /build/pkg/events/

# Copy service source
COPY services/auth-service/ /build/services/auth-service/

WORKDIR /build/services/auth-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /auth-service /auth-service

EXPOSE 8001

ENTRYPOINT ["/auth-service"]
