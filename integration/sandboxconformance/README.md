# integration/sandboxconformance 测试说明

## 功能域

`sandboxconformance` 负责沙箱执行隔离合同的一致性验证，覆盖 capability negotiation、session lifecycle 与 launch failure fallback 语义。

## 关键内容

- 最小后端矩阵（`linux-nsjail|linux-bwrap|oci-runtime|windows-job`）
- required capability 与 session mode 校验
- `per_call|per_session` 生命周期一致性
- fallback action（`deny|allow_and_record`）决策与 reason code 映射

## 关键入口

- `harness.go`
- `harness_test.go`

## 验证命令

- `go test ./integration/sandboxconformance -count=1`

## 维护约束

- 扩展新 backend 时，需先补最小矩阵与 deterministic 断言。
- fallback/reason taxonomy 变更必须同步 docs 与 contract index。
