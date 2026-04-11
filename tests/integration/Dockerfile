FROM golang:1.22-alpine

WORKDIR /tests

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENTRYPOINT ["go", "test", "-v", "-count=1", "-timeout", "120s", "./..."]
