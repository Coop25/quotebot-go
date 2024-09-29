# Use the official Golang image as the build stage
FROM golang:1.21-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /quote-bot

# Use a minimal base image to run the Go app
FROM scratch

# Copy the binary from the build stage
COPY --from=build /quote-bot /quote-bot

# Command to run the executable
CMD ["/quote-bot"]