# Multi-stage build for Ard programming language
FROM golang:1.25rc1-alpine AS builder

# Install git and build dependencies (including CGO requirements)
RUN apk add --no-cache git gcc musl-dev

# Clone and build the Ard language using GitHub token
WORKDIR /build
ARG GITHUB_TOKEN
RUN git clone https://${GITHUB_TOKEN}@github.com/akonwi/ard.git
WORKDIR /build/ard
RUN go mod download
ENV CGO_ENABLED=1
RUN go build --tags=goexperiment.jsonv2 -o /usr/local/bin/ard

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy the built Ard binary
COPY --from=builder /usr/local/bin/ard /usr/local/bin/ard

# Create app directory
WORKDIR /app

# Copy application files
COPY . .

# Set Zeabur required environment variables
ENV PORT=8080
ENV HOST=0.0.0.0

# Expose port
EXPOSE 8080

# Run the application
CMD ["ard", "run", "main.ard"]