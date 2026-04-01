FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy service source
COPY services/api-gateway/ /src/services/api-gateway/

WORKDIR /src/services/api-gateway

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api-gateway ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api-gateway /usr/local/bin/api-gateway

EXPOSE 8080

ENTRYPOINT ["api-gateway"]
