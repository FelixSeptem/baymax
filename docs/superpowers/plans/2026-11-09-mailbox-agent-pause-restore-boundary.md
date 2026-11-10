# Mailbox Agent Pause Restore Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the existing opt-in mailbox handler continuation path into a bounded, evidence-backed public capability while preserving lifecycle-only defaults and owner boundaries.

**Architecture:** `orchestration/mailbox` remains the sole owner of the original delivery and continuation claim. A durable FileStore plus registered resumer/revalidator supports same-host single-writer sequential reopen; the manifest promotes each pause cell only from owner evidence, and planbridge delegates without creating execution state.

**Tech Stack:** Go, existing mailbox FileStore/worker APIs, capability manifest, offline replay fixtures, PowerShell and shell contract gates, OpenSpec governance scripts.

---

### Task 1: Establish the change contract

**Files:** `openspec/changes/mailbox-agent-pause-restore-boundary/{proposal.md,design.md,tasks.md,specs/*/spec.md}`; `docs/superpowers/specs/2026-11-09-mailbox-agent-pause-restore-boundary-design.md`

- [x] Confirm the two modified capability paths and the explicit no-goals in the four OpenSpec artifacts.
- [x] Run `openspec status --change mailbox-agent-pause-restore-boundary --json` and `openspec validate --all`; expected result is no missing required artifact once all files exist.
- [x] Keep the change artifacts free of user dirty files and record the public documentation and example impact commitments.

### Task 2: Implement successor checkpoint lineage

**Files:** `orchestration/mailbox/continuation.go`, `orchestration/mailbox/worker.go`

- [x] Add an owner mutation that turns a claimed handler pause into a successor continuation with `Generation+1`, `PredecessorCheckpointID`, and a newly sealed digest, while retaining the original claim/result projection.
- [x] Ensure successor creation validates the same message/delivery/lease/attempt and current consumer before writing; return typed lease/conflict/recovery errors without a second consume.
- [x] Add bounded validation for successor references and make duplicate successor requests idempotent by resume identity.
- [x] Run `go test ./orchestration/mailbox -run 'Continuation' -count=1`; expected result is PASS.

### Task 3: Cover all pause reasons after a fresh process reopen

**Files:** `orchestration/mailbox/worker_continuation_test.go`

- [x] Build a table-driven fixture that pauses for input, approval, and budget, closes/reopens the FileStore, and resumes through a new worker.
- [x] Count handler/effect invocations and assert the original handler count stays one and completed effect references are not invoked again.
- [x] Add a resumer-pause case that creates a successor and then verifies duplicate successor resume invokes the successor once.
- [x] Run `go test ./orchestration/mailbox -run 'TestContinuationWorker' -count=1`; expected result is PASS.

### Task 4: Close failure and terminal-policy evidence

**Files:** `orchestration/mailbox/worker_continuation_test.go`, `orchestration/mailbox/continuation_test.go` (if needed)

- [x] Add expiry, lease reclaim, foreign deployment, revalidation denial, and recovery-required tests asserting no resumer/effect invocation.
- [x] Cover successful ack, requeue retry, nack/DLQ, duplicate terminal projection, and lifecycle causation preservation.
- [x] Run `go test ./orchestration/mailbox -count=1`; expected result is PASS.

### Task 5: Verify manifest and planbridge boundaries

**Files:** `orchestration/planbridge/capability_test.go`, `orchestration/planbridge/mailbox_binding_test.go`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`

- [x] Assert lifecycle-only workers retain `mailbox_handler_safe_point_unsupported` and configured owners promote all three cells with the same-host scope.
- [x] Assert planbridge rejects unsupported deployment/pause requests before publish/consume/claim and delegates Run/Stream to one owner.
- [x] Run `go test ./orchestration/planbridge ./orchestration/mailbox -count=1`; expected result is PASS.

### Task 6: Add replay fixture and contract gates

**Files:** `scripts/check-mailbox-agent-pause-restore-contract.ps1`, `scripts/check-mailbox-agent-pause-restore-contract.sh`, `tool/diagnosticsreplay/testdata/mailbox_agent_pause_restore.v1.json`, `docs/mainline-contract-test-index.md`

- [x] Add a bounded fixture covering the three pause cells, source identity, generation lineage, duplicate outcome, lifecycle terminal projection, and explicit unsupported deployment.
- [x] Make both gates run the affected tests and scan continuation code for second worker/queue/consume ownership; use shared Go cache only.
- [x] Run both gate variants and expect exit code zero.

### Task 7: Update public docs and example

**Files:** `README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `examples/host-execution-plan/README.md`

- [x] Mark roadmap I active during implementation and describe the exact same-host limitation, then mark it archived only after the official archive script runs.
- [x] Update support matrices, owner boundaries, markers, rollback notes, and verification commands consistently.
- [x] Run documentation consistency, documentation-impact, example-impact, and roadmap-status checks; expected result is PASS.

### Task 8: Full verification and lifecycle completion

**Files:** OpenSpec archive/index and only I-related implementation/docs files

- [x] Run affected tests, race tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, quality/docs gates, `openspec validate --all`, and `git diff --check`.
- [x] Mark every task complete, archive with `scripts/openspec-archive-seq.ps1 -ChangeName mailbox-agent-pause-restore-boundary`, and update archive index/roadmap.
- [ ] Stage only I-related files, commit, fetch `origin/master`, fast-forward merge into `master`, push, verify `HEAD == origin/master`, and delete the merged local feature branch. Preserve unrelated user dirty files.
