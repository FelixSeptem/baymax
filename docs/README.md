# Baymax 技术文档

本文档目录按读者任务分层。详细 contract 仍由代码、测试和既有专题文档负责；本页只提供路径和事实源导航。

## 三条读者路径

### 新用户：先跑起来

1. [根 README](../README.md) 了解定位、边界和最小示例。
2. [集成指南](guides/integration.md) 选择 local tool、MCP 或 Provider 接入方式。
3. [最佳实践](guides/best-practices.md) 了解 Run/Stream、超时、错误和诊断使用方式。

### 集成方：接入并验证

1. [架构总览](architecture/overview.md) 理解请求、工具、事件和终态。
2. [配置与诊断](guides/configuration-and-diagnostics.md) 确认 `env > file > default` 和 fail-fast 回滚。
3. [测试、回放与门禁](guides/testing-replay-gates.md) 选择 contract、replay 和 gate。
4. [组件责任](components/README.md) 查找 Provider、工具、MCP、适配器的 owner。

### 贡献者：修改并交付

1. [贡献指南](guides/contribution.md) 了解分支、验证、PR 和回滚。
2. [OpenSpec 工作流](guides/openspec-workflow.md) 创建提案并完成 Documentation Impact Assessment。
3. [故障排查](guides/troubleshooting.md) 处理测试、文件占用和门禁失败。

## 分层目录

- `architecture/`：系统概览、生命周期、数据/事件流、模块 ownership 和设计原则。
- `components/`：runner、context、provider、tool、orchestration、runtime、observability、adapter、MCP、extension、skills。
- `guides/`：集成、配置/诊断、测试/replay/gate、故障排查、最佳实践、贡献和 OpenSpec。
- [事实源与迁移矩阵](documentation-ownership.md)：canonical source、owner、迁移和验证命令。
- [写作与页面模板](documentation-style-guide.md)：术语、链接、弃用和 Documentation Impact Assessment 模板。

## 现有专题事实源

- [运行时配置与诊断](runtime-config-diagnostics.md)
- [运行时模块边界](runtime-module-boundaries.md)
- [主线契约测试索引](mainline-contract-test-index.md)
- [Diagnostics Replay](diagnostics-replay.md)
- [适配器模板索引](external-adapter-template-index.md)
- [适配迁移映射](adapter-migration-mapping.md)
- [版本与兼容](versioning-and-compatibility.md)
- [开发路线图](development-roadmap.md)
