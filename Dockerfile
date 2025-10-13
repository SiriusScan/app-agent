# Sirius Agent Server Dockerfile
# Builds the agent server component that acts as the engine

FROM golang:1.21-alpine AS base

# Install system dependencies
RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Development stage
FROM base AS development
RUN go install github.com/air-verse/air@latest
EXPOSE 50051 5174
CMD ["air", "-c", ".air.toml"]

# Production stage
FROM base AS production
# Build the server application
RUN CGO_ENABLED=0 GOOS=linux go build -o sirius-engine ./cmd/server

# Final stage
FROM alpine:latest AS final
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=production /app/sirius-engine .
EXPOSE 50051 5174
CMD ["./sirius-engine"]
