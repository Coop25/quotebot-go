# Use the official Golang image as the build stage
FROM golang:1.21-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Copy the source code into the container
COPY . .

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY *.go ./

# Copy the migrations directory into the container
COPY migrations ./migrations

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /quote-bot

# Copy the binary from the build stage
COPY --from=build /quote-bot /quote-bot

# Command to run the executable
CMD ["/quote-bot"]