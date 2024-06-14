# Use the ubuntu 22.04 base image
FROM ubuntu:24.10

# Set architecture to arm64
ARG ARCH=arm64
ARG AARCH=aarch64

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV CODEQL_VERSION=codeql-bundle-v2.17.5
ENV CODEQL_DOWNLOAD_URL=https://github.com/github/codeql-action/releases/download/${CODEQL_VERSION}/codeql-bundle-linux64.tar.gz
ENV JDK_VERSION=22.0.1
ENV JDK_DOWNLOAD_URL=https://download.oracle.com/java/21/latest/jdk-${JDK_VERSION}_linux-${AARCH}_bin.tar.gz
ENV JDK_DOWNLOAD_URL=https://download.java.net/java/GA/jdk${JDK_VERSION}/c7ec1332f7bb44aeba2eb341ae18aca4/8/GPL/openjdk-${JDK_VERSION}_linux-${AARCH}_bin.tar.gz

ENV CODEQL_JAVA_HOME=/usr/local/jdk-${JDK_VERSION}

# Install necessary tools
RUN apt-get update && \
    apt-get install -y curl tar && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add and extract the CodeQL bundle
RUN curl -L $CODEQL_DOWNLOAD_URL -o /tmp/${CODEQL_VERSION}.tar.gz && \
    tar -xzf /tmp/${CODEQL_VERSION}.tar.gz -C /opt && \
    rm /tmp/${CODEQL_VERSION}.tar.gz

# Add and extract the JDK
RUN curl -L $JDK_DOWNLOAD_URL -o /tmp/jdk-${JDK_VERSION}.tar.gz && \
    tar -xzf /tmp/jdk-${JDK_VERSION}.tar.gz -C /usr/local && \
    rm /tmp/jdk-${JDK_VERSION}.tar.gz

# Set PATH
ENV PATH=/opt/codeql:"$PATH"

# Prepare host mount point
RUN mkdir /mrva
