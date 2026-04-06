# Offline Adapter Scaffold Cache Index

此目录用于本地离线 scaffold 临时产物缓存，不作为仓库主线事实源。

## 保留策略

- 仓库侧仅保留：`.gitkeep` 与本索引文件。
- 运行时生成目录（如 `a23-offline-*`）属于本地缓存，按需自行清理，不纳入版本控制。

## 最小可复现样本（canonical）

请使用以下已跟踪样本进行复现与调试：

- `integration/testdata/adapter-scaffold/mcp-fixture`
- `integration/testdata/adapter-scaffold/model-fixture`
- `integration/testdata/adapter-scaffold/tool-fixture`

## 说明

- 历史离线目录保留在本地仅用于临时排障，不代表主线 contract。
- 若需要重新生成离线缓存，请通过对应 scaffold 工具按需生成。
