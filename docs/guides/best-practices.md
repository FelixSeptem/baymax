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
