# MRVS 医药代表学术拜访管理系统

MRVS（Medical Representative Visit System）是一个面向医药代表（MR）的学术拜访管理系统，用于记录并校验 MR 与医生（HCP）之间的学术拜访过程。系统覆盖「计划 → 签到 → 学术沟通 → 签退 → 提交拜访记录」的完整闭环，并在签退时自动完成停留时长与位置偏离的合规校验。

前端为桌面端 Web 应用，后端为 Go 服务，附带演示主数据，可一键部署运行。

## 功能特性

- **拜访全流程**：创建拜访计划、现场 GPS 坐标签到、拜访签退、提交拜访记录、取消拜访。
- **合规校验**：签退时自动计算停留时长与最大位置偏离；停留不足 5 分钟或偏离医院 500 米以上自动标记为异常，结果在签退时固化。
- **拜访记录**：谈话要点、医生反馈、后续跟进计划、学术资料派发明细；提交后表单转为只读。
- **产品维度月度看板**：按产品统计月度拜访次数与合规情况，支持按月、MR、产品筛选。
- **主数据下拉**：医药代表、医院、医生（按医院级联）、产品。
- **统一接口规范**：统一响应包络、请求 ID、参数校验与业务错误码。
- **单镜像部署**：前端构建产物内嵌进 Go 二进制，由后端直接提供静态页面与 API。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、Vite 5、Element Plus、Vue Router、Vitest |
| 后端 | Go 1.21、CloudWeGo Hertz、GORM |
| 数据库 | MySQL 8.0 |
| 部署 | Docker 多阶段构建、Docker Compose |

## 目录结构

```text
mrvs/
├── backend/                 # Go 后端服务
│   ├── cmd/server/          # 服务入口
│   ├── internal/
│   │   ├── config/          # 环境变量配置
│   │   ├── database/        # 连接、迁移、种子数据
│   │   ├── handler/         # HTTP 处理器
│   │   ├── service/         # 业务逻辑
│   │   ├── repository/      # 数据访问
│   │   ├── compliance/      # 合规校验规则
│   │   ├── router/          # 路由与中间件装配
│   │   └── web/             # 内嵌前端静态资源
│   └── test/e2e/            # 端到端测试
├── frontend/                # Vue 3 前端
│   └── src/
│       ├── views/           # 拜访列表 / 新建 / 详情 / 看板
│       ├── api/             # 接口封装
│       └── utils/           # 时间、距离等格式化工具
├── docs/                    # 需求与后端设计文档
├── Dockerfile               # 前后端一体多阶段构建
├── docker-compose.yaml      # MySQL + 应用一键部署
└── start.sh                 # 服务启动脚本
```

## 快速部署（Docker Compose）

> 前置条件：安装 Docker Desktop / Docker Engine（含 `docker compose` 插件）。

```bash
# 一键构建并启动 MySQL 与应用
./start.sh
```

脚本会构建镜像、后台启动容器，并等待健康检查通过后输出访问地址。

打开浏览器访问：

- 应用：<http://localhost:8080>
- 健康检查：<http://localhost:8080/healthz>

也可以不使用脚本，直接执行：

```bash
docker compose up -d --build
```

首次启动会自动建表并写入演示主数据（2 位医药代表、3 家医院、6 位医生、3 个产品），重复启动不会重复写入。

### 常用运维命令

| 命令 | 说明 |
| --- | --- |
| `./start.sh` | 构建镜像并启动服务 |
| `./start.sh stop` | 停止服务，保留数据库数据卷 |
| `./start.sh restart` | 重启服务 |
| `./start.sh status` | 查看容器状态 |
| `./start.sh logs` | 跟随查看服务日志 |
| `./start.sh clean` | 停止服务并删除数据卷（不可恢复，需二次确认） |

### 部署参数

`start.sh` 与 `docker-compose.yaml` 支持通过环境变量或项目根目录 `.env` 文件覆盖默认配置：

```bash
# 使用 9090 端口启动
HOST_PORT=9090 ./start.sh
```

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `HOST_PORT` | `8080` | 宿主机访问端口（容器内固定 8080） |
| `HEALTH_TIMEOUT` | `120` | 启动时等待健康检查的超时秒数 |
| `MYSQL_ROOT_PASSWORD` | `root_pass` | MySQL root 密码 |
| `DB_NAME` | `mrvs` | 数据库名 |
| `DB_USER` / `DB_PASSWORD` | `mrvs` / `mrvs_pass` | 应用数据库账号 |
| `COMPLIANCE_MIN_DURATION_SECONDS` | `300` | 最短停留时长阈值（秒） |
| `COMPLIANCE_MAX_DISTANCE_METERS` | `500` | 允许的最大偏离距离（米） |
| `ALLOW_TIME_OVERRIDE` | `false` | 是否允许请求指定签到/签退时间（演示用） |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |

## 本地开发启动

> 前置条件：Go 1.21+、Node.js 22+、MySQL 8.0。

### 1. 准备数据库

创建数据库（账号可按需替换）：

```sql
CREATE DATABASE IF NOT EXISTS mrvs
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

### 2. 启动后端

```bash
cd backend
DB_HOST=127.0.0.1 \
DB_PORT=3306 \
DB_USER=root \
DB_PASSWORD=your_password \
DB_NAME=mrvs \
go run ./cmd/server
```

服务默认监听 `:8080`。启动时默认执行自动迁移（`AUTO_MIGRATE=true`）与种子数据写入（`SEED_DATA=true`）。

### 3. 启动前端

另开一个终端：

```bash
cd frontend
npm install
npm run dev
```

访问 <http://127.0.0.1:5173>，Vite 已配置将 `/api` 代理到 `http://127.0.0.1:8080`。

### 前端生产构建

```bash
cd frontend
npm run build
```

如希望后端直接托管构建产物，将 `frontend/dist` 复制到 `backend/internal/web/dist` 后重新编译后端即可（Dockerfile 已自动完成该步骤）。

## 接口概览

所有接口以 `/api/v1` 为前缀，响应统一为：

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "request_id": "..."
}
```

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/visits` | 创建拜访计划 |
| `GET` | `/api/v1/visits` | 拜访列表（支持状态、日期、MR、医院、产品、异常筛选与分页） |
| `GET` | `/api/v1/visits/:id` | 拜访详情 |
| `POST` | `/api/v1/visits/:id/check-in` | 现场签到 |
| `POST` | `/api/v1/visits/:id/check-out` | 拜访签退 |
| `PUT` | `/api/v1/visits/:id/report` | 提交/更新拜访记录 |
| `POST` | `/api/v1/visits/:id/cancel` | 取消拜访 |
| `GET` | `/api/v1/dashboard/products` | 产品维度月度看板 |
| `GET` | `/api/v1/meta/mrs` | 医药代表下拉 |
| `GET` | `/api/v1/meta/hospitals` | 医院下拉 |
| `GET` | `/api/v1/meta/hcps` | 医生下拉（可按 `hospital_id` 过滤） |
| `GET` | `/api/v1/meta/products` | 产品下拉 |

拜访状态流转：

```text
planned ──签到──> checked_in ──签退──> checked_out ──提交记录──> reported
   │
   └──取消──> cancelled
```

详细字段、错误码与校验规则见 [docs/backend-design.md](docs/backend-design.md) 与 [docs/spec.md](docs/spec.md)。

## 测试

### 后端单元测试

```bash
cd backend
go test ./...
```

### 后端端到端测试

```bash
cd backend
TEST_DB_DSN="user:pass@tcp(127.0.0.1:3306)/mrvs_test?charset=utf8mb4&parseTime=true&loc=UTC" \
  go test ./test/e2e -v
```

- 未设置 `TEST_DB_DSN` 时，会自动探测本机 `3306` / `3307` 的 `mrvs_test` 库。
- 测试库名必须包含 `test`；如确需使用其他库名，可设置 `E2E_ALLOW_NON_TEST_DB=1` 绕过。
- 测试会重建相关数据表并写入测试主数据，请勿指向业务库。

### 前端测试

```bash
cd frontend
npm test
```

## 常见问题

- **8080 端口被占用**：先停止占用进程，或执行 `HOST_PORT=9090 ./start.sh` 换端口。
- **数据库连接失败**：确认 MySQL 已启动、账号密码与 `DB_*` 变量一致；容器部署时 `DB_HOST` 应为 `mysql`。
- **Docker 未运行**：`start.sh` 会检测并提示先启动 Docker 守护进程。
- **页面空白或接口 404**：本地开发时确认 Vite 代理目标端口与后端监听端口一致；生产镜像需保证前端产物已内嵌进二进制。
