# Use a Python 3.11 image as the base
FROM python:3.11-slim

# Install git
RUN apt-get update && apt-get install -y git

# Create the required directory structure
RUN mkdir -p /work-gh/mrva/

# Change to the directory and clone the repository
WORKDIR /work-gh/mrva/
RUN git clone https://github.com/hohn/mrvacommander.git && \
    cd mrvacommander && \
    git checkout hohn-0.1.24-demo

# Change to the client directory
WORKDIR /work-gh/mrva/mrvacommander/client/qldbtools/

# We're in a container, so use pip globally -- no virtual env
RUN pip install --upgrade pip

# Install the required Python packages from requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Install qldbtools
RUN pip install .

# Run forever
CMD ["tail", "-f", "/dev/null"]

