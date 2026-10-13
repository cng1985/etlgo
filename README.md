# ETL Go

一个可运行的 ETL 控制台。后端使用 Go、Gin、GORM、Uber FX 和 SQLite，前端使用 Vue 3、JavaScript 和 Element Plus。

管道把数据从 CSV、JSON、SQLite 或 HTTP 抽取出来，按顺序做过滤、选列、重命名、类型转换和派生计算，再写入 SQLite 表或 CSV / JSON 文件。

## 启动

需要 Go 1.22+ 和 Node.js 22+。

```bash
npm install --prefix frontend
go run ./cmd/server
```

另开一个终端：

```bash
npm --prefix frontend run dev
```

打开 `http://127.0.0.1:5173`。接口由 Vite 代理到 `http://127.0.0.1:8080`。

生产构建后，由同一个 Go 进程同时提供页面和接口：

```bash
make build
./bin/etlgo
```

环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DATA_DIR` | `data` | 样本、上传和导出目录 |
| `DB_PATH` | `data/etl.db` | SQLite 文件 |
| `GIN_MODE` | `release` | Gin 运行模式 |

首次启动会写入示例订单、客户和产品，并创建三条管道。

## 目录

```text
cmd/server          程序入口，用 FX 组装依赖
internal/engine     抽取、转换、加载
internal/service    数据源、管道、调度和数据集
internal/handler    Gin 接口
internal/store      GORM / SQLite
frontend            Vue 3 控制台
```

FX 在 `cmd/server/main.go` 里提供配置、数据库、服务、路由和 HTTP 服务，并在生命周期里启动调度器。

## 转换步骤

步骤是一个 JSON 数组，按顺序执行。

- `filter`：`eq`、`ne`、`contains`、`not_contains`、`gt`、`gte`、`lt`、`lte`、`empty`、`not_empty`
- `select` / `drop`：保留或删除列
- `rename`：`mapping` 为原列名到新列名
- `cast`：`string`、`int`、`float`、`bool`
- `derive`：`concat`、`lower`、`upper`、`trim`、`replace`、`add`、`sub`、`mul`、`div`、`const`、`coalesce`

SQLite 目标支持 `replace`（覆盖）和 `append`（字段一致时追加）。文件路径必须位于数据目录内。`sources`、`pipelines`、`runs`、`datasets`、`seed_products` 是系统表，不能作为加载目标。

调度间隔为分钟。设为 `0` 时只手动运行；调度器每 20 秒检查一次到期管道。

## 测试

```bash
go test ./...
```
