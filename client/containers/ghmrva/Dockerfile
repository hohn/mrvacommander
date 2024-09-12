# Use an official Golang image as the base image
FROM golang:1.22 AS builder

# Set the working directory inside the container
WORKDIR /work-gh/mrva/gh-mrva

# Clone the repository
RUN git clone https://github.com/hohn/gh-mrva.git . &&\
    git checkout hohn-0.1.24-demo

# Download dependencies
RUN go mod download

# Build the Go binary
RUN go build .

# Use a minimal base image for the final container
FROM debian:bookworm

# Set the working directory inside the final image
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /work-gh/mrva/gh-mrva/gh-mrva /usr/local/bin/gh-mrva

# Run forever
CMD ["tail", "-f", "/dev/null"]
