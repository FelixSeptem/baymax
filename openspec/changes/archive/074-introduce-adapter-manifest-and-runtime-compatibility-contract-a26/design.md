## Context

当前外部 adapter 接入链路已经具备：
- A21：模板与迁移映射；
- A22：一致性 conformance harness；
- A23：脚手架生成与 drift gate。

但缺少“接入前兼容边界”的结构化清单，导致运行时难以在加载阶段判定 adapter 是否与当前 Baymax 版本和能力契约匹配。结果是错误更晚暴露、失败语义不统一、维护成本升高。

## Goals / Non-Goals

**Goals:**
- 定义 adapter manifest 最小 schema，覆盖 `type/name/version/baymax_compat/capabilities/conformance_profile`。
- 在 runtime 接入路径提供 manifest fail-fast 兼容校验。
- 固化 `required + optional` 能力语义与降级行为。
- A23 脚手架默认生成 manifest，A22 conformance 校验 manifest 与实现一致性。
- 将 manifest 合法性检查纳入质量门禁阻断路径。

**Non-Goals:**
- 不引入 adapter marketplace/registry 平台能力。
- 不新增网络下载、签名分发或中心化发布流程。
- 不在 A26 扩展为供应链安全体系（仅做本地契约与兼容校验）。

## Decisions

### 1) 兼容表达采用 semver range，并允许 `-rc` 版本
- 方案：manifest 中 `baymax_compat` 使用 semver range；预发布版本（例如 `0.31.0-rc.1`）合法。
- 原因：与 `0.x` 阶段迭代节奏匹配，便于灰度验证。
- 备选：固定单版本号。拒绝原因：过于僵化，维护成本高。

### 2) manifest 缺失或非法时 fail-fast
- 方案：接入入口必须读取并校验 manifest；缺失/解析失败/语义非法均立即失败。
- 原因：避免“运行后才发现不兼容”。
- 备选：缺失时 best-effort 默认为兼容。拒绝原因：不可控风险过高。

### 3) 能力声明采用 required/optional 双层
- 方案：`required` 能力缺失立即失败；`optional` 缺失允许降级并记录标准 reason。
- 原因：兼顾安全边界和增量演进。
- 备选：单层能力列表。拒绝原因：无法区分硬依赖与可降级能力。

### 4) 脚手架与 conformance 同步消费 manifest
- 方案：A23 默认生成 manifest 模板；A22 conformance 增加 profile 对齐断言。
- 原因：保证“生成即校验”，避免模板与契约长期漂移。
- 备选：仅 runtime 校验 manifest。拒绝原因：反馈环过晚。

### 5) gate 集成优先复用现有路径
- 方案：新增 `check-adapter-manifest-contract.*`，由 `check-quality-gate.*` 调用阻断。
- 原因：延续已有门禁结构，降低集成复杂度。
- 备选：单独新增独立 CI job。拒绝原因：路径分散且维护成本高。

## Risks / Trade-offs

- [Risk] semver range 解析差异导致误判  
  → Mitigation: 固定解析库与测试矩阵（含 rc/边界值/非法输入）。

- [Risk] required/optional 分类滥用导致声明不可信  
  → Mitigation: conformance 对 required 能力做最小可执行断言。

- [Risk] A22/A23 并行改动引入脚本冲突  
  → Mitigation: 仅新增 manifest 子 gate，保持现有 gate 顺序与失败语义不变。

## Migration Plan

1. 新增 manifest schema 与校验库（含错误分类）。
2. 在 adapter 接入路径接入 fail-fast 兼容校验。
3. A23 脚手架生成 manifest 模板并填充默认字段。
4. A22 conformance 增加 manifest profile 一致性检查。
5. 新增 manifest contract 脚本并接入 quality gate。
6. 更新 README/模板文档/索引/roadmap。

回滚策略：
- 若 gate 稳定性不足，可先回滚 quality-gate 接入点；
- manifest 校验库与测试保留为非阻断路径，不影响主运行时行为。

## Open Questions

- 当前无阻塞问题；按推荐值冻结：
  - semver range（允许 `-rc`）
  - manifest 缺失 fail-fast
  - required/optional 双层能力语义
  - gate 默认阻断
