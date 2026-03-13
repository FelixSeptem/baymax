## 1. CA4 Threshold Strategy Convergence

- [x] 1.1 固化 CA4 阈值解析顺序（global/stage override/双触发择高）并补充注释与边界校验
- [x] 1.2 收敛 percent 与 absolute 触发冲突时的诊断字段口径（zone/reason/trigger source）
- [x] 1.3 补齐 stage1/stage2 覆盖规则的单元测试与契约测试

## 2. Token Counting Fallback Semantics

- [x] 2.1 固定 `sdk_preferred` 计数回退顺序（provider -> tiktoken -> lightweight estimate）
- [x] 2.2 明确 counting-only fail-open 语义，避免计数失败阻断 Run/Stream 主流程
- [x] 2.3 补充 OpenAI 场景下“阈值策略估算语义”测试与文档说明

## 3. Run/Stream CA4 Contract Tests

- [x] 3.1 增补 Run/Stream 在 CA4 边界输入下的语义等价测试
- [x] 3.2 增补 small delta / refresh interval 触发路径测试
- [x] 3.3 增补 fallback 分支覆盖（provider unsupported、local tokenizer unavailable）

## 4. Performance Gate Integration

- [x] 4.1 新增或更新 CA4 相关 benchmark（含 P95 维度）
- [x] 4.2 将 CA4 benchmark 相对百分比门禁接入现有质量流程
- [x] 4.3 更新 `docs/performance-policy.md` 的 CA4 验收口径与执行命令

## 5. Docs and Consistency

- [x] 5.1 同步 README 与 docs（`context-assembler-phased-plan`、`runtime-config-diagnostics`、`development-roadmap`）
- [x] 5.2 校验文档与实现一致性，修复漂移项
- [x] 5.3 在 change artifacts 中补充实现闭环（发现/修复/测试/门禁）记录

## 6. Final Validation

- [x] 6.1 执行 `go test ./...` 与 `go test -race ./...`
- [x] 6.2 执行 `golangci-lint run --config .golangci.yml`
- [x] 6.3 执行 `govulncheck ./...`（strict）并汇总验收证据
