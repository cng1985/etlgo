# ETL Go

一个可运行的 ETL 控制台。后端使用 Go、Gin、GORM、Uber FX 和 SQLite，前端使用 Vue 3、JavaScript 和 Element Plus。

管道把数据从 CSV、Excel、JSON、SQLite、MySQL、PostgreSQL、SQL Server 或 HTTP 抽取出来，按顺序做过滤、选列、重命名、类型转换和派生计算，再写入 SQLite 表或 CSV / JSON 文件。

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

生产构建把 Vue 页面嵌进同一个 Go 二进制。先用脚本把前端生成到 `web/dist`，`web` 包再通过 `//go:embed all:dist` 嵌进去。启动后页面和接口都在 `:8080`：

```bash
bash scripts/embed-vue.sh
go build -o bin/etlgo ./cmd/server
./bin/etlgo
```

Windows 使用同一套嵌入流程：

```bat
scripts\embed-vue.bat
go build -o bin\etlgo.exe .\cmd\server
.\bin\etlgo.exe
```

`scripts\embed-vue.bat` 会调用同目录的 `embed-vue.ps1`。`make build` 在 Windows 上走这个脚本，在 Linux / macOS 上走 `scripts/embed-vue.sh`。改完前端后也可以执行 `go generate ./web`，它调用的是 bash 脚本。

打开 `http://127.0.0.1:8080`。

环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DATA_DIR` | `data` | 样本、上传和导出目录 |
| `DB_PATH` | `data/etl.db` | SQLite 文件 |
| `GIN_MODE` | `release` | Gin 运行模式 |
| `API_TOKEN` | `etl-demo-token` | 接口令牌，请求头使用 `Authorization: Bearer <token>` |

首次启动会写入示例订单、客户和产品，并创建三条管道。

## 目录

```text
cmd/server             程序入口，只启动 app.Module
internal/app           组装各 FX 模块
internal/config        配置模块
internal/store         SQLite / GORM 模块
internal/service       数据源、管道、数据集模块，管道模块负责调度
internal/handler       HTTP 处理模块
internal/server        Gin 服务模块
internal/seed          示例数据模块
internal/engine        抽取、转换、加载
frontend               Vue 3 控制台
scripts/embed-vue.sh   Linux / macOS：把 Vue 构建结果写入 web/dist
scripts/embed-vue.bat  Windows：把 Vue 构建结果写入 web/dist
```

每个包导出自己的 `fx.Module`。`internal/app` 把配置、存储、服务、查询、示例数据、接口和 HTTP 服务按模块拼起来，`main` 不再逐个注册构造函数。管道模块在生命周期里启动调度器，server 模块在生命周期里监听端口。

## 接口

业务接口统一为 `POST /api/v1/{resource}/{action}`，请求体是独立的 JSON 对象，成功响应为：

```json
{"code":200,"msg":"success","data":{}}
```

分页接口额外返回 `page`、`size`、`total`、`totalPage` 和 `responseType: "page"`。页码从 1 开始，`size` 最大 200。时间为 `yyyy-MM-dd HH:mm:ss`。

动态查询走 `POST /api/v1/query/search`、`/query/list`、`/query/one`，模型名如 `pageForSource`、`listPipeline`、`oneRun`。过滤条件只接受白名单字段，排序字段同样白名单，不会把过滤值拼进 SQL。请求里不能带 `tenantId`，服务端固定使用当前租户。

文件上传使用 `POST /api/v1/file/upload`，表单字段为 `file`、`businessType`、`businessId`。进程探针 `GET /health` 不要求令牌。

## 数据源

| 类型 | 配置 |
| --- | --- |
| `csv` | `path`、`delimiter`、`hasHeader`。路径相对于数据目录 |
| `excel` | `path`、`sheet`、`hasHeader`。支持 `.xlsx`、`.xlsm`、`.xltx`，`sheet` 留空时读第一张工作表 |
| `mysql` | `host`、`port`、`user`、`password`、`database`、`query`。端口默认 3306 |
| `pg` | 同上，另有 `sslMode`：`disable`、`require`、`verify-ca`、`verify-full`。端口默认 5432 |
| `sqlserver` | 同上，`sslMode` 表示加密：`disable`、`false`、`true`。端口默认 1433 |
| `json` / `sqlite` / `http` | 保持原有文件、SQLite 查询和 HTTP JSON 配置 |

数据库查询只允许单条 `SELECT` 或 `WITH`。上传接口支持 CSV、JSON 和 Excel。

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
