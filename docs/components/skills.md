# Skills

## Purpose

skill loader 提供可发现、可审查、可复用的工作流说明，不直接承担运行时终态或业务状态。

## Owned facts/state

skill metadata、触发条件、引用资源和加载结果属于 loader owner。

## Inputs/outputs

输入是 skill package 和用户意图；输出是 instructions、resource references 和审查提示。

## Dependencies and forbidden dependencies

loader 依赖 skill 目录和受控 provider；禁止隐式修改仓库或绕过用户确认。

## Lifecycle / failure / rollback

discover → read full instructions → resolve references → apply bounded actions。缺失 skill 时提供 fallback，不伪造资源。

## Observability

记录 skill selection 和资源读取范围；不记录不必要的用户内容。

## Verification / source links

`go test ./skill/... -count=1`；[skill loader README](../../skill/loader/README.md)。
