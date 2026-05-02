FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy the shared packages (for the replace directives)
COPY pkg/events /src/pkg/events
COPY pkg/httpmetrics /src/pkg/httpmetrics

# Copy service source
COPY services/user-service/ /src/services/user-service/

WORKDIR /src/services/user-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/user-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/user-service /usr/local/bin/user-service

EXPOSE 8002

ENTRYPOINT ["user-service"]
