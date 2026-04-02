## 1. Runtime Config Schema and Validation

- [x] 1.1 在 `runtime/config` 新增 `runtime.hooks.*`、`runtime.tool_middleware.*` 配置字段与默认值。
- [x] 1.2 在 `runtime/config` 新增 `runtime.skill.discovery.*`、`runtime.skill.preprocess.*`、`runtime.skill.bundle_mapping.*` 字段与默认值。
- [x] 1.3 实现 A65 配置域 fail-fast 校验（枚举值、路径合法性、冲突策略合法性）。
- [x] 1.4 接入热更新非法回滚测试，确保配置快照原子回退。
- [x] 1.5 增加 `env > file > default` 解析优先级单测。

## 2. Lifecycle Hooks Core Wiring

- [x] 2.1 在 Runner 主循环接入六个固定 hook 点位：`before_reasoning|after_reasoning|before_acting|after_acting|before_reply|after_reply`。
- [x] 2.2 固化 hook 调用顺序与 context 透传模型，确保 deterministic 执行。
- [x] 2.3 实现 hook 失败策略（`fail_fast|degrade`）并统一 classified error 映射。
- [x] 2.4 增加 Run/Stream 等价测试（同输入同 hook phase 序列和终态）。

## 3. Tool Middleware Onion-Chain

- [x] 3.1 在 tool 调用链接入 onion middleware（inbound 正序 / outbound 逆序）。
- [x] 3.2 实现 middleware 短路返回语义和错误冒泡语义。
- [x] 3.3 接入 timeout/cancel 隔离，避免 middleware 泄漏 goroutine。
- [x] 3.4 增加 deterministic 顺序、短路、超时和错误分类单测。

## 4. Skill Discovery Source Unification

- [x] 4.1 实现 `runtime.skill.discovery.mode=agents_md|folder|hybrid` 解析与路由。
- [x] 4.2 实现 `runtime.skill.discovery.roots` 目录加载与路径校验。
- [x] 4.3 固化多来源 merge/dedup 顺序与冲突选择规则。
- [x] 4.4 增加 discovery source 矩阵测试（`agents_md|folder|hybrid`）与 deterministic 回归断言。

## 5. Discover/Compile Preprocess Stage

- [x] 5.1 将 `Discover/Compile` 挂入 Run/Stream 前统一预处理阶段。
- [x] 5.2 实现 `runtime.skill.preprocess.enabled` 与 `fail_mode=fail_fast|degrade` 语义。
- [x] 5.3 固化 preprocess 失败时的终态分类与 side-effect-free 语义。
- [x] 5.4 增加 Run/Stream 预处理等价测试与 degrade 场景测试。

## 6. SkillBundle Mapping Contract

- [x] 6.1 实现 `SkillBundle -> prompt augmentation` 映射模式与冲突策略。
- [x] 6.2 实现 `SkillBundle -> tool whitelist` 映射模式与冲突策略。
- [x] 6.3 增加 whitelist 上界约束（不得突破 A57 allowlist/sandbox/egress）。
- [x] 6.4 增加映射 deterministic 测试（模式组合、冲突仲裁、边界拒绝）。

## 7. Diagnostics and RuntimeRecorder Additive Fields

- [x] 7.1 在 `runtime/diagnostics` 增加 A65 additive 字段（hooks/middleware/discovery/preprocess/mapping）。
- [x] 7.2 在 `observability/event.RuntimeRecorder` 接入 A65 字段映射，保持单写入口与幂等。
- [x] 7.3 增加 QueryRuns parser compatibility 测试（additive + nullable + default）。
- [x] 7.4 增加 reason taxonomy drift guard，禁止同义字段重定义。

## 8. Replay Fixtures and Drift Taxonomy

- [x] 8.1 在 `tool/diagnosticsreplay` 新增 `hooks_middleware.v1` fixture 与归一化逻辑。
- [x] 8.2 新增 `skill_discovery_sources.v1` 与 `skill_preprocess_and_mapping.v1` fixtures。
- [x] 8.3 新增 drift 分类断言（至少覆盖 hooks 顺序、discovery source、bundle mapping）。
- [x] 8.4 增加 mixed-fixture 兼容测试（历史 fixtures + A65 fixtures）。

## 9. Gate and CI Wiring

- [x] 9.1 新增 `scripts/check-hooks-middleware-contract.sh/.ps1`。
- [x] 9.2 将 A65 gate 接入 `scripts/check-quality-gate.sh/.ps1` 并保持 shell/PowerShell 失败传播等价。
- [x] 9.3 在 gate 中实现 `control_plane_absent` 断言（禁止托管 hooks/middleware 控制面）。
- [x] 9.4 在 gate 中实现 impacted-contract suites 校验（按 A65 模块触发对应主干 suites）。
- [x] 9.5 在 CI 暴露 required-check 候选（`hooks-middleware-contract-gate`）。

## 10. Documentation and Validation

- [x] 10.1 更新 `docs/runtime-config-diagnostics.md`（A65 配置字段、默认值、失败语义）。
- [x] 10.2 更新 `docs/mainline-contract-test-index.md`（A65 fixtures + gate 映射）。
- [x] 10.3 更新 `docs/development-roadmap.md`（A65 从占位转可实施口径）。
- [x] 10.4 更新 `README.md` 与 `skill/loader/README.md`（discovery/preprocess/mapping 使用说明）。
- [x] 10.5 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 10.6 执行合同验证：`scripts/check-hooks-middleware-contract.sh/.ps1` 与 `scripts/check-quality-gate.sh/.ps1`。
