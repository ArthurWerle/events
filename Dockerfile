# Build stage
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Cache bust arg - change this value to force rebuild
ARG CACHE_BUST=1

# Copy source code
COPY . .

# Build the application (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary and the directories the service reads at runtime.
# The service resolves ./db/migrations (file://db/migrations) and
# ./templates/index.html relative to the working directory, so both must
# sit next to the binary under WORKDIR.
COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/templates ./templates

RUN chmod +x ./server

# Expose the port the service listens on (hardcoded :3000 in cmd/server)
EXPOSE 3000

CMD ["./server"]
