# Use Ubuntu 22.04 as the base image
FROM arm64v8/ubuntu:22.04

# Set environment variables to non-interactive to avoid prompts during installation
ENV DEBIAN_FRONTEND=noninteractive

# Update the package list
RUN apt-get update && \
    apt-get clean 

RUN mkdir /mrva

# Set the default command
CMD ["bash"]
