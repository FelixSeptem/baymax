## 1. Profile Pack and Manifest Contract

- [ ] 1.1 扩展 adapter manifest schema，新增 sandbox backend/profile/platform/session 字段与校验。
- [ ] 1.2 增加 profile-pack 加载与合法值校验（`linux_nsjail|linux_bwrap|oci_runtime|windows_job`）。
- [ ] 1.3 补齐 manifest compatibility fail-fast 测试（host mismatch、session mode unsupported、missing profile）。
- [ ] 1.4 增加 profile-pack 文档化字段索引与默认值说明。

## 2. Conformance Harness Matrix

- [ ] 2.1 在 external adapter conformance harness 增加 mainstream sandbox backend matrix suites。
- [ ] 2.2 增加 capability negotiation 场景（required missing、optional downgrade）与断言。
- [ ] 2.3 增加 session lifecycle 场景（`per_call|per_session`、crash/reconnect、close idempotent）。
- [ ] 2.4 增加 canonical drift 分类输出与断言（backend/profile/manifest/session/taxonomy）。

## 3. Replay Profile and Backward Compatibility

- [ ] 3.1 新增 sandbox adapter replay fixtures（`sandbox.v1`）并接入离线 deterministic 验证。
- [ ] 3.2 增加混合 profile 回放测试（existing profiles + `sandbox.v1`）。
- [ ] 3.3 增加 drift 分类断言（`sandbox_backend_profile_drift`、`sandbox_manifest_compat_drift`、`sandbox_session_mode_drift`）。

## 4. Readiness and Gate Integration

- [ ] 4.1 在 readiness preflight 接入 `sandbox.adapter.*` finding 分类与 strict/non-strict 映射。
- [ ] 4.2 新增 `check-sandbox-adapter-conformance-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [ ] 4.3 在 CI 暴露 sandbox adapter gate 作为独立 required-check 候选。
- [ ] 4.4 验证 shell/PowerShell gate parity（失败传播与退出码语义一致）。

## 5. Templates, Migration Docs, and Index Sync

- [ ] 5.1 更新 `docs/external-adapter-template-index.md`，补齐四类 sandbox backend onboarding 模板。
- [ ] 5.2 更新 `docs/adapter-migration-mapping.md`，补齐 legacy wrapper 到 profile-pack adapter 的迁移映射。
- [ ] 5.3 更新 `docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`。
- [ ] 5.4 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
