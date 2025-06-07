# System Overview

## Project Purpose

Plantd is a distributed control system (DCS) framework designed to provide building blocks for industrial control applications. The project aims to consolidate previously scattered Go services into a unified platform that can support reliable, distributed control operations.

## High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Client      │    │      Proxy      │    │       App       │
│   (plant CLI)   │    │   (Protocol     │    │   (Web UI)      │
│                 │    │   Translation)  │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │              ┌───────┴───────┐              │
          │              │               │              │
          └──────────────┼───────────────┼──────────────┘
                         │               │
                    ┌────▼────┐     ┌────▼────┐
                    │ Broker  │     │  State  │
                    │ (MDP/2) │     │ (K/V +  │
                    │         │     │ PubSub) │
                    └─────────┘     └─────────┘
                         │               │
                    ┌────▼────┐     ┌────▼────┐
                    │Identity │     │ Logger  │
                    │(Future) │     │         │
                    └─────────┘     └─────────┘
                         │
                    ┌────▼────┐
                    │Modules  │
                    │(Echo,   │
                    │Metric)  │
                    └─────────┘
```

## Core Design Principles

### 1. Message-Oriented Architecture
- Built on ZeroMQ for high-performance, low-latency messaging
- Implements Majordomo Protocol v2 (MDP/2) for reliable request-reply patterns
- Supports publish-subscribe patterns for data distribution

### 2. Service-Oriented Design
- Each service has a single, well-defined responsibility
- Services communicate exclusively through message passing
- Loose coupling enables independent development and deployment

### 3. Distributed State Management
- Centralized state service for persistence and coordination
- Scoped data organization by service namespace
- Real-time state synchronization through pub/sub mechanisms

### 4. Modular Extension System
- Plugin-like modules for domain-specific functionality
- Standardized interfaces for workers, clients, sources, and sinks
- Hot-pluggable components without system restart

## Technology Stack

### Core Technologies
- **Language**: Go 1.21.5
- **Messaging**: ZeroMQ with custom MDP/2 implementation
- **Database**: SQLite (via BoltDB-like interface)
- **Configuration**: YAML-based configuration
- **Logging**: Structured logging with logrus

### Infrastructure
- **Containerization**: Docker with multi-stage builds
- **Orchestration**: Docker Compose for development
- **Process Management**: Overmind for local development
- **Monitoring**: Grafana + Loki + Promtail stack
- **Database**: TimescaleDB (PostgreSQL) for metrics
- **Caching**: Redis for session/temporary data

### Development Tools
- **Build System**: Make-based with Go workspaces
- **Code Quality**: golangci-lint with comprehensive rules
- **Testing**: Go testing with coverage reporting
- **Live Reload**: Air for development hot-reloading
- **API Documentation**: Swagger/OpenAPI generation

## System Boundaries

### Internal Services
- **Broker**: Message routing and reliability
- **State**: Distributed state management
- **Proxy**: Protocol translation gateway
- **Logger**: Centralized logging service
- **Identity**: Authentication/authorization (planned)
- **App**: Web-based management interface

### External Interfaces
- **Client CLI**: Command-line interface for system interaction
- **REST API**: HTTP/JSON interface via proxy service
- **ZeroMQ API**: Native high-performance interface
- **Web UI**: Browser-based management interface

### Extension Points
- **Modules**: Custom business logic components
- **Workers**: Request processing units
- **Sources/Sinks**: Data ingestion and output
- **Protocol Adapters**: Custom communication protocols

## Deployment Models

### Development
- Local development with Overmind process management
- Docker Compose for infrastructure services
- Hot-reload enabled for rapid iteration

### Production (Planned)
- Containerized deployment with orchestration
- Service mesh for inter-service communication
- Centralized configuration management
- Health monitoring and alerting

## Key Characteristics

### Strengths
- **High Performance**: ZeroMQ provides microsecond-level latency
- **Reliability**: MDP/2 ensures message delivery guarantees
- **Scalability**: Distributed architecture supports horizontal scaling
- **Modularity**: Clean separation of concerns and plugin architecture
- **Observability**: Comprehensive logging and monitoring integration

### Current Limitations
- **Pre-alpha State**: Not production-ready
- **Limited Documentation**: Sparse user and developer documentation
- **Incomplete Services**: Several services are stubs or minimal implementations
- **Testing Coverage**: Limited integration and end-to-end testing
- **Security**: No authentication or authorization implemented

## Target Use Cases

### Industrial Control Systems
- Process control and automation
- Data acquisition and monitoring
- Real-time control loops
- Equipment integration

### IoT and Edge Computing
- Sensor data collection and processing
- Edge device coordination
- Real-time analytics and alerting
- Device management and configuration

### Distributed Computing
- Microservice coordination
- Event-driven architectures
- Real-time data processing pipelines
- System integration and orchestration