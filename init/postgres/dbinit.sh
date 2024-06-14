#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE server_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'server_db')\gexec
    SELECT 'CREATE DATABASE querypack_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'querypack_db')\gexec
    SELECT 'CREATE DATABASE qldb_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'qldb_db')\gexec
EOSQL
