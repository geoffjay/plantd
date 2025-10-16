# Image for the plantd identity service.
#
# When building from the root of the project:
#
# create: docker build -t geoffjay/plantd-identity:latest -f identity/Dockerfile .
# run:    docker run -it --rm --name identity \
#           -e PLANTD_BROKER_ENDPOINT=tcp://host.docker.internal:9797 \
#           -e PLANTD_IDENTITY_DATABASE_TYPE=sqlite \
#           -e PLANTD_IDENTITY_DATABASE_PATH=/data/identity.db \
#           geoffjay/plantd-identity:latest
# push:   docker push geoffjay/plantd-identity:latest

# Build stage
FROM golang:1.21.6-alpine3.19 AS build

# Install build dependencies
RUN apk update \
    && apk upgrade \
    && apk add --no-cache --virtual .build-deps \
        build-base \
        gcc \
        clang \
        pkgconf \
        zeromq-dev \
        czmq-dev \
        git \
        ca-certificates \
    && apk add --no-cache \
        bash

# Create app directory and copy source
RUN mkdir /app
COPY . /app/
WORKDIR /app

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o build/plantd-identity \
    ./identity/cmd/main.go

# Final stage
FROM alpine:3.19

# Install runtime dependencies and security updates
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
        zeromq \
        czmq \
        ca-certificates \
        sqlite \
        curl \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S plantd && \
    adduser -u 1001 -S plantd -G plantd

# Create data directory with proper permissions
RUN mkdir -p /data /app/config && \
    chown -R plantd:plantd /data /app

# Copy binary and set permissions
COPY --from=build /app/build/plantd-identity /usr/bin/identity
RUN chmod +x /usr/bin/identity

# Copy configuration files
COPY --from=build /app/identity/identity.yaml.example /app/config/identity.yaml

# Switch to non-root user
USER plantd

# Set working directory
WORKDIR /app

# Environment variables
ENV PLANTD_IDENTITY_CONFIG_PATH=/app/config/identity.yaml
ENV PLANTD_IDENTITY_DATABASE_TYPE=sqlite
ENV PLANTD_IDENTITY_DATABASE_PATH=/data/identity.db
ENV PLANTD_IDENTITY_LOG_LEVEL=info
ENV PLANTD_IDENTITY_LOG_FORMAT=json

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose ports
EXPOSE 8080

# Volume for data persistence
VOLUME ["/data"]

# Start the service
CMD ["/usr/bin/identity"]
