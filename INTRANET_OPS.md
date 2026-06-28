# TaskFlow 内网运维速查

一页纸运维手册，配合 `INTRANET_RELEASE_CHECKLIST.md`、`INTRANET_RUNBOOK.md`、`ACCEPTANCE_TESTING.md` 使用。

## 日常巡检

```bash
docker compose ps
docker compose --profile monitoring ps
curl -s http://localhost:8080/readyz
curl -s http://localhost:8080/metrics | head
curl -sf http://localhost:9090/-/ready
curl -sf http://localhost:9093/-/ready
```

关注：`readyz` 非 200、Mongo 容器重启、磁盘占用、Prometheus 无法 scrape `taskflow`。

## 重启服务

```bash
docker compose restart taskflow
docker compose --profile full restart web
```

Mongo 数据在 volume `taskflow_mongo_data`，重启 taskflow 不丢数据。

## 升级

```bash
git pull

# API-only
bash scripts/intranet_release.sh .env

# Same-origin nginx entry
TASKFLOW_RELEASE_INCLUDE_WEB=true bash scripts/intranet_release.sh .env
```

发布失败时，优先使用 `scripts/rollback_image.sh` 回滚到上一镜像；升级后确认 `/readyz`、同域入口与核心流程。

## 监控基线

```bash
docker compose --profile monitoring up -d
bash scripts/monitoring_smoke.sh
```

Prometheus 默认暴露在 `9090`，Alertmanager 默认暴露在 `9093`。

## 首批用户 / 加新用户

**冷启动（无用户）：**

```bash
cp .env.intranet.example .env
cp scripts/users.example.json scripts/users.intranet.json
# 修改 scripts/users.intranet.json 中每个默认密码
bash scripts/validate_production_env.sh .env
docker compose up -d --build
```

`docker-compose.yml` 已包含 `bootstrap` one-shot 服务，会读取 `.env` 中的 `BOOTSTRAP_USERS_FILE`。本地 / CI 默认值仍是 `scripts/users.example.json`，生产请改成你自己的文件，例如 `./scripts/users.intranet.json`。

如需在本机单独执行：

```bash
USERS_FILE=./my-users.json ./scripts/bootstrap_users.sh
```

**已有 owner 后加成员：**

```bash
curl -s -X POST http://<host>:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_owner","password":"<password>"}'

curl -X POST http://<host>:8080/users \
  -H "Authorization: Bearer <token>" \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_bob","name":"Bob","role":"human","password":"<temp-pass>"}'
```

`ALLOW_PUBLIC_REGISTER=false` 时匿名无法注册。

## 验收脚本

```bash
# 完整冷启动 + 全流程 + 重启保数据 + 备份恢复
COLD_START=1 ./scripts/intranet_acceptance.sh

# 已有 stack 上增量验收（不删 volume）
./scripts/intranet_acceptance.sh

# 浏览器与响应式
./scripts/web_acceptance_smoke.sh

# 同域 nginx / 监控 profile
./scripts/nginx_smoke.sh
./scripts/monitoring_smoke.sh
```

## 备份与恢复

```bash
# 方式 A：优先使用主机工具，缺失时可切换为 compose
MONGODB_URI='mongodb://localhost:27018/?replicaSet=rs0' ./scripts/backup_mongo.sh

# 方式 B：显式走 compose 容器
BACKUP_TOOL=compose ./scripts/backup_mongo.sh

# 校验最近备份是否新鲜且可校验
./scripts/backup_healthcheck.sh

# 恢复（--drop 会先删库内集合）
RESTORE_TOOL=compose ./scripts/restore_mongo.sh backups/taskflow-YYYYMMDD-HHMMSS.gz --drop
```

**cron 示例（每日 02:00 备份，02:15 校验）：**

```cron
0 2 * * * cd /opt/taskflow && BACKUP_TOOL=compose ./scripts/backup_mongo.sh >> /var/log/taskflow-backup.log 2>&1
15 2 * * * cd /opt/taskflow && ./scripts/backup_healthcheck.sh >> /var/log/taskflow-backup.log 2>&1
```

## 忘密码

MVP 期无自助重置页面：由 owner 禁用账号后重建，或第二期接邮件重置。

## 故障排查

| 现象 | 检查 |
|------|------|
| 401 / 403 | JWT 过期、`DEV_MODE`、账号是否 disabled |
| 5xx | 响应体 `request_id`，对照服务日志 |
| Mongo 连不上 | `docker compose logs mongo`、`rs.status()` |
| CORS 错误 | `CORS_ALLOWED_ORIGINS` 是否含前端 origin |
| `/api` 正常但首页异常 | `docker compose logs web`、`bash scripts/nginx_smoke.sh` |
| 监控页面空白 | `docker compose logs prometheus alertmanager`、`bash scripts/monitoring_smoke.sh` |

## 默认账号安全

bootstrap 脚本创建的临时密码仅用于首次登录。MVP 期可由 owner 通过 API 重建账号；正式环境建议尽快切到企业身份源。
