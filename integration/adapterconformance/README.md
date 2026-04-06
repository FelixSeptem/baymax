# integration/adapterconformance 测试说明

## 功能域

`adapterconformance` 负责外部 adapter 一致性矩阵验证，覆盖 `mcp|model|tool` 主类别与 sandbox/memory 扩展轨道。

## 关键内容

- 最小能力矩阵与优先级约束：`MinimumMatrix`
- manifest/profile/negotiation 对齐校验
- deterministic 错误分类与 reason taxonomy 稳定性
- sandbox backend/session/capability 组合矩阵回归

## 关键入口

- `harness.go`
- `harness_test.go`
- `sandbox_matrix.go`
- `sandbox_matrix_test.go`
- `memory_matrix.go`
- `memory_matrix_test.go`

## 验证命令

- `go test ./integration/adapterconformance -count=1`

## 维护约束

- 新增 adapter contract 字段时，需同步更新 matrix 断言与 drift 分类。
- reason code 命名空间必须稳定，避免回放与门禁误报。
