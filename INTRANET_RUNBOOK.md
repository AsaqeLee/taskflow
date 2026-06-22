# TaskFlow 内网首次上线 Runbook

面向第一次在内网主机部署 TaskFlow 的运维/开发同事。预计耗时 **30–60 分钟**。

## 前置条件

- Docker + Docker Compose 可用
- 内网主机可访问（建议仅内网网段）
- 已克隆本仓库

## 步骤 1：准备环境变量

**快捷方式（推荐本地/试点）：**

```bash
bash scripts/init_pilot_env.sh
# 生成 .env、scripts/users.intranet.json、evidence/pilot-credentials.txt
```

**手工方式：**

```bash
cp .env.intranet.example .env
```

编辑 `.env`，至少修改：

```env
JWT_SECRET=<openssl rand -hex 32 的输出>
APP_VERSION=<git tag 或 short SHA>
BOOTSTRAP_USERS_FILE=./scripts/users.intranet.json
STRICT_PRODUCTION_CONFIG=true
```

并先准备首批用户文件：

```bash
cp scripts/users.example.json scripts/users.intranet.json
# 把 scripts/users.intranet.json 中每个默认密码替换掉
```

生产若前后端分离，设置：

```env
CORS_ALLOWED_ORIGINS=https://taskflow.internal
```

`docker-compose.yml` 现在会直接读取这些 `.env` 参数，不再把 `JWT_SECRET`、`APP_VERSION`、`CORS_ALLOWED_ORIGINS` 和 bootstrap 用户文件写死成 `compose-local` 默认值。

启动前先校验生产参数：

```bash
bash scripts/validate_production_env.sh .env
bash scripts/release_candidate_check.sh .env
```

## 步骤 2：构建前端并启动栈

```bash
bash scripts/web_build_smoke.sh
docker compose up -d --build
```

启动顺序：`mongo` → `mongo-init` → `migrate` → `bootstrap` → `taskflow`。

确认健康：

```bash
curl -sf http://127.0.0.1:8080/readyz
docker compose ps
```

## 步骤 3：验证首批用户

默认账号来自 `BOOTSTRAP_USERS_FILE` 指向的文件。若你按上一步复制了 `scripts/users.intranet.json`，这里应以那个文件为准。仓库默认示例 `scripts/users.example.json` 只用于本地/CI。

```bash
curl -s -X POST http://127.0.0.1:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_owner","password":"<你在 BOOTSTRAP_USERS_FILE 里设置的 owner 密码>"}' | head
```

**bootstrap 密码必须视为临时密码**。MVP 期无 UI 修改密码，可由 owner 禁用后重建，或在上线前直接重新生成 bootstrap 用户文件。

## 步骤 4：自动化验收

```bash
# 不删 volume 的增量验收
./scripts/intranet_acceptance.sh

# 完整冷启动验收（会 docker compose down -v）
COLD_START=1 ./scripts/intranet_acceptance.sh
```

详见 [ACCEPTANCE_TESTING.md](./ACCEPTANCE_TESTING.md)。

## 步骤 5：启动前端（开发/演示）

```bash
cd web && npm ci && npm run dev
# 浏览器打开 http://localhost:5173
```

生产静态资源：`cd web && npm run build`，将 `web/dist` 交由 Nginx 同域托管或反代。

## 步骤 6：配置备份

```bash
mkdir -p backups
docker compose exec -T mongo mongodump \
  --uri='mongodb://127.0.0.1:27017/?replicaSet=rs0' \
  --db=taskflow --archive --gzip > backups/taskflow-$(date +%Y%m%d).gz
```

演练记录模板：`reports/backup-restore-evidence-template.md`。

## 步骤 7：HTTPS 与反代（推荐）

最小 Nginx 思路：

```text
https://taskflow.internal/
  /api/*  → proxy_pass http://127.0.0.1:8080/
  /*      → root /var/www/taskflow/web/dist
```

同域部署可省略 CORS。防火墙仅放行内网到 443。

## 升级与回滚

**升级：**

```bash
git pull
docker compose build taskflow
docker compose up -d migrate bootstrap taskflow
./scripts/intranet_acceptance.sh
```

**回滚：**

```bash
# 使用上一版镜像 tag（部署时建议打版本 tag）
docker tag taskflow-taskflow:previous taskflow-taskflow:latest
docker compose up -d taskflow
```

更完整的运维条目见 [INTRANET_OPS.md](./INTRANET_OPS.md)。

## 故障排查入口

| 现象 | 第一步 |
|------|--------|
| `/readyz` 非 200 | `docker compose logs taskflow mongo` |
| 登录 401 | 检查 bootstrap 是否成功、`docker compose logs bootstrap` |
| CORS 错误 | 检查 `CORS_ALLOWED_ORIGINS` 与浏览器 origin |
| 5xx | 响应 JSON 中的 `request_id` → 对照 taskflow 日志 |

## MVP 范围提醒

本阶段 **不包含**：用户管理 UI、密码重置页、公开注册页、任务分页搜索。见 `reports/mobile-acceptance-checklist.md`。
