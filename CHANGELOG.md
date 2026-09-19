# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and follows Semantic Versioning.

## [Unreleased]

### Added
- Provider 请求侧投影契约 `provider_request_projection.v1`：`model/conformance` 新增 `source`/`observed` 双投影、canonical digest 与 11 个稳定 drift/gap 分类码。
- 版本化 fixture `tool/diagnosticsreplay/testdata/model_request_projection.v1.json`（OpenAI / Anthropic / Gemini × Run / Stream + 无 tool result 分支），以及离线只读 replay 与幂等校验。
- 请求投影 contract gate `scripts/check-provider-request-projection-contract.sh` / `.ps1`，并对等接入 `scripts/check-quality-gate.sh` / `.ps1`。
- provider 请求投影审计测试（`model/openai`、`model/anthropic`、`model/gemini`）与 provider-neutral 边界守卫（`tool/contributioncheck`）。

### Changed
- `types.TokenUsage`、`ModelResponse.Usage`、`RunResult.TokenUsage` 的字段与既有语义保持不变（新增守卫测试固定其冻结形状）。
- 文档同步：`model/README.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`docs/pi-agent-comparison-and-adoption-study.md`、`README.md`。

### Fixed
- 

### Breaking Changes
- 

### Security
- 
