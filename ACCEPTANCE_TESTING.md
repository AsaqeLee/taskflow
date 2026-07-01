## 内网验收测试说明

### 脚本

| 脚本 | 用途 |
|------|------|
| `scripts/intranet_acceptance.sh` | P2 + P3-10 ～ P3-13：Hermes API key、冷启动、重启保数据、备份恢复、任务流 API 验收 |
| `scripts/compose_smoke.sh` | compose 起服、登录、建任务、`/metrics` 可达 |
| `scripts/web_build_smoke.sh` | 前端 lint、test、build |
| `scripts/web_acceptance_smoke.sh` | 浏览器 owner/assignee 全流程 + 375px/768px 响应式验收 |
| `scripts/nginx_smoke.sh` | dockerized nginx 同域入口、`/api` 代理、安全响应头 |
| `scripts/monitoring_smoke.sh` | Prometheus / Alertmanager profile 起服、scrape 与规则加载校验 |
| `scripts/release_candidate_check.sh` | 候选版本统一门禁入口 |

### `intranet_acceptance.sh` 运行前提

1. Docker / Compose 可用，端口 `8080`、`27018` 未被占用。
2. 脚本在仓库根目录执行。
3. 需要 `python3`、`curl`、`docker compose`。
4. 默认使用 bootstrap 账号（`u_owner` / `u_alice`）及 `BOOTSTRAP_USERS_FILE` 指向文件中的密码。

### 常用命令

```bash
# 增量验收：重用现有栈，只验证流程
STACK_COMPOSE_UP_MODE=skip bash scripts/intranet_acceptance.sh

# 默认验收：必要时重建镜像并起栈
bash scripts/intranet_acceptance.sh

# 冷启动验收：删除 volume 后重建
COLD_START=1 bash scripts/intranet_acceptance.sh

# 自定义 API 入口
BASE_URL=http://taskflow.internal:8080 bash scripts/intranet_acceptance.sh

# 同域 nginx 入口验收（需 full profile）
STACK_COMPOSE_UP_MODE=skip bash scripts/nginx_smoke.sh

# 浏览器 UI + 响应式验收（需后端已就绪）
bash scripts/web_acceptance_smoke.sh

# 候选版本统一门禁
bash scripts/release_candidate_check.sh .env
```

### 覆盖场景

| 编号 | 场景 | 脚本行为 |
|------|------|----------|
| P2 | Hermes API key | owner 创建 agent、签发 key、agent 用 key 调 `/me`、执行任务流、吊销后复用失败 |
| P3-10 | 冷启动 | 可选 `COLD_START=1`：`down -v` -> `up` -> bootstrap 登录 |
| P3-11 | 重启保数据 | 创建 completed 任务 -> `restart taskflow` -> 再查询 |
| P3-12 | 备份恢复 | `mongodump` -> 删任务 -> `mongorestore --drop` -> 任务恢复 |
| P3-13 | 双人协作 | owner 创建/分配/审批/关闭 + Hermes agent API key start/submit |
| P3-13 UI | 浏览器与响应式 | `scripts/web_acceptance_smoke.sh` 跑浏览器与响应式验收 |
| P3-14 | 同域入口与安全头 | `scripts/nginx_smoke.sh` 验证 `/api` 代理与关键安全响应头 |
| P3-15 | 监控基线 | `scripts/monitoring_smoke.sh` 验证 Prometheus scrape 与规则加载 |

### 成功输出

`scripts/intranet_acceptance.sh` 成功时会输出：

```text
[acceptance] ALL PASSED: P2, P3-10 (cold=0|1), P3-11, P3-12, P3-13
```

### 失败定位

| 失败点 | 排查 |
|--------|------|
| `docker / docker compose unavailable` | 检查 Docker Desktop / Colima 是否启动；确认 `docker info`、`docker compose version` 可用 |
| `readyz not healthy` | `docker compose ps`；`docker compose logs taskflow migrate bootstrap` |
| `login ... returned 401` | bootstrap 是否 Exited 0；`docker compose logs bootstrap` |
| `Hermes API key /me expected 200` | `docker compose logs taskflow`；确认 owner 创建 API key 返回 201 |
| `backup archive empty` | mongo 容器是否 healthy；`docker compose logs mongo` |
| `restore did not bring task back` | 检查 archive 路径与 `mongorestore` 输出 |
| `same-origin login via nginx returned` | `docker compose logs web taskflow`；`bash scripts/nginx_smoke.sh` |
| `taskflow target is not up in Prometheus` | `docker compose --profile monitoring ps`；`docker compose logs prometheus alertmanager` |

验收产物（备份文件）默认写入 `backups/acceptance/`。

### CI 集成

`.github/workflows/ci.yml` 的 backend job 会执行以下关键检查：

1. `scripts/validate_production_env.sh`
2. `scripts/web_build_smoke.sh`
3. `scripts/security_audit.sh`
4. `scripts/compose_smoke.sh`
5. `scripts/monitoring_smoke.sh`
6. `scripts/nginx_smoke.sh`
7. `scripts/intranet_acceptance.sh`
8. `scripts/web_acceptance_smoke.sh`

发布候选版本时，推荐额外手工执行一次：

```bash
bash scripts/release_candidate_check.sh .env
```
