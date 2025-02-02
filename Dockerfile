FROM golang:1.22.11-bookworm AS base


# Move to working directory /build
WORKDIR /build

# Copy the go.mod and go.sum files to the /build directory
COPY go.mod go.sum ./

# Install dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application
RUN go build -o gometeo

# Document the port that may need to be published
EXPOSE 1051

# Start the application
CMD ["/build/gometeo", "-addr", ":1051", "-limit", "15"]