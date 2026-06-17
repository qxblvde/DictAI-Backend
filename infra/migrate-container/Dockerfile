FROM ghcr.io/realkarych/seqwall:latest

ARG POSTGRES_USER
ARG POSTGRES_PASSWORD
ARG POSTGRES_DB
ARG POSTGRES_PORT
ARG MIGRATION_VERSION

RUN apk upgrade && apk add \
    curl \
    tar \
    postgresql-client\
    bash \
    ca-certificates

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.0/migrate.linux-amd64.tar.gz \
    | tar xvz && \
    mv migrate /usr/local/bin/migrate

COPY ./migrations /migrations

COPY ./scripts /scripts

RUN chmod +x /scripts/*

ENTRYPOINT ["sh", "./scripts/entrypoint.sh"]