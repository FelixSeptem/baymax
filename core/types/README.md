# core/types 组件说明

## 功能域

`core/types` 定义跨模块共享契约，是 Baymax 的稳定接口层：

- Runner / Model / Tool / MCP / Skill 抽象接口
- Run / Model / Tool 调用数据结构
- 事件与 timeline 语义
- 错误分类与可重试语义

## 架构设计

该包保持“纯契约、零业务逻辑”定位：

- 上层编排使用 `Runner`、`ModelClient`、`EventHandler` 等接口解耦
- 下层实现通过 `RunRequest`、`ModelRequest`、`ToolCallOutcome` 等 DTO 对齐
- 可观测性使用统一 `Event` 与 `ActionTimelineEvent`
- 错误通过 `ErrorClass` + `ClassifiedError` 归一

策略契约同样集中在该包：

- 循环策略：`LoopPolicy`
- 本地并发策略：`LocalDispatchPolicy`
- MCP 策略：`MCPRuntimePolicy`
- Action Gate / HITL Clarification 契约

## 关键入口

- `types.go`

## 边界与依赖

- 该包不依赖上层实现细节，供 `core/*`、`model/*`、`orchestration/*`、`runtime/*` 共同复用。
- 任意跨模块行为变化应先更新此处契约，再同步实现与测试。
