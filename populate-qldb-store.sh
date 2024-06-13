#!/bin/bash

# Configuration
MINIO_ALIAS="qldbminio"
MINIO_URL="http://localhost:9000"
MINIO_ROOT_USER="user"
MINIO_ROOT_PASSWORD="mmusty8432"
BUCKET_NAME="qldb"

# Check for MinIO client
if ! command -v mc &> /dev/null
then
    echo "MinIO client (mc) not found.  "
fi

# Configure MinIO client
mc alias set $MINIO_ALIAS $MINIO_URL $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD

# Create ql db bucket
mc mb $MINIO_ALIAS/$BUCKET_NAME

# Upload the two sample DBs
mc cp cmd/server/codeql/dbs/google/flatbuffers/google_flatbuffers_db.zip \
   cmd/server/codeql/dbs/psycopg/psycopg2/psycopg_psycopg2_db.zip \
   $MINIO_ALIAS/$BUCKET_NAME

# Check new diskuse
du -k minio-data
