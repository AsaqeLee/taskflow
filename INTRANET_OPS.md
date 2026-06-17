# TaskFlow 内网运维速查

一页纸运维手册，配合 `INTRANET_MVP.md` 使用。

## 日常巡检

```bash
docker compose ps
curl -s http://localhost:8080/readyz
curl -s http://localhost:8080/metrics | head
```

关注：`readyz` 非 200、Mongo 容器重启、磁盘占用。

## 重启服务

```bash
docker compose restart taskflow
```

Mongo 数据在 volume `taskflow_mongo_data`，重启 taskflow 不丢数据。

## 升级

```bash
git pull
docker compose build taskflow
docker compose up -d migrate taskflow
```

migrate 服务幂等；升级后确认 `/readyz` 与核心流程。

## 首批用户 / 加新用户

**冷启动（无用户）：**

```bash
cp .env.intranet.example .env   # 填写 JWT_SECRET
docker compose up -d --build    # migrate → bootstrap（自动）→ taskflow
```

`docker-compose.yml` 已包含 `bootstrap` one-shot 服务，默认读取 `scripts/users.example.json`。自定义用户文件可改 compose volume 挂载，或在本机执行：

```bash
USERS_FILE=./my-users.json ./scripts/bootstrap_users.sh
```

**已有 owner 后加成员：**

```bash
# owner 登录拿 token
curl -s -X POST http://<host>:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_owner","password":"<password>"}'

curl -X POST http://<host>:8080/users \
  -H "Authorization: Bearer <token>" \
  -H 'Content-Type: application/json' \
  -d '{"id":"u_bob","name":"Bob","role":"human","password":"<temp-pass>"}'
```

`ALLOW_PUBLIC_REGISTER=false` 时匿名无法注册。

## 验收脚本（API 级）

```bash
# 完整冷启动 + 全流程 + 重启保数据 + 备份恢复
COLD_START=1 ./scripts/intranet_acceptance.sh

# 已有 stack 上增量验收（不删 volume）
./scripts/intranet_acceptance.sh
```

P3-13 浏览器双人演练仍需人工：`cd web && npm run dev`，分别用 `u_owner` / `u_alice` 登录走 UI。

## 备份与恢复

```bash
# 方式 A：主机脚本（需安装 mongodump）
MONGODB_URI='mongodb://localhost:27018/?replicaSet=rs0' ./scripts/backup_mongo.sh

# 方式 B：经 mongo 容器（compose 环境推荐）
docker compose exec -T mongo mongodump \
  --uri='mongodb://127.0.0.1:27017/?replicaSet=rs0' \
  --db=taskflow --archive --gzip > backups/taskflow-$(date +%Y%m%d).gz

# 恢复（--drop 会先删库内集合）
MONGODB_URI='mongodb://localhost:27018/?replicaSet=rs0' ./scripts/restore_mongo.sh backups/taskflow-YYYYMMDD-HHMMSS.gz --drop
```

**cron 示例（每日 02:00）：**

```cron
0 2 * * * cd /opt/taskflow && MONGODB_URI='mongodb://127.0.0.1:27018/?replicaSet=rs0' ./scripts/backup_mongo.sh >> /var/log/taskflow-backup.log 2>&1
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

## 默认账号安全

bootstrap 脚本创建的临时密码仅用于首次登录，**首次登录后请修改密码**（第二期提供 UI；MVP 可用 API 或 owner 重建）。