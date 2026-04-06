# model/providererror 组件说明

## 功能域

`model/providererror` 负责 provider 错误归类与统一语义映射，输出稳定的 `Class + Reason + Retryable` 组合。

## 架构设计

- `Classified`：承载归类后的错误结构，支持 `Unwrap()` 与 `ClassifiedError()`。
- `FromStatusCode(err, status)`：按 HTTP 状态码映射标准 reason（`auth|rate_limit|request_invalid|timeout|server`）。
- `FromError(err)`：按上下文超时、网络超时与错误文本模式做兜底归类。

## 关键入口

- `classified.go`

## 边界与依赖

- 该包只负责错误归类，不处理 provider SDK 请求/响应映射。
- reason taxonomy 需保持稳定，避免上层回放与治理门禁漂移。
- 该包与 `core/types` 对齐分类语义，不在本层引入执行策略。

## 配置与默认值

- 无独立 runtime 配置项。
- 未命中已知规则时，默认返回 `Class=ErrModel`、`Reason=unknown`、`Retryable=false`。

## 可观测性与验证

- `go test ./model/providererror -count=1`
- 与 provider 适配联调时，应覆盖 `integration/model_multi_provider_contract_test.go` 的错误语义断言。

## 扩展点与常见误用

- 扩展点：新增 provider 专有错误模式时，优先补 reason 映射与测试用例。
- 常见误用：直接透传原始错误而不归类，导致 run/stream 终态语义不一致。
