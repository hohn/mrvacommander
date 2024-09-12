FROM codercom/code-server:4.92.2-debian

# ======================
# Pre-install a custom JDK for this platform and redirect CodeQL to it

USER root

ENV DEBIAN_FRONTEND=noninteractive

# Install packages
RUN apt-get update && apt-get install --no-install-recommends --assume-yes \
    ca-certificates                                 \
    curl                                            \
    default-jdk                                     \
    git                                             \
    libcurl4-openssl-dev                            \
    libssl-dev                                      \
    python3                                         \
    python3-dev                                     \
    unzip

# Build argument for CodeQL version, defaulting to the latest release
ARG CODEQL_VERSION=latest

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

# ======================
# Install code-server
USER coder

# Set environment variables
ENV PASSWORD mrva

# Install VS Code extensions as user root -- globally
RUN code-server --install-extension ms-python.python \
    && code-server --install-extension esbenp.prettier-vscode \
    && code-server --install-extension GitHub.vscode-codeql

# Expose the port that Code Server runs on
EXPOSE 9080

# Point CodeQL to the java binary for this platform
ENV CODEQL_JAVA_HOME=/usr

# Add
#   codeQl.cli.executablePath
# to user settings.
# This is in addition to the environment variable CODEQL_JAVA_HOME which has no
# effect on the plugin
USER root
COPY ./settings.json  /home/coder/.local/share/code-server/User/
RUN chown -R coder:coder /home/coder/.local/share/code-server/

# Start Code Server
ENTRYPOINT ["dumb-init", "code-server", "--bind-addr", "0.0.0.0:9080", "."]

# Run as the coder user
USER coder
