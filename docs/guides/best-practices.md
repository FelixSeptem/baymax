# 最佳实践

- 明确声明 Provider、model、tool、MCP endpoint 和 policy，不依赖自动发现。
- 把宿主 credential、超时、重试、限流和持久化策略留在宿主层。
- 对 Run 和 Stream 编写同一组业务终态断言。
- 使用结构化 tool schema 和 bounded 输入，避免把任意文本当作控制面。
- 诊断先脱敏再导出；fixture 不访问网络、不携带 secret。
- 配置变更先在文件或 fixture 中验证，再启用热更新；失败必须原子回滚。
- 新能力先写 OpenSpec、测试和文档影响评估，再改实现。
- 组件页面引用事实源和验证命令，避免 README 复制完整 contract。

不建议：隐式全局 registry、跨层导入 Provider SDK、旁路写诊断、为了通过门禁而放宽断言。

执行记录应消费真实 `types.EventHandler` 回调，用 [host/eventadapter](../../host/eventadapter/README.md) 显式绑定 source 和已授权 sink。不要把 RunResult 包装成额外 parent/terminal 事件；`input_required` 只表示暂停，未收到终态应记录 incomplete observation。业务 authorization、revision、effect key、receipt identity 和 commit gate 由宿主拥有。

直接事件的 EventID/TraceID 是 mapping reference，不是去重键；两条同 ID 回调都必须保留。并行 source 用独立 observer，不按时间戳生成全局 sequence。不合作 sink 会阻塞 source 回调；应响应独立 observer context 的 deadline，避免无界背压。sink error/timeout 可能发生在写入之后，必须作为 unconfirmed 处理，不能自动 retry。

快照接受有界 JSON-compatible 数据和标准 `time.Time`，拒绝自定义 JSON/Text marshaler。新增公开语义（包括 additive/opt-in）须同步其维护文档、spec、正负测试和门禁；运行 [专用 contract gate](../../scripts/check-native-execution-event-contract.ps1) 和离线示例后再交付。
