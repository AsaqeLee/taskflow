# 内网验收测试说明

## 脚本

| 脚本 | 用途 |
|------|------|
| `scripts/intranet_acceptance.sh` | P3-10～P3-12 API 级全流程验收 |
| `scripts/compose_smoke.sh` | compose 起服 + 登录 + 建任务 |
| `scripts/web_build_smoke.sh` | 前端 lint + test + build |
| `scripts/web_acceptance_smoke.sh` | 浏览器 owner/assignee 全流程 + 375px/768px 响应式验收 |

## `intranet_acceptance.sh` 运行前提

1. Docker / Compose 可用，端口 `8080`、`27018` 未被占用（或与 compose 配置一致）。
2. 脚本在仓库根目录执行。
3. 默认使用 bootstrap 账号（`u_owner` / `u_alice`）及 `BOOTSTRAP_USERS_FILE` 指向文件中的密码；若未设置，该默认值才是 `scripts/users.example.json`。
4. 需要 `python3`、`curl`、`docker compose`。

## 常用命令

```bash
# 增量验收（保留现有 volume）
./scripts/intranet_acceptance.sh

# 冷启动验收（删除 volume 后重建）
COLD_START=1 ./scripts/intranet_acceptance.sh

# 自定义入口
BASE_URL=http://taskflow.internal:8080 ./scripts/intranet_acceptance.sh

# 浏览器 UI + 响应式验收（需后端已就绪）
./scripts/web_acceptance_smoke.sh
```

## 覆盖场景

| 编号 | 场景 | 脚本行为 |
|------|------|----------|
| P3-10 | 冷启动 | 可选 `COLD_START=1`：down -v → up → bootstrap 登录 |
| P3-11 | 重启保数据 | 创建 completed 任务 → restart taskflow → 再查询 |
| P3-12 | 备份恢复 | mongodump → 删任务 → mongorestore --drop → 任务恢复 |
| P3-13 | 双人协作 | owner 创建/分配/审批/关闭 + assignee start/submit（API） |
| P3-13 UI | 浏览器与响应式 | `scripts/web_acceptance_smoke.sh` 运行 `acceptance:browser` + `acceptance:responsive` |

## 成功输出

```
[acceptance] ALL PASSED: P3-10 (cold=0|1), P3-11, P3-12, P3-13
```

## 失败定位

| 失败点 | 排查 |
|--------|------|
| `readyz not healthy` | `docker compose ps`；`docker compose logs taskflow migrate bootstrap` |
| `login ... returned 401` | bootstrap 是否 Exited 0；`docker compose logs bootstrap` |
| `create task returned` 非 201 | JWT/鉴权；`docker compose logs taskflow` |
| `backup archive empty` | mongo 容器是否 healthy |
| `restore did not bring task back` | 检查 archive 路径与 `mongorestore` 输出 |

验收产物（备份文件）默认写入 `backups/acceptance/`。

## CI 集成

`.github/workflows/ci.yml` 在 backend job 中依次执行：

1. `scripts/web_build_smoke.sh`
2. `scripts/validate_production_env.sh`（正反例）
2. `scripts/compose_smoke.sh`
3. `scripts/intranet_acceptance.sh`（增量，非冷启动）
