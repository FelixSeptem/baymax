## Example Impact Assessment

无需示例变更（附理由）

本 change 仅修复 `model/<provider>` 的 SDK 请求投影，并以 SDK 边界捕获、离线 fixture/replay 与 gate 验证；不新增 agent-mode 配置键、不改变 examples 的 runtime path 或 expected markers。若实施中的 Run/Stream/tool-result 行为使任何 `examples/agent-modes` 的可观察输出发生变化，必须先更新 `MATRIX.md` 与受影响模式 README，重新声明为“修改示例”，再继续相关代码任务。

## 1. Baseline, Branch, and Request-Facts Contract

- [ ] 1.1 在开始代码任务前，将当前已验证工作区提交/保存后从最新 `master` 创建语义化功能分支；验证 `git status --short`、`git branch --show-current` 与 worktree 路径均符合 AGENTS.md 分支生命周期要求。
- [ ] 1.2 审计 `core/runner` 的 `ModelRequest` 构造、`model/toolcontract.CanonicalInput`、三家 adapter 的 Generate/Stream/CountTokens 入口及归档 142 fixture；记录角色压平、文本信封、CountTokens 不对称与不修改 runtime/context/scheduler/ReAct 的可复现事实，并验证 audit 测试先红。
- [ ] 1.3 以测试先行定义 SDK-neutral canonical request interpretation：保留非空 source message 的顺序、在末尾追加非空 `Input`、保留有效 tool-result call/name/result 关联并拒绝无效关联；验证相同输入的归一化稳定且不含 SDK 类型、网络、时钟或持久化副作用。
- [ ] 1.4 为 canonical interpretation 定义有界输入与失败分类：空白项归一化、payload 上界、缺失 call/name 的既有 `feedback_invalid`、无法原生构造的稳定 request-shape 失败；验证错误发生在 provider 调用前且不存在文本信封 fallback。
- [ ] 1.5 为 OpenAI、Anthropic、Gemini 当前锁定依赖版本完成 SDK shape audit，分别钉住 system、user、assistant、tool-result/function-response 的官方构造入口和 CountTokens 能力边界；验证 SDK API 测试可编译且不把 SDK 类型带入 `model/toolcontract`。

## 2. Provider-owned Native Request Builders

- [ ] 2.1 在 `model/openai` 以测试先行实现私有 native request builder，使 Generate 与 Stream 保留 canonical role 顺序、最终 Input 位置和 tool-result call/name 关联；通过 `newResponse`/`newStream` 边界捕获验证原生 SDK 参数，不修改 provider selection 或终态路径。
- [ ] 2.2 为 OpenAI CountTokens 复用相同 canonical facts 或记录 API 不可表达字段的明确 projection capability；验证 CountTokens 与 Generate/Stream 的角色、顺序和关联事实等价，且未把不可用字段伪造成文本。
- [ ] 2.3 在 `model/anthropic` 以测试先行实现私有 native request builder，使 system、user、assistant 和有效 tool result 映射到官方 SDK 的原生消息/内容块；通过既有 GenerateFn/StreamFn 或 SDK 边界 seam 验证，无法安全构造时 fail-fast。
- [ ] 2.4 为 Anthropic CountTokens 复用相同 canonical facts；验证 CountTokens、Generate、Stream 的 normalized facts 对等，及成功/error tool-result、空内容、Unicode 和越界边界。
- [ ] 2.5 在 `model/gemini` 以测试先行实现私有 native request builder，使角色与有效 tool result 映射到官方 SDK 的原生 contents/parts/function response；通过既有 GenerateFn/StreamFn 或 SDK 边界 seam 验证原生关联与无文本信封 fallback。
- [ ] 2.6 为 Gemini CountTokens 复用相同 canonical facts；验证 CountTokens、Generate、Stream 的 normalized facts 对等，及 system 映射、assistant 映射、tool-result 成功/error 和边界失败。

## 3. Conformance, Replay, and Gates

- [ ] 3.1 扩展 `model/conformance` 的 provider-neutral observed projection，表达原生 role sequence、最终输入、native tool-result correlation 和 CountTokens capability exception；验证其只记录 digest/长度/枚举/关联标识，不保存 raw prompt、reasoning、credential 或无界 body。
- [ ] 3.2 将 `provider_request_projection.v1` 中本 change 实际修复的 role、tool-result-native、ordering 与 Run/Stream gaps 从 `declared_gap` 显式迁移为 no-gap native expectation，保留未解决的 capability/cache baseline；验证旧 gap 保留或已解决未更新都会导致 replay 失败。
- [ ] 3.3 扩展 diagnostics replay，覆盖 native role/order/correlation、invalid feedback、未知字段/历史 fixture、overflow、Run/Stream parity 与 CountTokens capability exception；验证同一 fixture 两次回放 canonical digest 与分类完全一致。
- [ ] 3.4 扩展 `tool/contributioncheck` 边界守卫，验证 SDK 类型只在 `model/<provider>`、`model/toolcontract` 不含 provider SDK/共享 wire 形状、无新 runtime 配置键/诊断写入/raw payload，taxonomy 与 specs/docs/gate 一致；验证每项守卫的负向变体可阻断。
- [ ] 3.5 更新 shell 与 PowerShell provider request projection gate，使其检查三 adapter 的 native SDK-boundary capture、fixture migration、Run/Stream/CountTokens parity、replay idempotency 与隐私/有界性；验证缺失 fixture、旧 declared gap 或文本信封回归会阻断两个 gate。

## 4. Documentation and Integrated Verification

- [ ] 4.1 更新 `model/README.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-module-boundaries.md` 与 `README.md`，说明 canonical facts 与 provider-owned native builders 的边界、fixture/gate 映射和不纳入的 cache schema；验证文档不宣称新的共享 protocol 或 runtime 控制面。
- [ ] 4.2 更新 `docs/development-roadmap.md`：将 Eval 首错归因候选与归档 141 对齐为已归档基线，并把 Provider 方向表述为归档 142 证据后的 native request projection 修复；运行 roadmap status consistency gate，验证无 `roadmap-status-drift`。
- [ ] 4.3 运行聚焦的 normal/race suites（`model/toolcontract`、`model/conformance`、三 adapter、`tool/diagnosticsreplay`、`tool/contributioncheck`），记录 role/tool-result 成功与失败、Run/Stream/CountTokens 对等、fixture migration 与 replay 幂等证据。
- [ ] 4.4 运行 `openspec validate --all`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、双平台 provider projection gate、`scripts/check-quality-gate.*` 与 `scripts/check-docs-consistency.*`；记录精确结果、环境受限命令及等价替代证据。
- [ ] 4.5 最终范围复核：确认没有新增 provider、共享 wire/gateway、credential store、remote catalog、runtime 配置键、context SDK 依赖、scheduler/ReAct/tail recap 接线、诊断 schema 或 raw payload 持久化；确认 feature branch 仅在门禁完成、归档并合并/推送后按 AGENTS.md 清理。
