# security-policy-event-delivery (production-ish)

## Purpose
用真实语义链路演示 `security-policy-event-delivery` 的生产治理闭环：在最小链路上增加安全门控与 replay 绑定。

## Variant Delta (vs minimal)
- 生产场景会触发高风险 deny 判定，并模拟事件投递失败后入 fallback 队列。
- 在 deny 语义保持后增加治理判定：`allow / allow_with_security_hold / deny`。
- 追加 replay 绑定，确保安全决策链可复放审计。

## Run
go run ./examples/agent-modes/security-policy-event-delivery/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `security.policy_event_delivery`.
- Classification: `security.policy_delivery`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/security,observability/event`.
- Semantic flow:
  - minimal 的 3 步安全链路；
  - 追加 `governance_security_gate_enforced` 与 `governance_security_replay_bound` 两步治理链路。
- Related contracts: `security-baseline-s1; security-event-delivery`.
- Required gates: `check-security-policy-contract.*; check-security-event-contract.*; check-security-delivery-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P2`
- `verification.semantic.anchor=security.policy_event_delivery`
- `verification.semantic.classification=security.policy_delivery`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/security,observability/event`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=security_policy_decision_emitted,security_event_delivery_attempted,security_deny_semantic_preserved,governance_security_gate_enforced,governance_security_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay` 字段，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `deny_preserved`、`delivery_status` 与 gate 决策是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
