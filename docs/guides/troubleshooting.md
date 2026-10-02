# 故障排查

| 症状 | 首查位置 | 常见原因 | 回滚/修复 |
| --- | --- | --- | --- |
| OpenSpec 状态漂移 | roadmap status gate | roadmap、目录或 archive 不同步 | 以 `openspec list --json` 为准修正文档 |
| 文档影响评估失败 | documentation-impact gate | 缺段落、非法 outcome、缺路径/验证 | 补齐 proposal/design/tasks |
| README 链接失败 | docs consistency | 页面迁移未留 stub | 恢复旧链接或添加迁移 stub |
| 配置热更新失败 | runtime config diagnostics | 校验失败或来源优先级错误 | 保留旧 snapshot，修正输入后重试 |
| Windows 测试文件占用 | package test/race | 并行编译缓存和 `.test.exe` 锁 | 隔离 `GOCACHE`/`GOTMPDIR`，串行逐包重试 |
| replay 不确定 | diagnostics replay | fixture 带时间/网络/secret | 固定输入，去除网络和敏感字段 |

所有门禁都应输出稳定 reason code；不要通过删除 fixture 或跳过失败检查来“修复”。
