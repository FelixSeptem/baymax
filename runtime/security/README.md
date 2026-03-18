# runtime/security 组件说明

## 功能域

`runtime/security` 当前聚焦脱敏基础能力，提供统一 redaction 组件：

- 结构化 payload 脱敏（map / nested object）
- JSON 文本脱敏
- 关键字匹配与可扩展 matcher

## 架构设计

实现位于 `runtime/security/redaction`：

- `Redactor` 基于关键词 token 规则判断敏感 key
- 命中后使用统一掩码值 `***`
- 支持默认关键词和运行时自定义关键词
- 支持 `Matcher` 扩展额外匹配策略

该能力被 `runtime/config.Manager`、`context/assembler`、`observability/event` 复用。

## 关键入口

- `redaction/redactor.go`

## 边界与依赖

- 安全脱敏是横切能力，必须保持纯函数行为和稳定输出语义。
- 不在该域混入调度、模型或传输逻辑，避免安全域职责扩散。
- 新增策略时需保证不破坏现有 key 分词匹配语义。
