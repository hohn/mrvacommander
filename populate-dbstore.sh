#!/bin/bash

# Configuration
MINIO_ALIAS="qldbminio"
MINIO_URL="http://localhost:9000"
MINIO_ROOT_USER="user"
MINIO_ROOT_PASSWORD="mmusty8432"
QL_DB_BUCKET_NAME="qldb"

# Check for MinIO client
if ! command -v mc &> /dev/null
then
    echo "MinIO client (mc) not found."
    exit 1
fi

# Configure MinIO client
mc alias set $MINIO_ALIAS $MINIO_URL $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD

# Create qldb bucket
mc mb $MINIO_ALIAS/$QL_DB_BUCKET_NAME

# Upload the two sample DBs with new names
mc cp cmd/server/codeql/dbs/google/flatbuffers/google_flatbuffers_db.zip \
    $MINIO_ALIAS/$QL_DB_BUCKET_NAME/google\$flatbuffers.zip

mc cp cmd/server/codeql/dbs/psycopg/psycopg2/psycopg_psycopg2_db.zip \
    $MINIO_ALIAS/$QL_DB_BUCKET_NAME/psycopg\$psycopg2.zip

# Check new disk use
du -k dbstore-data
