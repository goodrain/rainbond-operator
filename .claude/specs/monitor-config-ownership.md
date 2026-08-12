# Monitor configuration ownership specification

The executable specification is [`monitor-config-ownership.yaml`](monitor-config-ownership.yaml).

It removes VM and AI Engine ownership of `rbd-system/prometheus-config`, moves
their static scrape definitions into operator configuration, and adds a
deterministic monitor configuration checksum to cause exactly one normal
StatefulSet rollout per actual configuration change.
