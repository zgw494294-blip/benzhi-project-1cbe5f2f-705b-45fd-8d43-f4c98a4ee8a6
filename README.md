# tape-preservation-gate

`tape-preservation-gate` 是面向声音档案馆的磁带数字化质量验收工作台。它把多节目段时间轴、盘前风险处置与复检、冻结计划批量采集、确定性质量检测、告警确认与缺陷定向重采、逐项整改复核和保存包凭据封存组织成一条可追溯流程，防止未经验证的数字副本进入长期保存库。

服务为单进程 Go 应用，不依赖外部数据库或 Node 构建链。业务快照保存在带 `schemaVersion` 的 JSON 文件中，审计事件保存在 JSON Lines 摘要链中；写入通过临时文件、`Sync` 和原子 `Rename` 完成。所有写接口要求 `expectedVersion` 和 `Idempotency-Key`，用于乐观并发检查和稳定重放。

## 构建

```bash
go build ./...
```

## 运行

默认仅监听高位回环地址 `127.0.0.1:19081`：

```bash
go run ./cmd/server
```

也可以显式指定地址与数据目录：

```bash
go run ./cmd/server -addr=127.0.0.1:19181 -data-dir=./data
```

未显式提供 `-addr` 时，可以通过 `PORT` 指定端口，服务仍只绑定 `127.0.0.1`。启动后访问 `http://127.0.0.1:19081/app`，即可在浏览器页面完成全部验收流程。

## 测试与自检

运行单元测试和集成测试：

```bash
go test ./...
```

真实 HTTP 自检会创建隔离临时数据目录，通过回环请求依次完成建批、载体检查、冻结计划、采集、质量检测、专业复核、凭据签发和独立核验，然后主动关闭服务并退出：

```bash
go run ./cmd/server -addr=127.0.0.1:19081 -selfcheck
```

更严格的本地检查可运行：

```bash
go test -race ./...
go vet ./...
go build ./...
```

## 主要接口

工作台页面为 `GET /app`，健康检查为 `GET /healthz`。JSON 接口统一位于 `/api/v1`，包含批次、载体检查、计划冻结、采集轮次、质量检测、缺陷处置、复核决定、凭据签发与凭据核验。错误响应使用中文说明，并区分输入错误、状态错误、资源不存在和版本冲突。

载体接口的 `segments` 支持一次提交最多 100 个节目段；详情中的 `planSummary` 返回规范化时间范围、计划总时长和空白时长。盘前检查接口通过 `action` 区分 `inspect`、`reinspect` 与 `treat`，历史检查和风险处置都只追加不覆盖。采集接口兼容原有单个 `run`，也可通过 `runs` 原子提交最多 100 行。缺陷处置接口以 `accept_warning` 确认非阻断告警，以 `replace` 比较 blocking 缺陷的定向重采证据。复核退回使用结构化 `reasons`，整改态再次提交时在 `remediations` 中逐项提供说明与批次内业务证据引用。

凭据封存后，文件清单、质量结论与批次均不可再修改。`GET /api/v1/batches/{batchID}/certificate/verification` 会重新规范化文件清单并计算摘要，独立判断凭据内容是否一致。
