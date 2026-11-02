## ADDED Requirements

### Requirement: Manifest SHALL distinguish coordinated durable scope from local reopen

For each history or continuation owner, the manifest SHALL expose deployment
profile, writer/fencing/transaction scope, current coordinator availability and
evidence reference. The built-in file coordinator MUST report same-host
multi-instance scope and MUST NOT report cross-machine support. It MUST NOT infer multi-writer or cross-machine support
from a FileStore path, successful reopen, DTO presence or a sibling owner.

#### Scenario: Local owner remains local in manifest
- **WHEN** an owner uses its default file or memory store
- **THEN** the manifest reports its existing local scope and does not set
  coordinated/multi-writer capability

#### Scenario: Lost coordination evidence marks owner unavailable
- **WHEN** a coordinated owner cannot validate its live fence or coordinator
- **THEN** the manifest records an unavailable/unsupported capability cell with
  the source-owned evidence reason
