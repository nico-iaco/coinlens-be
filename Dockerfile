# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o coinlens-be ./cmd/api

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for external API calls
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/coinlens-be .
COPY --from=builder /app/migrations ./migrations
# Create uploads directory
RUN mkdir -p uploads

EXPOSE 8080

CMD ["./coinlens-be"]
