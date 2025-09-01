FROM golang:1.21-alpine AS builder
WORKDIR /src

# cache deps
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/migrator ./cmd/migrator

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/migrator /usr/local/bin/migrator
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/migrator"]
