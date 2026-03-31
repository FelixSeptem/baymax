## Context

A53（sandbox adapter conformance）与 A54（memory SPI + builtin filesystem）将引入更多 runtime 配置域、diagnostics additive 字段与 gate 维度。当前仓库已具备 `RuntimeRecorder` 单写、readiness preflight、diagnostics replay 与 quality gate 主链路，但缺少统一的“对外观测出口 + 事故取证包”合同层：
- 外部观测接入（OTLP/Langfuse 等）依赖分散实现，无法保证字段与语义一致；
- 诊断取证产物缺少统一 schema 和版本控制，跨环境复盘成本高；
- A53/A54 新字段若不一次性纳入 export/bundle contract，后续将重复拆提案补齐。

A55 目标是在不改变 `library-first` 边界和 Run/Stream 外部语义的前提下，补齐观测出口与取证 contract，并与 A53/A54 形成顺滑衔接。

## Goals / Non-Goals

**Goals:**
- 定义观测出口 profile contract：`none|otlp|langfuse|custom`，并提供统一 exporter SPI。
- 定义 diagnostics bundle contract：标准化 schema、版本、redaction、replay hints 与 gate fingerprint。
- 将 export/bundle 纳入 `runtime/config`（`env > file > default`）、fail-fast、热更新原子回滚治理。
- 将 export/bundle 纳入 readiness finding、diagnostics additive 字段、replay fixture 与 quality gate。
- 与 A53/A54 字段一体化对齐，避免后续重复在 sandbox/memory 上追加“观测补丁提案”。

**Non-Goals:**
- 不引入平台化观测控制面（UI、RBAC、多租户运营面板）。
- 不在 A55 扩展新的 provider/tool 执行语义，仅处理观测与取证 contract。
- 不要求内置所有第三方后端 SDK；`custom` profile 通过 SPI 扩展即可。

## Decisions

### Decision 1: 保持 `RuntimeRecorder` 单写不变，exporter 仅消费标准化事件流

- 方案：`RuntimeRecorder` 继续作为唯一写入入口，exporter 在写入后消费标准事件和诊断快照，不允许绕过 recorder 直接写外部系统。
- 备选：在各模块内直接调用 exporter。
- 取舍：直接调用实现简单，但会破坏单写与幂等保障；统一消费链路可保持现有 contract 稳定。

### Decision 2: 观测出口采用 profile + SPI 双层模型

- 方案：
  - profile：`none|otlp|langfuse|custom`；
  - SPI：`ExportEvents` / `Flush` / `Shutdown`（名称以实现为准），返回 canonical error taxonomy。
- 备选：仅内置单一 OTLP 实现。
- 取舍：单一实现短期快，但无法覆盖现网差异；profile+SPI 能兼顾主流后端和宿主扩展。

### Decision 3: exporter 失败策略采用有界队列 + 策略化退化

- 方案：`runtime.observability.export.*` 定义 `queue_capacity`、`flush_timeout`、`on_error=fail_fast|degrade_and_record`、`on_overflow=drop_oldest|drop_newest|fail_fast`。
- 备选：exporter 失败直接影响主执行路径。
- 取舍：主路径强依赖 exporter 会放大外部故障；策略化退化更符合库场景，且通过 diagnostics 保证可观测。

### Decision 4: diagnostics bundle 固化为版本化 manifest + artifact 集合

- 方案：bundle 至少包含：
  - manifest（schema/version/build/runtime metadata）
  - timeline window
  - diagnostics snapshots
  - redacted effective config
  - replay hints（fixture selector / normalization hints）
  - gate fingerprint（执行过的 gate 与版本）
- 备选：仅导出原始日志压缩包。
- 取舍：原始压缩包难以自动回放；版本化结构更利于 contract test 与跨环境比对。

### Decision 5: readiness preflight 对 export/bundle 做前置可用性评估

- 方案：新增 `observability.export.*` 与 `diagnostics.bundle.*` findings；strict 模式下 required 不可用视为 blocked，non-strict 下 degrade 并记录。
- 备选：仅运行时报错，不做 preflight。
- 取舍：仅运行时报错会导致“带病启动”；preflight 可前置阻断并与现有 admission 语义一致。

### Decision 6: replay 与 gate 一次性收口，避免后补治理

- 方案：新增 `observability.v1` fixture + drift 分类（export profile drift、sink mapping drift、bundle schema drift、redaction drift、fingerprint drift），并接入独立 gate + quality gate。
- 备选：先功能落地后补 replay/gate。
- 取舍：后补会导致语义窗口不稳定；一次性收口能减少后续重复提案。

## Risks / Trade-offs

- [Risk] 外部观测后端抖动可能导致 exporter 队列积压。  
  -> Mitigation: 有界队列 + overflow 策略 + `degrade_and_record` 默认策略。

- [Risk] bundle 字段增长带来体积膨胀与敏感信息泄漏风险。  
  -> Mitigation: A45 cardinality 约束复用 + 强制 redaction + bundle 大小阈值。

- [Risk] A53/A54 在研字段变化导致 A55 早期 schema 反复调整。  
  -> Mitigation: A55 仅在 A53/A54 冻结字段后合入，并通过 mixed fixtures 做向后兼容校验。

- [Risk] 新 gate 增加 CI 时长。  
  -> Mitigation: 拆分 smoke 与 full matrix；保留独立 required-check 候选以便按需启用。

## Migration Plan

1. 定义 `runtime.observability.export.*` 与 `runtime.diagnostics.bundle.*` schema、默认值与校验规则。
2. 实现 exporter SPI 与 profile resolver（`none|otlp|langfuse|custom`），接入 recorder 后消费链路。
3. 增加 export/bundle diagnostics additive 字段与 canonical reason taxonomy。
4. 实现 bundle 生成器与 versioned manifest，打通 redaction 与 replay hints。
5. 在 readiness preflight 增加 export/bundle findings 并接入 strict/non-strict 语义。
6. 增加 `observability.v1` replay fixtures 与 drift 分类，验证 mixed fixtures 兼容。
7. 新增 `check-observability-export-and-bundle-contract.sh/.ps1` 并接入 `check-quality-gate.*` 与 CI 独立 job。
8. 同步 `README`、runtime config/diagnostics 文档、mainline contract index、roadmap。

## Open Questions

- 默认 `on_error` 策略是否在 `export.profile!=none` 时统一为 `degrade_and_record`，或按 profile 区分默认值。
- bundle 产物存储目标（仅本地文件/宿主注入 sink）在首版是否需要同时支持；建议首版先固定本地文件并保留 SPI 扩展位。
