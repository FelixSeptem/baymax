# runtime/security/redaction 组件说明

## 功能域

`runtime/security/redaction` 提供统一脱敏能力，支持结构化 payload 与 JSON 文本的敏感字段掩码处理。

## 架构设计

- `Redactor`：按关键词和 matcher 规则执行脱敏。
- `SanitizeMap`：递归处理 `map[string]any` 与嵌套数组对象。
- `SanitizeJSONText`：对 JSON 字符串反序列化后再按同一规则脱敏。
- `NormalizeKeywords` / `DefaultKeywords`：关键词归一化与默认词典管理。

## 关键入口

- `redactor.go`

## 边界与依赖

- 该包保持纯脱敏语义，不承载 policy decision、egress 判定或 adapter 激活逻辑。
- 脱敏输出口径必须稳定，避免 diagnostics/logging 出现字段不一致。
- 新 matcher 只能扩展匹配能力，不应改变默认关键词基础语义。

## 配置与默认值

- 通过 `New(enabled, keywords, opts...)` 注入开关与关键词。
- `keywords` 为空时使用默认关键词集合：`token/password/secret/api_key/apikey`。
- 命中敏感键后使用统一掩码值 `***`。

## 可观测性与验证

- `go test ./runtime/security/redaction -count=1`
- 与调用侧联调时需确认 `runtime/config`、`observability/event`、`context/assembler` 输出口径一致。

## 扩展点与常见误用

- 扩展点：新增 matcher、扩展关键词词典、增加域特定敏感键规则。
- 常见误用：在上层重复实现脱敏逻辑，导致统一口径被破坏。
