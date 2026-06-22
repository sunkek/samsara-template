#!/bin/bash

BIND="${BIND:-127.0.0.1}" \
PROTECTED_MODE="${PROTECTED_MODE:-yes}" \
ADMIN_PASSWORD="${ADMIN_PASSWORD:-password}" \
APP_USER="${APP_USER:-app}" \
APP_PASS="${APP_PASS:-password}" \
envsubst < "/redis/redis.tmpl.conf" > "/redis/redis.conf"
