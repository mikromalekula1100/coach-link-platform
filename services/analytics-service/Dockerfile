FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy service source
COPY services/analytics-service/ /src/services/analytics-service/

WORKDIR /src/services/analytics-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/analytics-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/analytics-service /usr/local/bin/analytics-service

EXPOSE 8005

ENTRYPOINT ["analytics-service"]
