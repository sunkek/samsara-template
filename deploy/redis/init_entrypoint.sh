#!/bin/bash

BIND="${BIND:-127.0.0.1}" \
PROTECTED_MODE="${PROTECTED_MODE:-yes}" \
ADMIN_PASSWORD="${ADMIN_PASSWORD:-password}" \
VOVLEE_USER="${VOVLEE_USER:-vovlee}" \
VOVLEE_PASS="${VOVLEE_PASS:-password}" \
envsubst < "/redis/redis.tmpl.conf" > "/redis/redis.conf"
