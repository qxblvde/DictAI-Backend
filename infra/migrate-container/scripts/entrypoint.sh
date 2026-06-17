#!/bin/bash
sleep 2

seqwall staircase --postgres-url=postgresql://"${POSTGRES_USER}":"${POSTGRES_PASSWORD}"@postgres:"${POSTGRES_PORT}"/dictai-test?sslmode=disable \
  --migrations-path=/migrations \
  --upgrade='/scripts/up.sh {current_migration}' \
  --downgrade='/scripts/down.sh' \
  --migrations-extension=.up.sql

TESTS_EXIT_CODE=$?

if [ $TESTS_EXIT_CODE -ne 0 ]; then
    exit 1
fi

migrate -source=file://migrations -database=postgresql://"${POSTGRES_USER}":"${POSTGRES_PASSWORD}"@postgres:"${POSTGRES_PORT}"/"${POSTGRES_DB}"?sslmode=disable up