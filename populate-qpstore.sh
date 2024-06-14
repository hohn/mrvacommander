#!/bin/bash

# Configuration
MINIO_ALIAS="qpstore"
MINIO_URL="http://localhost:19000"
MINIO_ROOT_USER="user"
MINIO_ROOT_PASSWORD="mmusty8432"
BUCKET_NAME="qpstore"

# Check for MinIO client
if ! command -v mc &> /dev/null
then
    echo "MinIO client (mc) not found.  "
fi

# Configure MinIO client
mc alias set $MINIO_ALIAS $MINIO_URL $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD

# Create ql db bucket
mc mb $MINIO_ALIAS/$BUCKET_NAME
