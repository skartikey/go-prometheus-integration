# Use the official Golang image with the required version for building the application
FROM golang:1.23.4 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./

# Enable Go modules and download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application binary
RUN go build -o app .

# Use a smaller, more secure base image for running the application
FROM debian:bullseye-slim

# Set working directory inside the runtime container
WORKDIR /app

# Copy the application binary from the builder image
COPY --from=builder /app/app .

# Expose port 9000 for the application
EXPOSE 9000

# Run the application binary
CMD ["./app"]
