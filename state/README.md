[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/state)](https://goreportcard.com/report/github.com/geoffjay/plantd/state)

---

# 🗄 Distributed State Management

Currently, modules that exist in a `plantd` network manage their own state and
there's not a good way of persisting data if the service goes down. The idea
behind this service would be to receive state updates using a PUB/SUB system,
and allow for some kind of PUSH/PULL by the modules to load and store state.

## Quickstart

### Modern Architecture (Recommended)

The State service now supports multiple transport protocols through a unified interface:

```bash
# Build the unified state service
make build-state

# Run with gRPC transport (recommended)
PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_ENABLE_MDP=false ./build/plantd-state

# Run with both transports during migration
PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_ENABLE_MDP=true ./build/plantd-state

# Use gRPC client
./build/plant state state-grpc get <key> --service=<scope> --grpc-endpoint=http://localhost:8080
```

### Legacy Architecture (Deprecated)

* build and run a `broker`

```shell
git clone git@gitlab.com:plantd/broker.git
cd broker
make
PLANTD_BROKER_CONFIG=configs/broker.yaml ./target/plantd-broker
```

* build and run `state`

```shell
make build-state
PLANTD_STATE_LOG_LEVEL=debug ./build/plantd-state
```

* build `client`

```shell
make build-client
```

* run test commands

```shell
./build/plant state set --service="org.plantd.Client" foo bar
./build/plant state get --service="org.plantd.Client" foo
```

## Transport Configuration

The State service supports multiple transport protocols:

### Configuration File (`config.yml`)

```yaml
transport:
  enable-mdp: true      # Enable MDP (ZeroMQ) transport
  enable-grpc: false    # Enable gRPC transport  
  grpc-port: "8080"     # Port for gRPC service
```

### Environment Variables

```bash
# Transport control
PLANTD_STATE_ENABLE_MDP=true/false    # Override MDP transport
PLANTD_STATE_ENABLE_GRPC=true/false   # Override gRPC transport  
PLANTD_STATE_GRPC_PORT=8080           # Override gRPC port

# Endpoints
PLANTD_STATE_BROKER_ENDPOINT=tcp://127.0.0.1:9797  # MDP broker endpoint
```

### Migration Strategy

1. **Current State**: MDP-only transport
2. **Transition**: Both MDP and gRPC enabled
3. **Target State**: gRPC-only transport

```bash
# Phase 1: Add gRPC alongside MDP
PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_ENABLE_MDP=true ./build/plantd-state

# Phase 2: Test gRPC client
./build/plant state state-grpc get <key> --grpc-endpoint=http://localhost:8080

# Phase 3: Disable MDP
PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_ENABLE_MDP=false ./build/plantd-state
```

## Example

If `libplantd` is installed the following example can be used to demonstrate
all of the currently available calls.

```python
import gi
import json

gi.require_version("Pd", "1.0")

from gi.repository import Pd


client = Pd.Client.new("tcp://127.0.0.1:7200")

service = "org.plantd.Test"
messages = [
    ("create-scope", json.dumps({"service": service})),
    ("set", json.dumps({"service": service, "key": "foo", "value": "oof"})),
    ("set", json.dumps({"service": service, "key": "bar", "value": "rab"})),
    ("get", json.dumps({"service": service, "key": "foo"})),
    ("get", json.dumps({"service": service, "key": "bar"})),
    ("delete", json.dumps({"service": service, "key": "foo"})),
    ("delete", json.dumps({"service": service, "key": "bar"})),
    ("delete-scope", json.dumps({"service": service})),
]

for message in messages:
    client.send_request("org.plantd.State", message[0], message[1])
    response = client.recv_response()
    print(f"{message[0]}: {response}")
```

The data sink can be tested with a source from `libplantd`.

```python
import time


# eg. if,
# frontend (XSUB) on @tcp://127.0.0.1:11000
# backend  (XPUB) on @tcp://127.0.0.1:11001 - plantd-state should be connected here

source = Pd.Source.new(">tcp://localhost:11000", "org.plantd.Test")
source.start()
for i in range(10):
    source.queue_message(f"{\"key\":\"foo\",\"value\":\"{i}\"}")
    time.sleep(.5)
source.stop()
```

The above assumes that a scoped bucket already exists with the name
`org.plantd.Test`, to set up a new bucket and publish to it the following could
be used.

```python
import time


client.send_request("org.plantd.State", "create-scope", json.dumps({"service": "org.plantd.Derp"}))
source = Pd.Source.new(">tcp://localhost:11000", "org.plantd.Derp")
time.sleep(.5)
source.start()
source.queue_message(f"derp")
time.sleep(.1)
source.queue_message(f"derp")
time.sleep(.1)
source.queue_message(f"derp")
time.sleep(.5)
source.stop()
client.send_request("org.plantd.State", "delete-scope", json.dumps({"service": "org.plantd.Derp"}))
```
