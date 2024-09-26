# ######################
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

# ######################
# Provide codeql and java
# 
FROM ubuntu:24.10 as runner
ENV DEBIAN_FRONTEND=noninteractive

# Build argument for CodeQL version, defaulting to the latest release
ARG CODEQL_VERSION=latest

# Install packages
RUN apt-get update && apt-get install --no-install-recommends --assume-yes \
    unzip \
    curl \
    ca-certificates \
    default-jdk

# If the version is 'latest', get the latest release version from GitHub, unzip
# the bundle into /opt, and delete the archive
RUN if [ "$CODEQL_VERSION" = "latest" ]; then \
    CODEQL_VERSION=$(curl -s https://api.github.com/repos/github/codeql-cli-binaries/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'); \
    fi && \
    echo "Using CodeQL version $CODEQL_VERSION" && \
    curl -L "https://github.com/github/codeql-cli-binaries/releases/download/$CODEQL_VERSION/codeql-linux64.zip" -o /tmp/codeql.zip && \
    unzip /tmp/codeql.zip -d /opt && \
    rm /tmp/codeql.zip && \
    chmod -R +x /opt/codeql

# Set environment variables for CodeQL
ENV CODEQL_CLI_PATH=/opt/codeql/codeql

# Set environment variable for CodeQL for `codeql database analyze` support on ARM
# This env var has no functional effect on CodeQL when running on x86_64 linux
ENV CODEQL_JAVA_HOME=/usr

# ######################

# Set the working directory inside the final image
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /work-gh/mrva/gh-mrva/gh-mrva /usr/local/bin/gh-mrva

# Put CodeQL on the PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/codeql

# Run forever
CMD ["tail", "-f", "/dev/null"]
