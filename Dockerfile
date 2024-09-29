# Use the official Golang image as the base image
FROM golang:1.21-alpine AS builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
COPY . .

RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /quote-bot

# Install CA certificates
RUN apk --no-cache add ca-certificates

# Use a minimal base image to run the Go app
FROM scratch

# Copy the CA certificates from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from the build stage
COPY --from=builder /quote-bot /quote-bot

# Copy the migrations directory from the build stage
COPY --from=builder /app/migrations /migrations

# Command to run the executable
CMD ["/quote-bot"]