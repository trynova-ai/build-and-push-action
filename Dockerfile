# Use the official Golang image as a build stage
FROM golang:1.22-bullseye AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o docker-push-action

# Use a smaller base image for the final stage
FROM alpine:3.16

# Install necessary packages
RUN apk --no-cache add \
    curl \
    bash \
    docker-cli

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/docker-push-action /app/docker-push-action

# Ensure the binary is executable
RUN chmod +x /app/docker-push-action

# Set the command to run the binary
ENTRYPOINT ["/app/docker-push-action"]
