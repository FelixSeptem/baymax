# Agent Modes Doc Baseline Freeze

更新时间：2026-04-09
冻结标识：`doc-baseline-v1`

## Scope

- 目录：`examples/agent-modes`
- 模式数量：`28`
- 覆盖变体：`minimal` + `production-ish`

## Baseline Artifacts

- `MATRIX.md`（列结构与模式映射基线）
- `PLAYBOOK.md`（文档先行流程与回滚步骤）
- `*/minimal/README.md`（模式最小语义说明）
- `*/production-ish/README.md`（模式治理差异说明）

## Integrity Snapshot

- `matrix_sha256=5a6547144c9ab4a8c30edc96d030f0b1d6d27ffc8c80eced7f26d0e30f835b46`
- `playbook_sha256=c0a6cc5a49d023342693b7fa9642569d0afb132e6bd2228489a7f3622dbf7338`

## Usage Rule

- 代码替换任务必须以本基线为输入。
- 若文档基线发生变更，需先更新本文件再进入后续实现任务。
