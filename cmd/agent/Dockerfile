FROM golang:1.22 AS builder

# Copy the entire project
WORKDIR /app
COPY . .

# Download dependencies
RUN go mod download

# Set the working directory to the cmd/agent subproject
WORKDIR /app/cmd/agent

# Build the agent
RUN go build -o /bin/mrva_agent ./main.go

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

# If the version is 'latest', lsget the latest release version from GitHub, unzip the bundle into /opt, and delete the archive
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

# Copy built agent binary from the builder stage
WORKDIR /app
COPY --from=builder /bin/mrva_agent ./mrva_agent

# Run the agent
ENTRYPOINT ["./mrva_agent"]