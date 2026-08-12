# TaskFlow

[English](./README.md) | 简体中文

TaskFlow 是一个面向内网协作场景的全栈任务工作流系统。当前仓库同时包含：

- Go HTTP API
- `web/` 下的 React 前端
- 面向 Mongo、发布校验和内网试点的脚本与运维文档

当前核心流程围绕一条明确的状态机展开：

```text
create -> assign -> start -> submit -> approve/reject -> close
```

## 当前能力范围

- 基于密码的登录、refresh token 轮转、密码重置、账号禁用、会话吊销
- 非开发环境下的 JWT 鉴权
- 健康检查、就绪检查、存活检查、指标、结构化日志、请求 ID、可选 OTLP tracing
- Mongo 迁移、带审计保留的软删除、限流、幂等
- 浏览器端登录页、任务列表、任务详情、创建任务、当前用户信息页面

## 仓库结构

```text
taskflow/
├── cmd/                 # 服务和迁移入口
├── internal/
│   ├── bootstrap/       # 应用组装
│   ├── config/          # 环境配置与严格生产校验
│   ├── domain/          # 聚合、状态机、ports
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 鉴权、日志、tracing、限流、幂等
│   ├── repository/      # Mongo 与内存实现
│   ├── router/          # 路由装配
│   └── service/         # 任务与身份用例
├── web/                 # React + Vite 前端
├── scripts/             # smoke、验收、审计、回滚脚本
├── deploy/              # 本地观测配置
├── reports/             # 发布、安全、性能说明
└── docs/                # 项目说明与本地计划目录
```

## 快速开始

### 1. 仅启动后端，最快本地闭环

要求：

- Go `1.25.12`

用内存仓储启动开发模式：

```bash
go test ./...

DEV_MODE=true \
TASK_REPOSITORY_DRIVER=memory \
JWT_SECRET=change-me-change-me-change-me-123 \
go run ./cmd/server
```

开发模式会预置这些用户：

- `u_test_001` / `creator-pass-123`
- `u_test_002` / `assignee-pass-123`
- `u_agent_001` / `agent-pass-123`

在 `DEV_MODE=true` 下，如果没有显式覆盖 `ALLOW_PUBLIC_REGISTER`，则 `POST /users` 会开放。

### 2. 本地预览前端

要求：

- Node `22`

先启动后端，再运行前端开发服务器：

```bash
cd web
npm ci
VITE_API_PROXY_TARGET=http://localhost:8080 npm run dev
```

浏览器打开 `http://127.0.0.1:5173`。

常用前端命令：

```bash
cd web
npm run lint
npm run test
npm run build
npm run preview
```

前端默认访问 `/api`，Vite 开发服务器会把它代理到 `VITE_API_PROXY_TARGET`，未设置时默认是 `http://localhost:8080`。

### 3. 本地 Mongo Compose 基线

启动带 Mongo 和 bootstrap 数据的后端栈：

```bash
docker compose up -d --build
bash scripts/compose_smoke.sh
```

如果还需要同域打包前端入口：

```bash
docker compose --profile full up -d --build web
bash scripts/nginx_smoke.sh
```

## API 概览

公开接口：

- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/password-reset/request`
- `POST /auth/password-reset/confirm`
- `POST /users`，仅在允许公开注册时开放

鉴权后接口：

- `GET /me`
- `GET /users`
- `POST /users/:id/disable`
- `POST /users/:id/revoke-sessions`
- `POST /tasks`
- `GET /tasks`
- `GET /tasks/:id`
- `PATCH /tasks/:id`
- `DELETE /tasks/:id`
- `POST /tasks/:id/{assign,start,submit,reject,approve,close,cancel,reactivate}`
- `GET /tasks/:id/records`
- `GET /tasks/:id/audit_logs`

系统接口：

- `GET /health`
- `GET /livez`
- `GET /readyz`
- `GET /metrics`

## 测试与发布校验

后端：

```bash
go test ./...
```

前端：

```bash
cd web
npm ci
npm run lint
npm run test
npm run build
```

仓库级 smoke / 验收脚本：

- `bash scripts/compose_smoke.sh`
- `bash scripts/web_build_smoke.sh`
- `bash scripts/web_acceptance_smoke.sh`
- `bash scripts/nginx_smoke.sh`
- `bash scripts/monitoring_smoke.sh`
- `bash scripts/intranet_acceptance.sh`
- `bash scripts/security_audit.sh`

GitHub Actions 会在 [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) 中同时执行前后端检查。

## 部署与运维文档

- 部署基线：[`DEPLOYMENT.md`](./DEPLOYMENT.md)
- Mongo 迁移纪律：[`MIGRATIONS.md`](./MIGRATIONS.md)
- 内网发布检查单：[`INTRANET_RELEASE_CHECKLIST.md`](./INTRANET_RELEASE_CHECKLIST.md)
- 首次上线 Runbook：[`INTRANET_RUNBOOK.md`](./INTRANET_RUNBOOK.md)
- 日常运维：[`INTRANET_OPS.md`](./INTRANET_OPS.md)
- 验收与 smoke 脚本说明：[`ACCEPTANCE_TESTING.md`](./ACCEPTANCE_TESTING.md)
- 个人/团队收尾与维护约定：[`docs/团队收尾.md`](./docs/团队收尾.md)

## 生产环境说明

- 生产部署应使用 `DEV_MODE=false`、`STRICT_PRODUCTION_CONFIG=true`、`TASK_REPOSITORY_DRIVER=mongo`。
- 严格生产配置现在还要求设置 `PASSWORD_RESET_WEBHOOK_URL`，避免密码重置 token 只在 `DEV_MODE` 下可见。
- 涉及事务的写路径要求 Mongo 以副本集成员或 `mongos` 方式提供，不支持独立单机 Mongo。
- 打包前端与 API 可以通过 compose 的 `full` profile 走同域入口。
