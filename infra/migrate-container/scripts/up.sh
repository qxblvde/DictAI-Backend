#! /bin/bash

MIGRATION_FILE=$(basename "$1")
MIGRATION_VERSION="${MIGRATION_FILE%%_*}"


migrate -source=file://migrations -database=postgresql://"${POSTGRES_USER}":"${POSTGRES_PASSWORD}"@postgres:"${POSTGRES_PORT}"/dictai-test?sslmode=disable goto "$MIGRATION_VERSION"