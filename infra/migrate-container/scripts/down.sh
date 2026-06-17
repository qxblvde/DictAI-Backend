#! /bin/bash

migrate -source=file://migrations \
        -database=postgresql://"${POSTGRES_USER}":"${POSTGRES_PASSWORD}"@postgres:"${POSTGRES_PORT}"/dictai-test?sslmode=disable down 1