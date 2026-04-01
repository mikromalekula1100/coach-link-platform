FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy the shared events package (for the replace directive)
COPY pkg/events /src/pkg/events

# Copy service source
COPY services/notification-service/ /src/services/notification-service/

WORKDIR /src/services/notification-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/notification-service /usr/local/bin/notification-service

EXPOSE 8004

ENTRYPOINT ["notification-service"]
