# 数据与事件流

```text
input/config -> admission -> context snapshot -> provider request
                                      |
                                tool/MCP attempt
                                      |
                         RuntimeRecorder (single writer)
                                      |
                    timeline + diagnostics + replay fixture
                                      |
                            terminal outcome/checkpoint
```

Canonical facts 只在其 owner 组件产生；投影层可以添加 nullable/default 字段，但不能重写 owner 事实。敏感输入应在 recorder 边界按安全策略脱敏，replay fixture 不携带 credential。

Run 与 Stream 必须对同一输入产生语义等价的工具决策、错误分类和终态；事件顺序可因传输方式不同而分段，但不可改变事实含义。

验证入口：[Diagnostics Replay](../diagnostics-replay.md)、[主线契约测试索引](../mainline-contract-test-index.md)。
