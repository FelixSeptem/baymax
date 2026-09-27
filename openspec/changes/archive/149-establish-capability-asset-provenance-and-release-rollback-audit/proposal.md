## Why

Baymax 已分别拥有 Skill/extension manifest、adapter capability、memory lifecycle、context assembly 和 evaluation corpus 的局部身份与版本信息，但没有一个 bounded、reference-only 的跨能力资产投影来回答“当前运行引用了哪个版本、由谁拥有、依赖什么、哪些消费者会受撤回影响”。这会使能力漂移、跨作用域引用和发布回滚影响面只能靠人工比对；在继续扩展能力资产之前，应先把这些风险固定为可回放、可审计的离线合同。

## What Changes

- 新增版本化的 capability asset provenance projection，统一表达 stable identity、version、digest、owner、scope、来源、依赖、消费者引用和验证时间。
- 为 Skill、extension、adapter、memory、context 和 evaluation asset 建立有界的类型/引用规范；历史输入缺失新字段时使用 nullable/default 语义，未知 additive 字段安全忽略。
- 新增确定性的 provenance drift、scope violation、dependency mismatch、consumer reference conflict、withdrawal impact 和 replacement compatibility 审计分类。
- 新增离线 fixture、replay、正向/负向/边界测试以及 shell/PowerShell parity gate，验证重复回放、撤回影响计算和替代版本选择的稳定性。
- 保持 projection 为审计证据，不写入运行时事实源、不改变 RuntimeRecorder schema、不引入 registry、marketplace、动态下载、自动发布或运行时回滚状态机。

## Example Impact Assessment

无需示例变更（附理由）：首阶段只增加离线资产投影、回放与门禁，不改变 `examples/agent-modes` 的 semantic anchor、runtime path 或 expected markers。

## Capabilities

### New Capabilities

- `capability-asset-provenance-and-release-rollback-audit`: 对跨能力资产的身份、版本、依赖、消费者引用、漂移、撤回和替代版本兼容性进行 bounded、reference-only、可回放的离线审计。

### Modified Capabilities

- None. Existing extension, adapter, memory, context, evaluation and diagnostics requirements remain unchanged; the new capability consumes their declared metadata without changing runtime behavior.

## Impact

- Affected areas: `skill/loader` and existing manifest/resource metadata owners, memory/context/evaluation reference adapters, `tool/diagnosticsreplay`, offline fixtures and contract gate scripts, plus governance documentation.
- No new runtime configuration, persistence migration, hosted service, provider SDK dependency, or authoritative state store.
- Implementation must preserve library-first boundaries, `env > file > default` configuration semantics, additive/null/default compatibility, `RuntimeRecorder` single-writer ownership, and Run/Stream semantic equivalence.
