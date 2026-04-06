# model/toolcontract 组件说明

## 功能域

`model/toolcontract` 提供模型工具结果回灌的输入合同构建能力，负责把 `ToolResult` 规范化为统一 envelope。

## 架构设计

- `CanonicalInput(req)`：基于 `ModelRequest` 生成 canonical input。
- `WithCanonicalInput(req)`：在保留其他字段的前提下覆写 `req.Input`。
- tool feedback 固定使用 `FeedbackHeader`（`[tool_result_feedback.v1]`）+ JSON 数组格式。
- 非法条目（如空 `tool_call_id` 或空 `tool_name`）返回 `providererror.Classified`，并标记 `feedback_invalid`。

## 关键入口

- `input.go`

## 边界与依赖

- 该包只做输入合同规范化，不做 provider SDK 调用。
- 该包不处理网络重试、模型能力探测或流式协议映射。
- 错误分类复用 `model/providererror` 与 `core/types`，保持上层错误语义稳定。

## 配置与默认值

- 无独立运行时配置项；输出格式由常量合同固定。
- 当 `ModelRequest.Input` 为空时，回退使用最后一条 `Messages` 内容作为 base input。
- 若没有 `ToolResult`，返回原始 base input，不注入 feedback envelope。

## 可观测性与验证

- `go test ./model/toolcontract -count=1`
- 与 provider 适配联调时，应补 `integration/model_multi_provider_contract_test.go` 相关回归断言。

## 扩展点与常见误用

- 扩展点：新增 envelope 字段时，保持 additive 兼容并补充回放/合同测试。
- 常见误用：修改 header 或字段名而不更新合同测试，导致 provider 兼容性漂移。
- 常见误用：绕过输入合同层，直接拼接 provider 私有字段到 `Input`。
