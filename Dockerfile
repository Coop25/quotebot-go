# Use the official Golang image as the base image
FROM golang:1.20-alpine AS builder
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
# Run
CMD ["/quote-bot"]