FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy service source
COPY services/ai-service/ /src/services/ai-service/

WORKDIR /src/services/ai-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ai-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/ai-service /usr/local/bin/ai-service

EXPOSE 8006

ENTRYPOINT ["ai-service"]
