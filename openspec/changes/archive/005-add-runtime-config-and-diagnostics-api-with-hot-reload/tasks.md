## 1. Configuration Foundation

- [x] 1.1 Add runtime config schema structs with defaults for reliability, concurrency, diagnostics, and reload settings
- [x] 1.2 Integrate `github.com/spf13/viper` for YAML file loading and env binding with documented prefix/key mapping
- [x] 1.3 Implement deterministic precedence merge (`env > file > default`) and expose effective config snapshot builder
- [x] 1.4 Implement startup validation for required fields, numeric ranges, and enum values with fail-fast error returns

## 2. Hot Reload and Atomic Runtime Integration

- [x] 2.1 Implement file watch based hot reload pipeline (parse -> validate -> build snapshot -> atomic swap)
- [x] 2.2 Ensure invalid reload keeps current active snapshot unchanged and emits reload failure diagnostics
- [x] 2.3 Wire runtime components (`mcp/http`, `mcp/stdio`, runtime profile resolver) to read from shared immutable config snapshot
- [x] 2.4 Add concurrency-safe config access primitives for multi-goroutine read/write paths

## 3. Diagnostics API (Library Only)

- [x] 3.1 Add exported diagnostics APIs for recent run summaries and recent MCP call summaries with bounded history
- [x] 3.2 Add exported API for sanitized effective config output and define secret masking rules
- [x] 3.3 Remove or avoid CLI diagnostics entry points and keep diagnostics surface API-only
- [x] 3.4 Align MCP diagnostic fields across transports to preserve semantic consistency

## 4. Quality Gates and Documentation

- [x] 4.1 Add unit tests for precedence resolution, validation failures, and env/file/default merge behavior
- [x] 4.2 Add hot reload tests for successful swap, invalid update rollback, and concurrent read consistency
- [x] 4.3 Add race-oriented/concurrency tests for diagnostics record/query under load
- [x] 4.4 Update `README.md` and `docs/*` for config schema, env mapping, hot reload behavior, diagnostics API, and limitations
- [x] 4.5 Add/refresh `golangci-lint` recommended config file and document usage in developer docs
