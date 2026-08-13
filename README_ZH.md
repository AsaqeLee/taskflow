# TaskFlow

[English](./README.md) | 简体中文

面向内网协作的任务工作流系统：Go API + React 工作台 + Mongo 持久化。

```text
create → assign → start → submit → approve/reject → close
```

内网 MVP / 试点候选。**维护模式**，不是企业生产级。

[![CI](https://github.com/AsaqeLee/taskflow/actions/workflows/ci.yml/badge.svg)](https://github.com/AsaqeLee/taskflow/actions/workflows/ci.yml)

## 演示

本地 compose 走查：登录 → 任务列表 → 已提交详情 → 审计 → 审批对话框。

![TaskFlow 走查](docs/demo/walkthrough.gif)

[MP4](docs/demo/walkthrough.mp4)

| 登录 | 任务列表 |
|---|---|
| ![登录](docs/demo/01_login.png) | ![任务列表](docs/demo/02_tasks_list.png) |

| 任务详情 | 审计 |
|---|---|
| ![任务详情](docs/demo/03_task_detail.png) | ![审计](docs/demo/04_task_audit.png) |

| 审批 | 用户 |
|---|---|
| ![审批对话框](docs/demo/05_approve_dialog.png) | ![用户](docs/demo/07_users.png) |

素材说明见 [`docs/demo/README.md`](docs/demo/README.md)。以上是本机演示栈，不是目标内网已上线。

## 能力范围

- 明确的任务状态机，动作受角色约束
- 后端返回 `available_actions`；前端优先用后端值，并保留 fallback 矩阵
- JWT 登录、refresh 轮转、密码重置、账号禁用、会话吊销
- 面向 Agent / 无人值守调用的 API Key
- 任务协作记录 + 审计日志
- 健康 / 就绪 / 存活 / metrics、结构化日志、可选 OTLP
- Mongo + memory 双持久化、migration、bootstrap、备份脚本
- 同域 nginx 工作台：列表、详情、新建、用户、个人信息

## 快速开始

### 只跑后端

需要 Go `1.25.12`。

```bash
go test ./...

DEV_MODE=true \
TASK_REPOSITORY_DRIVER=memory \
JWT_SECRET=change-me-change-me-change-me-123 \
go run ./cmd/server
```

开发模式预置账号（仅本地）：

- `u_test_001` / `creator-pass-123`
- `u_test_002` / `assignee-pass-123`
- `u_agent_001` / `agent-pass-123`

### 预览前端

需要 Node `22`。先启动后端：

```bash
cd web
npm ci
VITE_API_PROXY_TARGET=http://localhost:8080 npm run dev
```

打开 `http://127.0.0.1:5173`。Vite 会把 `/api` 代理到后端。

### 本地 Mongo + 界面（演示栈）

```bash
docker compose --profile full up -d --build
bash scripts/compose_smoke.sh
bash scripts/nginx_smoke.sh
```

- API：`http://127.0.0.1:8080`
- Web：`http://127.0.0.1:8081`
- 主机 Mongo：`127.0.0.1:27018`

Compose bootstrap 用户来自挂载的 users 文件（本地常见 `scripts/users.intranet.json`，仓库示例是 `scripts/users.example.json`）。不要提交真实密码。

## 仓库结构

```text
taskflow/
├── cmd/                 # server、migrate、bootstrap
├── internal/            # domain、handler、service、repo
├── web/                 # React + Vite 工作台
├── docs/demo/           # README 截图与走查视频
├── scripts/             # smoke、发布、审计
├── deploy/              # 本地观测配置
└── reports/             # 发布 / 安全说明
```

## API 概览

公开接口：

- `POST /auth/login` — 字段是 `id`，不是 `username`
- `POST /auth/refresh`
- `POST /auth/password-reset/request`
- `POST /auth/password-reset/confirm`
- `POST /users`，仅在允许公开注册时开放

鉴权后：

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

系统接口：`GET /health` · `GET /livez` · `GET /readyz` · `GET /metrics`

## 测试

```bash
go test ./...
go vet ./...

cd web
npm ci
npm run lint
npm run test
npm run build
```

辅助脚本：`scripts/compose_smoke.sh`、`scripts/web_build_smoke.sh`、`scripts/web_acceptance_smoke.sh`、`scripts/nginx_smoke.sh`、`scripts/intranet_acceptance.sh`、`scripts/security_audit.sh`。

CI：[`.github/workflows/ci.yml`](./.github/workflows/ci.yml)。

## 运维文档

- [`DEPLOYMENT.md`](./DEPLOYMENT.md)
- [`MIGRATIONS.md`](./MIGRATIONS.md)
- [`INTRANET_RELEASE_CHECKLIST.md`](./INTRANET_RELEASE_CHECKLIST.md)
- [`INTRANET_RUNBOOK.md`](./INTRANET_RUNBOOK.md)
- [`INTRANET_OPS.md`](./INTRANET_OPS.md)
- [`ACCEPTANCE_TESTING.md`](./ACCEPTANCE_TESTING.md)
- [`docs/团队收尾.md`](./docs/团队收尾.md)

生产应使用 `DEV_MODE=false`、`STRICT_PRODUCTION_CONFIG=true`、`TASK_REPOSITORY_DRIVER=mongo`，并配置真实 `PASSWORD_RESET_WEBHOOK_URL`。走事务的写路径需要副本集成员或 `mongos`。
