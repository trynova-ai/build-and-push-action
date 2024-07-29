# Use the official Golang image as a build stage
FROM golang:1.22-bullseye AS base

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o docker-push-action

# Use the Docker official Docker image as the final stage
FROM docker:27.1.1-dind

# Install necessary packages
RUN apk --no-cache add \
    curl \
    bash

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=base /app/docker-push-action .

# Copy any other necessary files
COPY . .

# Run the binary
ENTRYPOINT ["/app/docker-push-action"]
