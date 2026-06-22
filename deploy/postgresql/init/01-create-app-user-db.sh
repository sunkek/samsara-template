#!/usr/bin/env bash
set -euo pipefail

: "${APP_DB:?must be set}"
: "${APP_USER:?must be set}"
: "${APP_PASSWORD:?must be set}"

psql --username "postgres" <<-EOSQL
-- 1) Create an application user with a password
DO \$\$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${APP_USER}') THEN
      CREATE ROLE "${APP_USER}" WITH LOGIN PASSWORD '${APP_PASSWORD}';
   END IF;
END
\$\$;

-- 2) Create the application database (if missing)
SELECT 'CREATE DATABASE "${APP_DB}"' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${APP_DB}')\gexec

-- 4) Make the new user the owner of the new database
ALTER DATABASE "${APP_DB}" OWNER TO "${APP_USER}";

-- 5) Grant public permissions for migrations
GRANT USAGE, CREATE ON SCHEMA public TO "${APP_USER}";
EOSQL


