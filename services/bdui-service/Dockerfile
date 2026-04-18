FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy service source
COPY services/bdui-service/ /src/services/bdui-service/

WORKDIR /src/services/bdui-service

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bdui-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/bdui-service /usr/local/bin/bdui-service

EXPOSE 8007

ENTRYPOINT ["bdui-service"]
