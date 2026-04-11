# security-policy-event-delivery (minimal)

## Purpose
用真实语义链路演示 `security-policy-event-delivery` 的最小闭环：策略判定、事件投递、deny 语义保持。

## Run
go run ./examples/agent-modes/security-policy-event-delivery/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `security.policy_event_delivery`.
- Classification: `security.policy_delivery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/security,observability/event`.
- Semantic flow:
  - `security_policy_decision_emitted`: 产出 allow/deny、风险分值与判定原因。
  - `security_event_delivery_attempted`: 产出投递状态、重试次数与 fallback 队列状态。
  - `security_deny_semantic_preserved`: 验证 deny 语义不被投递链路结果改变。
- Related contracts: `security-baseline-s1; security-event-delivery`.
- Required gates: `check-security-policy-contract.*; check-security-event-contract.*; check-security-delivery-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=security.policy_event_delivery`
- `verification.semantic.classification=security.policy_delivery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/security,observability/event`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=security_policy_decision_emitted,security_event_delivery_attempted,security_deny_semantic_preserved`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `decision/delivery/fallback_queued/deny_preserved/effective_action` 等真实字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If deny semantics appear inconsistent,优先核对 `policy_decision` 与 `effective_action` 是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
