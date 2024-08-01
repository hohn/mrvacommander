import qldbtools.utils as utils
import pandas as pd
import numpy as np
import sys
from minio import Minio
from minio.error import S3Error
from pathlib import Path

#
#* Collect the information and select subset
#
df = pd.read_csv('scratch/db-info-2.csv')
seed = 4242
if 0:
    # Use all entries
    entries = df
else:
    # Use num_entries, chosen via pseudo-random numbers
    entries = df.sample(n=3,
                        random_state=np.random.RandomState(seed))
#
#* Push the DBs
#
# Configuration
MINIO_URL = "http://localhost:9000"
MINIO_ROOT_USER = "user"
MINIO_ROOT_PASSWORD = "mmusty8432"
QL_DB_BUCKET_NAME = "qldb"

# Initialize MinIO client
client = Minio(
    MINIO_URL.replace("http://", "").replace("https://", ""),
    access_key=MINIO_ROOT_USER,
    secret_key=MINIO_ROOT_PASSWORD,
    secure=False
)

# Create the bucket if it doesn't exist
try:
    if not client.bucket_exists(QL_DB_BUCKET_NAME):
        client.make_bucket(QL_DB_BUCKET_NAME)
    else:
        print(f"Bucket '{QL_DB_BUCKET_NAME}' already exists.")
except S3Error as err:
    print(f"Error creating bucket: {err}")

# (test) File paths and new names
files_to_upload = {
    "cmd/server/codeql/dbs/google/flatbuffers/google_flatbuffers_db.zip": "google$flatbuffers.zip",
    "cmd/server/codeql/dbs/psycopg/psycopg2/psycopg_psycopg2_db.zip": "psycopg$psycopg2.zip"
}

# (test) Push the files
prefix = Path('/Users/hohn/work-gh/mrva/mrvacommander')
for local_path, new_name in files_to_upload.items():
    try:
        client.fput_object(QL_DB_BUCKET_NAME, new_name, prefix / Path(local_path))
        print(f"Uploaded {local_path} as {new_name} to bucket {QL_DB_BUCKET_NAME}")
    except S3Error as err:
        print(f"Error uploading file {local_path}: {err}")


# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
