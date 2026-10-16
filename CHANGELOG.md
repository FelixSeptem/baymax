# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and follows Semantic Versioning.

## [Unreleased]

### Added
- 宿主恢复重验证契约 `host_resume_admission.v1` 与 `runner.WithHostResumeVerifier`：默认关闭，在原生动态 action resume 的 source admission 边界重新验证可信暂停绑定和当前授权、revision、plan、scope、confirmation、effect 事实；提供离线示例与双平台 contract gate。
- Provider 请求侧投影契约 `provider_request_projection.v1`：`model/conformance` 新增 `source`/`observed` 双投影、canonical digest 与 11 个稳定 drift/gap 分类码。
- 版本化 fixture `tool/diagnosticsreplay/testdata/model_request_projection.v1.json`（OpenAI / Anthropic / Gemini × Run / Stream + 无 tool result 分支），以及离线只读 replay 与幂等校验。
- 请求投影 contract gate `scripts/check-provider-request-projection-contract.sh` / `.ps1`，并对等接入 `scripts/check-quality-gate.sh` / `.ps1`。
- provider 请求投影审计测试（`model/openai`、`model/anthropic`、`model/gemini`）与 provider-neutral 边界守卫（`tool/contributioncheck`）。

### Changed
- `types.TokenUsage`、`ModelResponse.Usage`、`RunResult.TokenUsage` 的字段与既有语义保持不变（新增守卫测试固定其冻结形状）。
- 文档同步：`model/README.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`docs/pi-agent-comparison-and-adoption-study.md`、`README.md`。

### Fixed
- 动态 action continuation 保留原迭代与 tool result；执行中的重复恢复返回稳定的 `ErrDynamicActionInProgress`，并通过 checkpoint generation 隔离后续暂停，防止旧 continuation 覆盖新 checkpoint。
- 动态 action pause 与 retained duplicate 结果隔离可变引用；准入后执行失败通过 `DynamicActionContinuationError` 保留原错误 cause，Host 首次/duplicate admission 仍为 accepted，执行失败由 source terminal 事件表达。

### Breaking Changes
- 动态 action duplicate 恢复要求全部 normalized decision identity 字段与 Run/Stream mode 一致，包括未安装宿主 verifier 的路径。调用方应重复原决策字段，并在 `ErrDynamicActionInProgress` 后等待 continuation 完成；checkpoint ID 是 opaque token，不应解析或重新构造。迁移细节见 [宿主恢复准入指南](docs/guides/host-resume-admission.md)。

### Security
- 
