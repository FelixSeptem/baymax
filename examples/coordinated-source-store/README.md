# Coordinated source store

This offline example creates an explicit same-host coordinator, acquires a
writer fence, publishes one revision-checked source snapshot, and proves that a
plain local deployment scope rejects a multi-instance request.

```bash
go run ./examples/coordinated-source-store
```

Expected markers:

```text
coordinated-cas=revision:1
local-multi-instance-rejected=true
cross-machine-supported=false
```

The coordinator is opt-in and source-owner scoped. It is a same-host,
multi-instance CAS/fencing boundary, not a distributed database, scheduler,
queue, or cross-machine recovery guarantee. Roll back by removing the explicit
coordinator configuration before new source writes; do not reinterpret retained
coordinated state through a local FileStore.
