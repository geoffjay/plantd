[![codecov](https://codecov.io/gh/geoffjay/plantd/graph/badge.svg?token=sHiAEpWC7e)](https://codecov.io/gh/geoffjay/plantd)
[![License MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://img.shields.io/badge/License-MIT-brightgreen.svg)

---

# Plantd

Core services for building distributed control systems.

Pre-alpha state, might burn your house down, don't use.

## Purpose

So much of `plantd` related tooling has been spread apart, the purpose of this
project is to attempt to bring together the `go` services that are actually
used in the hopes that all the rest can be archived one day.

## Quickstart

```shell
make
sudo make install
# eg. to test one of the services
PLANTD_PROXY_LOG_LEVEL=debug PORT=8080 plantd-proxy
```

For additional service control steps see the contents of `init/` for
`launchctl` and `systemd` options.

## Projects

### 🏚 Existing

The list of projects that should be brought into this one:

* [Broker][broker]
* [Command Line Client][plantcli]
* [Control Tool][plantctl]

### 🏠 Planned

* [Core](core/README.md)
* [Identity](identity/README.md)
* [Proxy](proxy/README.md)
* [State](state/README.md)
* [App](app/README.md) - Web application interface

## Documentation

### 📋 Configuration & Setup
- **[SSL/TLS Quick Reference](docs/ssl-quick-reference.md)** - Quick solutions for common SSL/TLS scenarios
- **[SSL/TLS Configuration Guide](docs/ssl-tls-configuration.md)** - Comprehensive guide for certificate management across all services
- **[App Service README](app/README.md)** - Web application setup and configuration
- **[Identity Service Documentation](identity/README.md)** - Authentication and authorization setup

### 🔧 Development
- **[SSL/TLS Testing](scripts/test-ssl)** - Built-in script for testing certificate configuration
- **Development Setup**: Use `overmind start` for local development
- **HTTP vs HTTPS**: Services support both modes (see SSL/TLS guide)

### 🏗 Architecture
- **[System Overview](docs/analysis/system-overview.md)** - High-level architecture
- **[Service Architecture](docs/analysis/service-architecture.md)** - Individual service designs
- **[Current State](docs/analysis/current-state.md)** - Project status and maturity matrix

## Contributing

It's recommended that some common tooling and commit hooks be installed.

```shell
make setup
```

Once complete you can start everything with `docker` and `overmind`.

```shell
docker compose up -d
overmind start
```

### Development Modes

**HTTPS (Default - Recommended):**
```shell
overmind start
# Access: https://localhost:8443 (accept self-signed certificate)
```

**HTTP (Development Alternative):**
```shell
export PLANTD_APP_USE_HTTP=true
overmind restart app
# Access: http://localhost:8080
```

For detailed SSL/TLS configuration, see the [SSL/TLS Configuration Guide](docs/ssl-tls-configuration.md).

<!-- links -->

[broker]: https://gitlab.com/plantd/broker
[plantctl]: https://gitlab.com/plantd/plantctl
[plantcli]: https://gitlab.com/plantd/plantcli
